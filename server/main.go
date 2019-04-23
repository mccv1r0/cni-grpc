package main

import (
	"fmt"
	"log"
	"net"
	"syscall"

	"github.com/mccv1r0/cni-grpc/cnigrpc"
	"google.golang.org/grpc"
)

const (
	unixSocketPath = "/tmp/grpc.sock"
)

func startGRPCunixServer(address string) error {
	// create a listener on unix socket
	syscall.Unlink(unixSocketPath)
	lis, err := net.Listen("unix", unixSocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// create a CNI server instance
	cni := cnigrpc.CNIServer{}

	// create a gRPC server object
	//grpcCNIServer := grpc.NewServer(opts...)
	grpcCNIServer := grpc.NewServer()

	// attach the CNI service to the server
	cnigrpc.RegisterCNIserverServer(grpcCNIServer, &cni)

	// start the server
	log.Printf("starting CNI unix socket gRPC server on %s", unixSocketPath)
	if err := grpcCNIServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %s", err)
	}

	return nil
}

func startGRPCtcpServer(address string) error {
	// create a listener on TCP port
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// create a CNI server instance
	cni := cnigrpc.CNIServer{}

	// create a gRPC server object
	//grpcCNIServer := grpc.NewServer(opts...)
	grpcCNIServer := grpc.NewServer()

	// attach the CNI service to the server
	cnigrpc.RegisterCNIserverServer(grpcCNIServer, &cni)

	// start the server
	log.Printf("starting CNI HTTP/2 gRPC server on %s", address)
	if err := grpcCNIServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %s", err)
	}

	return nil
}

// main start a gRPC server and waits for connection
func main() {
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", 7777)

	// fire the gRPC unix socket server in a goroutine
	go func() {
		err := startGRPCunixServer(grpcAddress)
		if err != nil {
			log.Fatalf("failed to start unix socket gRPC server: %s", err)
		}
	}()

	// fire the gRPC tcp server in a goroutine
	go func() {
		err := startGRPCtcpServer(grpcAddress)
		if err != nil {
			log.Fatalf("failed to start tcp gRPC server: %s", err)
		}
	}()

	// infinite loop
	log.Printf("Entering infinite loop")
	select {}
}
