package libbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mDNS "github.com/miekg/dns"
	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/http3"
)

// Compile-time interface checks
var (
	_ HTTPClient  = (*customHTTPClient)(nil)
	_ HTTPRequest = (*customHTTPRequest)(nil)
	_ HTTPResponse = (*customHTTPResponse)(nil)
)

// customHTTPClient implements HTTPClient with HTTP/3 and ECH support.
// Use NewCustomHTTPClient() to create an instance; do not use the original
// NewHTTPClient() for anti-censorship subscription fetching.
type customHTTPClient struct {
	timeout time.Duration
}

// NewCustomHTTPClient creates a new HTTPClient that supports HTTP/3 and ECH.
// gomobile will automatically export this method to Swift and Java/Kotlin as
// LibboxNewCustomHTTPClient() / Libbox.newCustomHTTPClient().
func NewCustomHTTPClient() HTTPClient {
	return &customHTTPClient{
		timeout: 10 * time.Second,
	}
}

func (c *customHTTPClient) RestrictedTLS()        {}
func (c *customHTTPClient) ModernTLS()            {}
func (c *customHTTPClient) PinnedTLS12()          {}
func (c *customHTTPClient) PinnedSHA256(_ string) {}
func (c *customHTTPClient) TrySocks5(_ int32)     {}
func (c *customHTTPClient) KeepAlive()            {}
func (c *customHTTPClient) Close()                {}

func (c *customHTTPClient) NewRequest() HTTPRequest {
	return &customHTTPRequest{
		client: c,
		method: "GET",
		header: make(http.Header),
	}
}

type customHTTPRequest struct {
	client *customHTTPClient
	url    string
	method string
	header http.Header
	body   []byte
}

func (r *customHTTPRequest) SetURL(link string) error {
	r.url = link
	return nil
}

func (r *customHTTPRequest) SetMethod(method string) {
	r.method = method
}

func (r *customHTTPRequest) SetHeader(key, value string) {
	r.header.Set(key, value)
}

func (r *customHTTPRequest) SetContent(content []byte) {
	r.body = content
}

func (r *customHTTPRequest) SetContentString(content string) {
	r.body = []byte(content)
}

func (r *customHTTPRequest) RandomUserAgent() {
	r.header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
}

func (r *customHTTPRequest) SetUserAgent(userAgent string) {
	r.header.Set("User-Agent", userAgent)
}

type httpResponseJSON struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

func (r *customHTTPRequest) Execute() (HTTPResponse, error) {
	parsedURL, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	domain := parsedURL.Hostname()
	echConfig, _ := fetchECHConfig(domain)

	ctx, cancel := context.WithTimeout(context.Background(), r.client.timeout)
	defer cancel()

	// Helper: copy request headers
	copyHeaders := func(req *http.Request) {
		for k, vv := range r.header {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
	}

	// Helper: build a fresh TLS config for each attempt
	buildTLS := func() *tls.Config {
		cfg := &tls.Config{
			ServerName: domain,
		}
		if len(echConfig) > 0 {
			cfg.EncryptedClientHelloConfigList = echConfig
		}
		return cfg
	}

	type requestFunc func() (*http.Response, error)
	var funcs []requestFunc

	// Attempt 1: TCP + TLS (with ECH when available)
	funcs = append(funcs, func() (*http.Response, error) {
		var reqBody io.Reader
		if len(r.body) > 0 {
			reqBody = bytes.NewReader(r.body)
		}
		req, err := http.NewRequestWithContext(ctx, r.method, r.url, reqBody)
		if err != nil {
			return nil, err
		}
		copyHeaders(req)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: buildTLS(),
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
			},
		}
		return client.Do(req)
	})

	// Attempt 2: HTTP/3 over QUIC (with ECH when available)
	// Note: do NOT defer tr.CloseIdleConnections() inside the goroutine —
	// the goroutine returns a live response; close only on failure.
	funcs = append(funcs, func() (*http.Response, error) {
		var reqBody io.Reader
		if len(r.body) > 0 {
			reqBody = bytes.NewReader(r.body)
		}
		req, err := http.NewRequestWithContext(ctx, r.method, r.url, reqBody)
		if err != nil {
			return nil, err
		}
		copyHeaders(req)
		tr := &http3.Transport{
			TLSClientConfig: buildTLS(),
			QUICConfig: &quic.Config{
				MaxIdleTimeout: 5 * time.Second,
			},
		}
		resp, err := (&http.Client{Transport: tr}).Do(req)
		if err != nil {
			tr.CloseIdleConnections()
			return nil, err
		}
		return resp, nil
	})

	// Race: pick the first successful response
	// No wrapper struct needed — channel carries *http.Response directly.
	successCh := make(chan *http.Response, len(funcs))
	var mu sync.Mutex
	var finalErr error
	var successCount int32
	var failedCount int32

	for i, f := range funcs {
		go func(index int, run requestFunc) {
			resp, err := run()
			if err != nil {
				mu.Lock()
				finalErr = errors.Join(finalErr, fmt.Errorf("task %d: %w", index, err))
				mu.Unlock()
				if atomic.AddInt32(&failedCount, 1) >= int32(len(funcs)) {
					cancel()
				}
				return
			}
			if atomic.CompareAndSwapInt32(&successCount, 0, 1) {
				successCh <- resp
			} else {
				resp.Body.Close() // loser — discard
			}
		}(i, f)
	}

	select {
	case resp := <-successCh:
		// Validate HTTP status before handing response to caller
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("http %d: %s", resp.StatusCode, resp.Status)
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read body error: %w", err)
		}
		jsonBytes, err := json.Marshal(httpResponseJSON{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       string(bodyBytes),
		})
		if err != nil {
			return nil, err
		}
		return &customHTTPResponse{content: jsonBytes}, nil

	case <-ctx.Done():
		mu.Lock()
		defer mu.Unlock()
		if finalErr != nil {
			return nil, finalErr
		}
		return nil, ctx.Err()
	}
}

