#!/bin/bash

export GOPATH=${MYLIBSPATH}/golang
export GOROOT=${MYTOOLSPATH}/go-1.21
export PATH=${GOROOT}/bin:$PATH

export CGO_ENABLE=1

go build -o ./nomad-ondemand-scaler .
