#!/bin/bash

set -e

current=v1.12.3
next=$1

if [ -z "$next" ]; then
    echo "Error: Please provide the next version as an argument"
    echo "Usage: $0 <next_version>"
    exit 1
fi

echo "Upgrading from $current to $next"

echo "curl -H 'Accept: application/vnd.github.v3.diff' https://github.com/SagerNet/sing-box/compare/${current}...${next}.diff"

diff_content="$(curl -H 'Accept: application/vnd.github.v3.diff' \
    "https://github.com/SagerNet/sing-box/compare/${current}...${next}.diff")"

echo "$diff_content" | git apply \
    --exclude 'core/clients/*' \
    -C1 \
    --directory \
    core

if [ $? -eq 0 ]; then
    SCRIPT_PATH="$0"
    
    sed -i.bak "s/^current=.*/current=$next/" "$SCRIPT_PATH"
    
    echo "Successfully upgraded to $next"
    
    rm -f "${SCRIPT_PATH}.bak"
else
    echo "Upgrade failed"
    exit 1
fi

cd ..
