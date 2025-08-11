.ONESHELL:
PRODUCT_NAME=libcore
BASENAME=$(PRODUCT_NAME)
BINDIR=bin
LIBNAME=$(PRODUCT_NAME)

BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --tags || echo "unknown version")
ifeq ($(OS),Windows_NT)
Not available for Windows! use bash in WSL
endif

TAGS=with_gvisor,with_quic,with_utls,with_grpc,with_conntrack,with_clash_api
IOS_ADD_TAGS=with_dhcp,with_low_memory
DARWIN_ADD_TAGS=with_dhcp
GOBUILDLIB=CGO_ENABLED=1 CGO_CFLAGS="-O2 -g0 -pipe" CGO_CXXFLAGS="-O2 -g0 -pipe" CGO_LDFLAGS="-s" go build -trimpath -buildvcs=false -tags $(TAGS) -ldflags="-w -s" -buildmode=c-shared

lib_install:
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.7
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.7

headers:
	go build -buildmode=c-archive -o $(BINDIR)/$(LIBNAME).h ./custom

android: lib_install
	gomobile bind -v -androidapi=21 -javapkg=io.nekohasekai -libname=box -tags=$(TAGS) -trimpath -buildvcs=false -ldflags="-w -s -checklinkname=0" -target=android -o $(BINDIR)/$(LIBNAME).aar github.com/sagernet/sing-box/experimental/libbox

ios-full: lib_install
	gomobile bind -v -target ios,tvos,macos -libname=box -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -buildvcs=false -ldflags="-w -s" -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox
	mv $(BINDIR)/$(PRODUCT_NAME).xcframework $(BINDIR)/$(LIBNAME).xcframework 
	cp Libcore.podspec $(BINDIR)/$(LIBNAME).xcframework/

ios: lib_install
	gomobile bind -v -target ios -libname=box -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -buildvcs=false -ldflags="-w -s" -o $(BINDIR)/Libcore.xcframework github.com/sagernet/sing-box/experimental/libbox
	cp Info.plist $(BINDIR)/Libcore.xcframework/

.PHONY: build
windows-amd64:
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GOBUILDLIB) -o $(BINDIR)/$(LIBNAME).dll ./custom

linux-amd64:
	mkdir -p $(BINDIR)/lib
	env GOOS=linux GOARCH=amd64 $(GOBUILDLIB) -o $(BINDIR)/lib/$(LIBNAME).so ./custom
	mkdir lib

macos-amd64:
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-O2 -g0 -pipe -mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(DARWIN_ADD_TAGS) -ldflags="-w -s" -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-amd64.dylib ./custom
macos-arm64:
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-O2 -g0 -pipe -mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(DARWIN_ADD_TAGS) -ldflags="-w -s" -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-arm64.dylib ./custom
macos-universal: macos-amd64 macos-arm64 
	lipo -create $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME)-arm64.dylib -output $(BINDIR)/$(LIBNAME).dylib

clean:
	rm $(BINDIR)/*

build_protobuf:
	protoc --go_out=. --go-grpc_out=. phantomrpc/phantom.proto
