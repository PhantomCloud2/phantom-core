#!/bin/bash

find . -name "*.go" -type f -exec sed -i 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "Makefile" -type f -exec sed -i 's|github\.com/sagernet/sing-box/|github.com/phantomcloude/core/|g' {} +
find . -name "*.go" -type f -exec sed -i 's|github\.com/sagernet/sing-box"|github.com/phantomcloude/core"|g' {} +
find . -name "*.mod" -type f -exec sed -i 's|github\.com/sagernet/sing-box|github.com/phantomcloude/core|g' {} +
