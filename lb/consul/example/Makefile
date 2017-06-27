PROJECT=github.com/olivere/grpc/lb/consul/example

all: bindir proto server client

.PHONY: bindir
bindir:
	mkdir -p bin

.PHONY: server
server:
	go build -o bin/server $(PROJECT)/server

.PHONY: client
client:
	go build -o bin/client $(PROJECT)/client

.PHONY: proto
proto:
	protoc -I. ./proto/echo/echo.proto --go_out=plugins=grpc:.
