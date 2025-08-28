#!/bin/bash

if command -v gsed &> /dev/null; then
    SED_CMD="gsed"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    SED_CMD="sed -i ''"
else
    SED_CMD="sed -i"
fi

find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "Makefile" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.go" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box"|github.com/phantomcloude/core"|g' {} +
find . -name "*.mod" -type f -exec $SED_CMD 's|github\.com/sagernet/sing-box|github.com/phantomcloude/core|g' {} +
