package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"internal-libcore/bridge"
	pb "internal-libcore/corerpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var (
	Box               *BoxService
	lastConfigContent string
	coreLogFactory    log.Factory
	useFlutterBridge  bool = true
)

func StopAndAlert(msgType pb.MessageType, message string) {
	SetCoreStatus(pb.CoreState_STOPPED, msgType, message)

	if Box != nil {
		Box.Close()
		Box = nil
	} else if oldCommandServer != nil {
		oldCommandServer.CloseService()
	}

	if oldCommandServer != nil {
		oldCommandServer.Close()
		oldCommandServer = nil
	}

	if useFlutterBridge {
		alert := msgType.String()
		msg, _ := json.Marshal(
			StatusMessage{Status: convert2OldState(CoreState), Alert: &alert, Message: &message},
		)
		bridge.SendStringToPort(statusPropagationPort, string(msg))
	}
}

func DeferPanicToError(name string, err func(error)) {
	if r := recover(); r != nil {
		s := fmt.Errorf("%s panic: %s\n%s", name, r, string(debug.Stack()))
		err(s)
	}
}

func Start(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	defer DeferPanicToError("start", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	Log(pb.LogLevel_INFO, pb.LogType_CORE, "Starting")

	if CoreState != pb.CoreState_STOPPED {
		Log(pb.LogLevel_INFO, pb.LogType_CORE, "Starting0000")
		Stop()
		// return &pb.CoreInfoResponse{
		// 	CoreState:   CoreState,
		// 	MessageType: pb.MessageType_INSTANCE_NOT_STOPPED,
		// }, fmt.Errorf("instance not stopped")
	}

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Core")
	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")
	libbox.SetMemoryLimit(!in.GetDisableMemoryLimit())
	resp, err := StartService(in)

	return resp, err
}

func StartService(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Core Service")

	content := in.GetConfigContent()

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Parsing Config")

	ctx := createBaseContext()
	parsedContent, err := readOptions(ctx, content)

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Parsed")

	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(
			pb.CoreState_STOPPED,
			pb.MessageType_ERROR_PARSING_CONFIG,
			err.Error(),
		)
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())

		return resp, err
	}

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Saving config")

	if in.GetEnableOldCommandServer() {
		Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Command Server")

		err = startCommandServer()
		if err != nil {
			Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
			resp := SetCoreStatus(
				pb.CoreState_STOPPED,
				pb.MessageType_START_COMMAND_SERVER,
				err.Error(),
			)
			StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())

			return resp, err
		}

		// CommandServer manages the box lifecycle via daemon.StartedService,
		// so that gRPC subscribers (Status, Groups) can receive live data.
		Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Service via CommandServer")

		if in.GetDelayStart() {
			<-time.After(250 * time.Millisecond)
		}

		err = oldCommandServer.StartOrReloadService(content, &libbox.OverrideOptions{})
		if err != nil {
			Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
			resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
			StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
			return resp, err
		}

		lastConfigContent = content
		resp := SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")
		return resp, nil
	}

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Stating Service ")

	instance, err := NewService(ctx, parsedContent)
	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_CREATE_SERVICE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
		return resp, err
	}

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Service.. started")

	if in.GetDelayStart() {
		<-time.After(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
		return resp, err
	}

	Box = instance

	resp := SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")

	return resp, nil
}

func Stop() (*pb.CoreInfoResponse, error) {
	defer DeferPanicToError("stop", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	if CoreState != pb.CoreState_STARTED {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, "Core is not started")

		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, errors.New("instance not started")
	}

	if Box == nil && oldCommandServer == nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, errors.New("instance not found")
	}

	SetCoreStatus(pb.CoreState_STOPPING, pb.MessageType_EMPTY, "")

	if oldCommandServer != nil {
		err := oldCommandServer.CloseService()
		if err != nil {
			return &pb.CoreInfoResponse{
				CoreState:   CoreState,
				MessageType: pb.MessageType_UNEXPECTED_ERROR,
				Message:     "Error while stopping the service",
			}, errors.New("error while stopping the service")
		}
		oldCommandServer.Close()
		oldCommandServer = nil
	} else {
		err := Box.Close()
		if err != nil {
			return &pb.CoreInfoResponse{
				CoreState:   CoreState,
				MessageType: pb.MessageType_UNEXPECTED_ERROR,
				Message:     "Error while stopping the service",
			}, errors.New("error while stopping the service")
		}
		Box = nil
	}

	resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_EMPTY, "")

	return resp, nil
}

func Restart(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	defer DeferPanicToError("restart", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	log.Debug("[Service] Restarting")

	if CoreState != pb.CoreState_STARTED {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, errors.New("instance not started")
	}

	if Box == nil && oldCommandServer == nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, errors.New("instance not found")
	}

	resp, err := Stop()
	if err != nil {
		return resp, err
	}

	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")
	<-time.After(250 * time.Millisecond)

	libbox.SetMemoryLimit(!in.GetDisableMemoryLimit())
	resp, gErr := StartService(in)

	return resp, gErr
}
