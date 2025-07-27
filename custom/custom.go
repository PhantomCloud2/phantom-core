package main

/*
#include "stdint.h"
*/
import "C"

import (
	"unsafe"

	"github.com/phantomcloude/phantom-core/bridge"
	pb "github.com/phantomcloude/phantom-core/phantomrpc"
	v2 "github.com/phantomcloude/phantom-core/v2"
	"github.com/sagernet/sing-box/log"
)

//export setupOnce
func setupOnce(api unsafe.Pointer) {
	bridge.InitializeDartApi(api)
}

//export setup
func setup(baseDir, workingDir, tempDir *C.char, statusPort C.longlong, debug bool) (CErr *C.char) {
	err := v2.Setup(
		C.GoString(baseDir),
		C.GoString(workingDir),
		C.GoString(tempDir),
		int64(statusPort),
		debug,
	)

	return emptyOrErrorC(err)
}

//export start
func start(config *C.char, disableMemoryLimit bool) (CErr *C.char) {
	_, err := v2.Start(&pb.StartRequest{
		ConfigContent:          C.GoString(config),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

//export stop
func stop() (CErr *C.char) {
	_, err := v2.Stop()
	return emptyOrErrorC(err)
}

//export restart
func restart(config *C.char, disableMemoryLimit bool) (CErr *C.char) {
	_, err := v2.Restart(&pb.StartRequest{
		ConfigContent:          C.GoString(config),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

//export startCommandClient
func startCommandClient(command C.int, port C.longlong) *C.char {
	err := v2.StartCommand(int32(command), int64(port))
	return emptyOrErrorC(err)
}

//export stopCommandClient
func stopCommandClient(command C.int) *C.char {
	err := v2.StopCommand(int32(command))
	return emptyOrErrorC(err)
}

//export selectOutbound
func selectOutbound(groupTag, outboundTag *C.char) (CErr *C.char) {
	_, err := v2.SelectOutbound(&pb.SelectOutboundRequest{
		GroupTag:    C.GoString(groupTag),
		OutboundTag: C.GoString(outboundTag),
	})

	return emptyOrErrorC(err)
}

//export urlTest
func urlTest(groupTag *C.char) (CErr *C.char) {
	_, err := v2.URLTest(&pb.UrlTestRequest{
		GroupTag: C.GoString(groupTag),
	})

	return emptyOrErrorC(err)
}

func emptyOrErrorC(err error) *C.char {
	if err == nil {
		return C.CString("")
	}
	log.Error(err.Error())
	return C.CString(err.Error())
}

func main() {}
