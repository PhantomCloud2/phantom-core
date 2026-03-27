package tlspreset

import (
	"crypto/x509"
	"embed"
	"path"
	"strings"
)

//go:embed certs/*.pem
var certificateFS embed.FS

// presetCertPools 在包初始化时构建，key 为规范化的 server_name（文件名去掉 .pem），value 为已构建好的 CertPool。
// 无需手动维护映射表：把证书 PEM 放到 certs/<server_name>.pem 即可自动生效。
var presetCertPools = map[string]*x509.CertPool{}

func init() {
	entries, err := certificateFS.ReadDir("certs")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pem") {
			continue
		}
		serverName := strings.TrimSuffix(entry.Name(), ".pem")
		pemBytes, err := certificateFS.ReadFile(path.Join("certs", entry.Name()))
		if err != nil {
			continue
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			continue
		}
		presetCertPools[serverName] = pool
	}
}

func normalizeServerName(serverName string) string {
	name := strings.ToLower(strings.TrimSpace(serverName))
	return strings.TrimSuffix(name, ".")
}

// CertPoolForServerName 返回与 serverName 对应的预内置 CA 证书池。
// 文件名（去掉 .pem 后缀）即为 server_name，大小写不敏感。
// 如需新增域名，只需将对应的 PEM 证书文件放到 certs/<server_name>.pem 即可，无需修改代码。
func CertPoolForServerName(serverName string) (*x509.CertPool, bool) {
	pool, exists := presetCertPools[normalizeServerName(serverName)]
	return pool, exists
}
