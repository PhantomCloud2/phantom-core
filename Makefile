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

# TAGS=with_gvisor,with_quic,with_utls,with_grpc,with_conntrack,with_clash_api,with_naive_outbound
TAGS=with_quic,with_utls,with_grpc,with_conntrack,with_clash_api
IOS_ADD_TAGS=with_dhcp,with_low_memory,with_purego
DARWIN_ADD_TAGS=with_dhcp
WINDOWS_ADD_TAGS=with_purego
LINUX_ADD_TAGS=with_purego

CRONET_GO_VERSION=$(shell go list -m -f '{{.Version}}' github.com/sagernet/cronet-go 2>/dev/null)

GOMODCACHE=$(shell go env GOMODCACHE)
SING_PACKAGES=$(shell ls $(GOMODCACHE)/github.com/sagernet 2>/dev/null | grep '^sing-' | sort -u)
TRIMPATH_REPLACEMENTS=$(shell echo '$(SING_PACKAGES)' | tr ' ' '\n' | sed 's|^sing-\(.*\)$$|$(GOMODCACHE)/github.com/sagernet/sing-\1=>internal-\1|' | tr '\n' ';' | sed 's|;$$||')

COMMON_FLAGS=-trimpath -ldflags="-w -s -buildid= -checklinkname=0" -buildvcs=false -gcflags=all=-trimpath="$(GOMODCACHE)/github.com/sagernet=>internal-core;$(shell pwd)=>internal-core;$(TRIMPATH_REPLACEMENTS)"
GOBUILD_FLAGS=$(COMMON_FLAGS) -asmflags=all=-trimpath="$(GOMODCACHE)/github.com/sagernet=>internal-core;$(shell pwd)=>internal-core;$(TRIMPATH_REPLACEMENTS)"
GOBUILDLIB=CGO_ENABLED=1 CGO_CFLAGS="-O2 -g0 -pipe" CGO_CXXFLAGS="-O2 -g0 -pipe" CGO_LDFLAGS="-s" go build -buildmode=c-shared -tags $(TAGS) $(GOBUILD_FLAGS)

mod_download:
	go mod download all

lib_install: mod_download
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.11
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.11

headers: mod_download
	go build -buildmode=c-archive -o $(BINDIR)/$(LIBNAME).h ./custom

android: lib_install
	gomobile bind -v -androidapi=21 -javapkg=io.nekohasekai -libname=box -tags=$(TAGS) -trimpath -buildvcs=false $(COMMON_FLAGS) -target=android -o $(BINDIR)/$(LIBNAME).aar github.com/sagernet/sing-box/experimental/libbox

ios-full: lib_install
	gomobile bind -v -target ios,tvos,macos -libname=box -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -buildvcs=false $(COMMON_FLAGS) -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox
	mv $(BINDIR)/$(PRODUCT_NAME).xcframework $(BINDIR)/$(LIBNAME).xcframework 
	cp Libcore.podspec $(BINDIR)/$(LIBNAME).xcframework/

ios: lib_install
	gomobile bind -v -target ios -libname=box -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -buildvcs=false $(COMMON_FLAGS) -o $(BINDIR)/Libcore.xcframework github.com/sagernet/sing-box/experimental/libbox
	cp Info.plist $(BINDIR)/Libcore.xcframework/

.PHONY: build
windows-amd64: TAGS := $(TAGS),$(WINDOWS_ADD_TAGS)
windows-amd64: mod_download
	go run -v "github.com/sagernet/cronet-go/cmd/build-naive@$(CRONET_GO_VERSION)" extract-lib --target windows/amd64 -o $(BINDIR)/
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GOBUILDLIB) -o $(BINDIR)/$(LIBNAME).dll ./custom

linux-amd64: TAGS := $(TAGS),$(LINUX_ADD_TAGS)
linux-amd64: mod_download
	mkdir -p $(BINDIR)/lib
	go run -v "github.com/sagernet/cronet-go/cmd/build-naive@$(CRONET_GO_VERSION)" extract-lib --target linux/amd64 -o $(BINDIR)/lib/
	env GOOS=linux GOARCH=amd64 $(GOBUILDLIB) -o $(BINDIR)/lib/$(LIBNAME).so ./custom

macos-amd64: mod_download
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-O2 -g0 -pipe -mmacosx-version-min=12.0" CGO_LDFLAGS="-mmacosx-version-min=12.0" CGO_ENABLED=1 go build -tags $(TAGS),$(DARWIN_ADD_TAGS) $(GOBUILD_FLAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-amd64.dylib ./custom

macos-arm64: mod_download
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-O2 -g0 -pipe -mmacosx-version-min=12.0" CGO_LDFLAGS="-mmacosx-version-min=12.0" CGO_ENABLED=1 go build -tags $(TAGS),$(DARWIN_ADD_TAGS) $(GOBUILD_FLAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-arm64.dylib ./custom

macos-universal: macos-amd64 macos-arm64 
	lipo -create $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME)-arm64.dylib -output $(BINDIR)/$(LIBNAME).dylib

clean:
	rm $(BINDIR)/*

build_protobuf:
	protoc --go_out=. --go-grpc_out=. corerpc/core.proto

show-trimpath: mod_download
	@echo "GOMODCACHE: $(GOMODCACHE)"
	@echo "SING_PACKAGES: $(SING_PACKAGES)"
	@echo "TRIMPATH_REPLACEMENTS: $(TRIMPATH_REPLACEMENTS)"