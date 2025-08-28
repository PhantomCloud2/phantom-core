#!/bin/bash

set -e

if command -v gsed &> /dev/null; then
    SED_CMD="gsed"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew install -y gnu-sed
    SED_CMD="gsed"
else
    SED_CMD="sed -i"
fi

find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "Makefile" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box"|github.com/phantomcloude/core"|g' {} +
find . -name "*.mod" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box|github.com/phantomcloude/core|g' {} +
