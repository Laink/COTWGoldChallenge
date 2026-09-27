#!/bin/sh
# Builds the Windows executable into dist/. The mod class (internal/modclass/mod.swc) comes from build_as3.sh.
set -e
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo dev)}
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o dist/COTWGoldChallenge.exe ./cmd/cotwgoldchallenge
