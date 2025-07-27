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

	err := libbox.Setup(&libbox.SetupOptions{
		BasePath:    basePath,
		WorkingPath: workingPath,
		TempPath:    tempPath,
		// use tcp conn when running on windows or tvos
		IsTVOS: runtime.GOOS == "windows",
	})
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

func NewService(ctx context.Context, options option.Options) (*libbox.BoxService, error) {
	runtimeDebug.FreeOSMemory()

	ctx, cancel := context.WithCancel(ctx)
	ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
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

	service := libbox.NewBoxService(
		ctx,
		cancel,
		instance,
		urlTestHistoryStorage,
	)

	return &service, nil
}

func readOptions(ctx context.Context, configContent string) (option.Options, error) {
	return json.UnmarshalExtendedContext[option.Options](
		ctx,
		[]byte(configContent),
	)
}
