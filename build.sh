#!/bin/bash

set -eux

GO_VERSION=1.7

# Builds the binary in a go container, and drops the Linux-compatible binary in $PWD/bin
docker run --rm \
	-v "$PWD":/go/src/github.com/clockworksoul/smudge \
	-v "$PWD/tmp":/go/bin \
	-w /go/bin \
	-e "CGO_ENABLED=0" \
	-e "GOOS=linux" \
	golang:${GO_VERSION} \
	go build -a -installsuffix cgo -v github.com/clockworksoul/smudge/smudge

docker build -t clockworksoul/smudge:latest .

sudo rm -R tmp
