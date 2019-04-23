package cnigrpc

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	//proto "github.com/golang/protobuf/proto"
	//jsonpb "github.com/golang/protobuf/jsonpb"

	"github.com/containernetworking/cni/libcni"
)

const (
	EnvCNIPath = "CNI_PATH"
	EnvNetDir  = "NETCONFPATH"

	DefaultNetDir = "/etc/cni/net.d"
)

// Server represents the gRPC server
type CNIServer struct {
}

func (s *CNIServer) CNIconfig(ctx context.Context, confPath *ConfPath) (*CNIerror, error) {

	cniError := CNIerror{}

	if confPath != nil {
		if confPath.NetDir != "" {
			log.Printf("Receive message NetDir: %s", confPath.NetDir)
		}
		if confPath.NetConf != "" {
			log.Printf("Receive message NetConf: %s", confPath.NetConf)
		}
	}

	log.Printf("Response from server: %v", cniError)
	return &cniError, nil
}

// CNIadd generates result to a CNIaddMsg
func (s *CNIServer) CNIadd(ctx context.Context, in *CNIaddMsg) (*ADDresult, error) {

	log.Printf("Receive message Conf file: %s", in.Conf)
	log.Printf("Receive message ContainerID: %s", in.ContainerID)
	log.Printf("Receive message NetNS: %s", in.NetNS)
	log.Printf("Receive message IfName: %s", in.IfName)
	log.Printf("Receive message CniArgs: %s", in.CniArgs)
	log.Printf("Receive message CniCapArgs: %s", in.CapArgs)

	netconf, rt, cninet, err := cniCommon(in.Conf, in.NetNS, in.IfName, in.CniArgs, in.CapArgs)
	if err != nil {
		return nil, err
	}

	result, err := cninet.AddNetworkList(context.TODO(), netconf, rt)
	if err != nil {
		return nil, err
	}

	cniResult := ADDresult{}
	if result != nil {
		cniResult.StdOut = result.String()
	}

	log.Printf("Response from server: %s", cniResult.StdOut)
	return &cniResult, nil
}

// CNIcheck generates result to a CNIcheckMsg
func (s *CNIServer) CNIcheck(ctx context.Context, in *CNIcheckMsg) (*CHECKresult, error) {

	log.Printf("Receive message Conf file: %s", in.Conf)
	log.Printf("Receive message ContainerID: %s", in.ContainerID)
	log.Printf("Receive message NetNS: %s", in.NetNS)
	log.Printf("Receive message IfName: %s", in.IfName)
	log.Printf("Receive message CniArgs: %s", in.CniArgs)
	log.Printf("Receive message CniCapArgs: %s", in.CapArgs)

	netconf, rt, cninet, err := cniCommon(in.Conf, in.NetNS, in.IfName, in.CniArgs, in.CapArgs)
	if err != nil {
		return nil, err
	}

	err = cninet.CheckNetworkList(context.TODO(), netconf, rt)
	if err != nil {
		return nil, err
	}

	cniResult := CHECKresult{
		Error: "",
	}

	log.Printf("Response from server: %s", cniResult.Error)
	return &cniResult, nil
}

// CNIdel generates result to a CNIdelMsg
func (s *CNIServer) CNIdel(ctx context.Context, in *CNIdelMsg) (*DELresult, error) {

	log.Printf("Receive message Conf file: %s", in.Conf)
	log.Printf("Receive message ContainerID: %s", in.ContainerID)
	log.Printf("Receive message NetNS: %s", in.NetNS)
	log.Printf("Receive message IfName: %s", in.IfName)
	log.Printf("Receive message CniArgs: %s", in.CniArgs)
	log.Printf("Receive message CniCapArgs: %s", in.CapArgs)

	netconf, rt, cninet, err := cniCommon(in.Conf, in.NetNS, in.IfName, in.CniArgs, in.CapArgs)
	if err != nil {
		return nil, err
	}

	err = cninet.DelNetworkList(context.TODO(), netconf, rt)
	if err != nil {
		return nil, err
	}

	cniResult := DELresult{
		Error: "",
	}

	log.Printf("Response from server: %s", cniResult.StdOut)
	return &cniResult, nil
}

func parseArgs(args string) ([][2]string, error) {
	var result [][2]string

	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result = append(result, [2]string{kv[0], kv[1]})
	}

	return result, nil
}

func cniCommon(conf string, netns string, ifName string, args string, capabilityArgsValue *CNIcapArgs) (*libcni.NetworkConfigList, *libcni.RuntimeConf, *libcni.CNIConfig, error) {

	var err error

	log.Printf("cniCommon Called")

	netconf := libcni.NetworkConfigList{}
	if err = json.Unmarshal([]byte(conf), &netconf); err != nil {
		return nil, nil, nil, err
	}

	// Example of how to walk a received protobuf message
	portMappings := capabilityArgsValue.GetPortMappings()
	log.Printf(" portMappings = %v", portMappings)
	for _, portMap := range portMappings {
		log.Printf(" portMap = %v", portMap)
	}

	var capabilityArgs map[string]interface{}
	data, err := json.Marshal(capabilityArgsValue)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := json.Unmarshal(data, &capabilityArgs); err != nil {
		return nil, nil, nil, err
	}

	var cniArgs [][2]string
	if args != "" {
		if len(args) > 0 {
			cniArgs, err = parseArgs(args)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	if ifName == "" {
		ifName = "eth0"
	}

	if netns != "" {
		netns, err = filepath.Abs(netns)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		return nil, nil, nil, fmt.Errorf("network namespace is required")
	}

	// Generate the containerid by hashing the netns path
	s := sha512.Sum512([]byte(netns))
	containerID := fmt.Sprintf("cnitool-%x", s[:10])

	cninet := libcni.NewCNIConfig(filepath.SplitList(os.Getenv(EnvCNIPath)), nil)

	rt := &libcni.RuntimeConf{
		ContainerID:    containerID,
		NetNS:          netns,
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
		//CapabilityArgs: portMappings,
	}

	return &netconf, rt, cninet, nil
}
