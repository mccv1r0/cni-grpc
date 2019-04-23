package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mccv1r0/cni-grpc/cnigrpc"
	//proto "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	"github.com/containernetworking/cni/libcni"
)

const (
	EnvCNIPath        = "CNI_PATH"
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"
	EnvCNIArgs        = "CNI_ARGS"
	EnvCNIIfname      = "CNI_IFNAME"

	DefaultNetDir = "/etc/cni/net.d"

	CmdAdd   = "add"
	CmdCheck = "check"
	CmdDel   = "del"
)

// Authentication holds the login/password
type Authentication struct {
	Login    string
	Password string
}

// GetRequestMetadata gets the current request metadata
func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"login":    a.Login,
		"password": a.Password,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security
func (a *Authentication) RequireTransportSecurity() bool {
	return true
}

func main() {
	if len(os.Args) < 4 {
		usage()
		return
	}

	if os.Args[2] == "" {
		fmt.Fprintf(os.Stderr, "  network config name must be supplied as 2ed argument\n")
		os.Exit(1)
	}

	netdir := os.Getenv(EnvNetDir)
	if netdir == "" {
		netdir = DefaultNetDir
	}
	netconf, err := libcni.LoadConfList(netdir, os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "  failed to find config %s\n", os.Args[2])
		return
	}

	confBytes, err := json.Marshal(&netconf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  failed to marshal config bytes\n")
		return
	}

	// //var capArgs string
	//var err error
	capabilityArgs := cnigrpc.CNIcapArgs{}
	capabilityArgsValue := os.Getenv(EnvCapabilityArgs)
	if len(capabilityArgsValue) > 0 {
		println("capabilityArgsValue: ", capabilityArgsValue)
		if err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs); err != nil {
			fmt.Fprintf(os.Stderr, "  failed to unmarshal capabilitiesArgs, err = %v\n", err)
			return
		}
		//data, err := proto.Marshal(&capabilityArgs)
		//println("data: %v", data)
		//if err != nil {
		//	exit(err)
		//}
		//capArgs = string(data)
		// // capArgs = capabilityArgsValue
	}

	args, okArgs := os.LookupEnv(EnvCNIArgs)
	if !okArgs {
		args = ""
	}

	ifName, okIfName := os.LookupEnv(EnvCNIIfname)
	if !okIfName {
		ifName = "eth0"
	}

	netns := os.Args[3]
	if netns == "" {
		fmt.Fprintf(os.Stderr, "  network namespace path must be supplied as 3rd argument\n")
		os.Exit(1)
	}

	conn, err := gRPCtcp()
	//conn, err := gRPCunix()
	defer conn.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  failed to connect to server: %v\n", err)
		return
	}

	switch os.Args[1] {
	case CmdAdd:
		gRPCsendAdd(conn, string(confBytes), netns, ifName, args, capabilityArgs)
	case CmdCheck:
		gRPCsendCheck(conn, string(confBytes), netns, ifName, args, capabilityArgs)
	case CmdDel:
		gRPCsendDel(conn, string(confBytes), netns, ifName, args, capabilityArgs)
	}

	os.Exit(0)
}

func gRPCtcp() (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn

	// Initiate a connection with the server
	//conn, err = grpc.Dial("localhost:7777", grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(&auth))
	conn, err := grpc.Dial("localhost:7777", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
		return nil, err
	}

	return conn, nil
}

func gRPCunix() (*grpc.ClientConn, error) {

	var conn *grpc.ClientConn

	// Initiate a connection with the server
	//conn, err = grpc.Dial("unix:///tmp/grpc.sock", grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(&auth))
	conn, err := grpc.Dial("unix:///tmp/grpc.sock", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
		return nil, err
	}

	return conn, nil
}

func gRPCsendAdd(conn *grpc.ClientConn, conf string, netns string, ifName string, args string, capArgs cnigrpc.CNIcapArgs) error {

	cni := cnigrpc.NewCNIserverClient(conn)

	cniAddMsg := cnigrpc.CNIaddMsg{
		Conf:    conf,
		NetNS:   netns,
		IfName:  ifName,
		CniArgs: args,
		CapArgs: &capArgs,
	}
	resultAdd, err := cni.CNIadd(context.Background(), &cniAddMsg)
	if err != nil {
		log.Fatalf("error when calling CNIadd: %s", err)
		return err
	}
	log.Printf("Response from TCP server: %s", resultAdd.StdOut)

	return nil
}

func gRPCsendCheck(conn *grpc.ClientConn, conf string, netns string, ifName string, args string, capArgs cnigrpc.CNIcapArgs) error {

	cni := cnigrpc.NewCNIserverClient(conn)

	cniCheckMsg := cnigrpc.CNIcheckMsg{
		Conf:    conf,
		NetNS:   netns,
		IfName:  ifName,
		CniArgs: args,
		CapArgs: &capArgs,
	}
	resultCheck, err := cni.CNIcheck(context.Background(), &cniCheckMsg)
	if err != nil {
		log.Fatalf("error when calling CNIcheck: %s", err)
		return err
	}
	log.Printf("Response from TCP server: %s", resultCheck.Error)

	return nil
}

func gRPCsendDel(conn *grpc.ClientConn, conf string, netns string, ifName string, args string, capArgs cnigrpc.CNIcapArgs) error {

	cni := cnigrpc.NewCNIserverClient(conn)

	cniDelMsg := cnigrpc.CNIdelMsg{
		Conf:    conf,
		NetNS:   netns,
		IfName:  ifName,
		CniArgs: args,
		CapArgs: &capArgs,
	}
	resultDel, err := cni.CNIdel(context.Background(), &cniDelMsg)
	if err != nil {
		log.Fatalf("error when calling CNIdel: %s", err)
		return err
	}
	log.Printf("Response from TCP server: %s", resultDel.Error)

	return nil
}

func usage() {
	exe := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "%s: Add, check, or remove network interfaces from a network namespace\n", exe)
	fmt.Fprintf(os.Stderr, "  %s add   <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s check <net> <netns>\n", exe)
	fmt.Fprintf(os.Stderr, "  %s del   <net> <netns>\n", exe)
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
