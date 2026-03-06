package v2

import (
	"errors"

	pb "github.com/phantomcloude/phantom-core/phantomrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var oldCommandServer *libbox.CommandServer

type CommandServerHandler struct {
	logger log.Logger
}

func (csh *CommandServerHandler) ServiceReload() error {
	csh.logger.Trace("Reloading service")

	if lastConfigContent == "" {
		return errors.New("no config content to reload")
	}

	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")

	err := oldCommandServer.StartOrReloadService(lastConfigContent, &libbox.OverrideOptions{})
	if err != nil {
		SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
		return err
	}

	SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")
	return nil
}

func (csh *CommandServerHandler) GetSystemProxyStatus() (*libbox.SystemProxyStatus, error) {
	csh.logger.Trace("Getting system proxy status")
	return &libbox.SystemProxyStatus{Available: true, Enabled: false}, nil
}

func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
	csh.logger.Trace("Setting system proxy status, enabled? ", isEnabled)
	return csh.ServiceReload()
}

func (csh *CommandServerHandler) ServiceStop() error {
	csh.logger.Trace("Stopping service")
	_, err := Stop()
	return err
}

func (csh *CommandServerHandler) WriteDebugMessage(message string) {
	csh.logger.Trace(message)
}

func startCommandServer() error {
	if oldCommandServer != nil {
		return nil // already running
	}
	logger := coreLogFactory.NewLogger("[Command Server Handler]")
	logger.Trace("Starting command server")
	// Pass nil as platformInterface (Hiddify-style):
	// sing-box will use the OS-native interface monitor and UpdateInterfaces will
	// call interfaceFinder.Update() directly, never touching linkFlags.
	server, err := libbox.NewCommandServer(&CommandServerHandler{logger: logger}, nil)
	if err != nil {
		return err
	}
	oldCommandServer = server
	if err := oldCommandServer.Start(); err != nil {
		oldCommandServer = nil
		return err
	}
	return nil
}