type customHTTPResponse struct {
	content []byte
}

func (h *customHTTPResponse) GetContent() (*StringBox, error) {
	return wrapString(string(h.content)), nil
}

func (h *customHTTPResponse) WriteTo(path string) error {
	return os.WriteFile(path, h.content, 0o644)
}

func (h *customHTTPResponse) WriteToWithProgress(path string, handler HTTPResponseWriteToProgressHandler) error {
	err := os.WriteFile(path, h.content, 0o644)
	if err == nil {
		handler.Update(int64(len(h.content)), int64(len(h.content)))
	}
	return err
}

// ─── DNS / ECH Config Lookup ──────────────────────────────────────────────────

type echDNSResult struct {
	ech []byte
	err error
}

// fetchECHConfig resolves the HTTPS DNS record for the given domain and extracts
// the ECH (Encrypted Client Hello) config list. It races 4 UDP resolvers and
// one DoH endpoint and returns the first successful result.
func fetchECHConfig(domain string) ([]byte, error) {
	msg := new(mDNS.Msg)
	msg.SetQuestion(mDNS.Fqdn(domain), mDNS.TypeHTTPS)
	msg.RecursionDesired = true

	udpResolvers := []string{
		"8.8.8.8:53",
		"114.114.114.114:53",
		"119.29.29.29:53",
		"180.184.2.2:53",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	totalTasks := len(udpResolvers) + 1 // 4 UDP + 1 DoH
	ch := make(chan echDNSResult, totalTasks)

	// UDP resolvers (parallel)
	for _, server := range udpResolvers {
		go func(srv string) {
			c := new(mDNS.Client)
			c.Net = "udp"
			c.Timeout = 1500 * time.Millisecond
			resp, _, err := c.ExchangeContext(ctx, msg, srv)
			if err != nil {
				ch <- echDNSResult{err: err}
				return
			}
			ch <- parseECHFromDNS(resp)
		}(server)
	}

	// Alidns DoH — directly dials 223.5.5.5 to avoid bootstrap DNS loop
	go func() {
		resp, err := queryDoHAlidns(ctx, msg)
		if err != nil {
			ch <- echDNSResult{err: err}
			return
		}
		ch <- parseECHFromDNS(resp)
	}()

	var lastErr error
	for i := 0; i < totalTasks; i++ {
		res := <-ch
		if res.ech != nil {
			cancel() // let remaining goroutines exit early
			return res.ech, nil
		}
		if res.err != nil {
			lastErr = res.err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no ECH config found for %s", domain)
}

func parseECHFromDNS(resp *mDNS.Msg) echDNSResult {
	if resp.Rcode != mDNS.RcodeSuccess {
		return echDNSResult{err: fmt.Errorf("rcode %d", resp.Rcode)}
	}
	for _, rr := range resp.Answer {
		httpsRecord, ok := rr.(*mDNS.HTTPS)
		if !ok {
			continue
		}
		for _, val := range httpsRecord.Value {
			if val.Key().String() == "ech" {
				data, err := base64.StdEncoding.DecodeString(val.String())
				if err == nil {
					return echDNSResult{ech: data}
				}
			}
		}
	}
	return echDNSResult{err: fmt.Errorf("no ech in answer")}
}

// queryDoHAlidns sends a DNS-over-HTTPS query to 223.5.5.5 (Alidns).
// It dials directly to the IP and uses "dns.alidns.com" as the TLS SNI to
// verify the server certificate without triggering a recursive DNS lookup.
func queryDoHAlidns(ctx context.Context, msg *mDNS.Msg) (*mDNS.Msg, error) {
	packed, err := msg.Pack()
	if err != nil {
		return nil, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(packed)
	reqURL := fmt.Sprintf("https://223.5.5.5/dns-query?dns=%s", encoded)

	// Use an independent context with its own deadline so that a cancellation
	// of the outer fetchECHConfig context (triggered by a UDP winner) does not
	// abort an in-flight DoH response mid-stream.  1500 ms hard cap is enough.
	dohCtx, dohCancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer dohCancel()
	// Still respect explicit parent cancellation (e.g. overall app shutdown).
	go func() {
		select {
		case <-ctx.Done():
			dohCancel()
		case <-dohCtx.Done():
		}
	}()

	req, err := http.NewRequestWithContext(dohCtx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-message")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: "dns.alidns.com",
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respMsg := new(mDNS.Msg)
	if err = respMsg.Unpack(body); err != nil {
		return nil, err
	}
	return respMsg, nil
}
