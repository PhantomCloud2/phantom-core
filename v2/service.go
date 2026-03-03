package v2

import (
	"context"
	"io"
	"os"
	"runtime"
	runtimeDebug "runtime/debug"
	"time"

	B "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

var (
	sWorkingPath          string
	sTempPath             string
	sUserID               int
	sGroupID              int
	statusPropagationPort int64
)

func Setup(basePath, workingPath, tempPath string, statusPort int64, debug bool) error {
	statusPropagationPort = statusPort

	setupOpts := &libbox.SetupOptions{
		BasePath:    basePath,
		WorkingPath: workingPath,
		TempPath:    tempPath,
		Debug:       debug,
	}
	// Windows does not support Unix domain sockets; use TCP instead.
	if runtime.GOOS == "windows" {
		setupOpts.CommandServerListenPort = 8964
	}
	err := libbox.Setup(setupOpts)
	if err != nil {
		return err
	}

	sWorkingPath = workingPath

	err = os.Chdir(sWorkingPath)
	if err != nil {
		return err
	}

	sTempPath = tempPath
	sUserID = os.Getuid()
	sGroupID = os.Getgid()

	var defaultWriter io.Writer
	if !debug {
		defaultWriter = io.Discard
	}

	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    true,
			// Options: option.LogOptions{
			// 	Disabled: false,
			// 	Level:    "trace",
			// 	Output:   "stdout",
			// },
		})
	coreLogFactory = factory

	return err
}

type BoxService struct {
	cancel context.CancelFunc
	box    *B.Box
}

func (s *BoxService) Start() error { return s.box.Start() }
func (s *BoxService) Close() error { s.cancel(); return s.box.Close() }

func createBaseContext() context.Context {
	return B.Context(
		filemanager.WithDefault(
			context.Background(),
			sWorkingPath, sTempPath, sUserID, sGroupID,
		),
		include.InboundRegistry(),
		include.OutboundRegistry(),
		include.EndpointRegistry(),
		include.DNSTransportRegistry(),
		include.ServiceRegistry(),
	)
}

func NewService(ctx context.Context, options option.Options) (*BoxService, error) {
	runtimeDebug.FreeOSMemory()

	ctx, cancel := context.WithCancel(ctx)
	urlTestHistoryStorage := urltest.NewHistoryStorage()
	ctx = service.ContextWithPtr(ctx, urlTestHistoryStorage)

	instance, err := B.New(B.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		cancel()
		return nil, E.Cause(err, "create service")
	}

	runtimeDebug.FreeOSMemory()

	return &BoxService{cancel: cancel, box: instance}, nil
}

func readOptions(ctx context.Context, configContent string) (option.Options, error) {
	return json.UnmarshalExtendedContext[option.Options](
		ctx,
		[]byte(configContent),
	)
}
