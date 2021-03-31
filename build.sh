#!/bin/bash

set -o errexit
set -x

function builder() {
    GOOS=${2}
    GOARCH=${1}
    env GOARCH=${GOARCH} GOOS=${GOOS} go build -a -ldflags '-s -w -extldflags "-static"' -o bin/dssh-${GOOS}-${GOARCH} main.go
    upx bin/dssh-${GOOS}-${GOARCH}
}

# MacOS
builder amd64 darwin

# Linux
builder amd64 linux

# Windows
builder amd64 windows
