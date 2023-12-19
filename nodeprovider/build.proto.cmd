@echo off
setlocal

set PATH=%MYTOOLSPATH%/protoc-24.x;%MYLIBSPATH%/golang/bin;%PATH%

set PKGPATH=.\karpenterprovidergrpc

mkdir %PKGPATH%
protoc --go_out=%PKGPATH% --go_opt=paths=source_relative --go-grpc_out=%PKGPATH% --go-grpc_opt=paths=source_relative ./karpenter.proto