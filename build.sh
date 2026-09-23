#!/bin/sh
# Builds the Windows executable into dist/.
set -e
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo dev)}
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o dist/COTWSpottingPlus.exe ./cmd/cotwspottingplus
