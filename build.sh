#!/bin/sh
# Builds the Windows executable into dist/. The mod class (internal/modclass/mod.swc) comes from build_as3.sh.
set -e
# Local builds: the latest tag, which may be on the release branch only, and the commit.
VERSION=$1
if [ -z "$VERSION" ]; then
	TAG=$(git tag --sort=-v:refname 2>/dev/null | head -n 1)
	VERSION="${TAG:-dev}-dev-$(git rev-parse --short HEAD 2>/dev/null || echo 0)"
fi
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o dist/COTWGoldChallenge.exe ./cmd/cotwgoldchallenge
