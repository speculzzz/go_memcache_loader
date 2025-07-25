#!/bin/bash

mkdir -p tools/go
wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz -O tools/go.tar.gz
tar -C tools/go -xzf tools/go.tar.gz --strip-components=1
rm -f tools/go.tar.gz
rm -fr tools/go/test

mkdir -p tools/protoc
PB_REL="https://github.com/protocolbuffers/protobuf/releases"
curl -LO $PB_REL/download/v31.1/protoc-31.1-linux-x86_64.zip
unzip protoc-31.1-linux-x86_64.zip -d tools/protoc
rm -f protoc-31.1-linux-x86_64.zip

GOBIN=$PWD/tools go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
