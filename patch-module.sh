#!/bin/bash

set -e

if command -v gsed &> /dev/null; then
    SED_CMD="gsed -i"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew install gnu-sed
    SED_CMD="gsed -i"
else
    SED_CMD="sed -i"
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
    if ! command -v protoc &> /dev/null; then
        brew install protobuf
    fi
else
    if ! command -v protoc &> /dev/null; then
        sudo apt install -y protobuf-compiler
    fi
fi

find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.proto" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "Makefile" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box"|github.com/phantomcloude/core"|g' {} +
find . -name "*.mod" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box|github.com/phantomcloude/core|g' {} +

(cd core && make proto_install && make fmt_install && make proto)
