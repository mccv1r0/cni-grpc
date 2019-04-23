SERVER_OUT := "bin/server"
CLIENT_OUT := "bin/client"
CNIGRPC_OUT := "cnigrpc/cnigrpc.pb.go"
PKG := "github.com/mccv1r0/cni-grpc"
SERVER_PKG_BUILD := "${PKG}/server"
CLIENT_PKG_BUILD := "${PKG}/client"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)

.PHONY: all cnigrpc server client

all: server client

cnigrpc/cnigrpc.pb.go: cnigrpc/cnigrpc.proto
	protoc -I cnigrpc/ \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:cnigrpc \
		cnigrpc/cnigrpc.proto

cnigrpc: cnigrpc/cnigrpc.pb.go 

dep: 
	go get -v -d ./...

server: dep cnigrpc 
	go build -i -v -o $(SERVER_OUT) $(SERVER_PKG_BUILD)

client: dep cnigrpc 
	go build -i -v -o $(CLIENT_OUT) $(CLIENT_PKG_BUILD)

clean: 
	rm $(SERVER_OUT) $(CLIENT_OUT) $(CNIGRPC_OUT)
