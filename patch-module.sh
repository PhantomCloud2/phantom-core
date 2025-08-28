#!/bin/bash

SED=$(command -v gsed || echo sed)

find . -name "*.go" -type f -exec $SED -i 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "Makefile" -type f -exec $SED -i 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.go" -type f -exec $SED -i 's|github\.com/sagernet/sing-box"|github.com/phantomcloude/core"|g' {} +
find . -name "*.mod" -type f -exec $SED -i 's|github\.com/sagernet/sing-box|github.com/phantomcloude/core|g' {} +
