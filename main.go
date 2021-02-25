package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

const defaultConfig string = "bazaar.yml"

func main() {
	// TODO: rest of project

	node, err := CreateNodeFromConfigPath(defaultConfig)
	if err != nil {
		log.Fatalf("Error creating node from config at %s: %s", defaultConfig, err)
		return
	}
	log.Printf("Loaded config for bazaar node. Node ID: %d\n", node.config.NodeID)

	// Register inbound RPC calls on node's port
	rpcServer := rpc.NewServer()
	rpcServer.Register(node)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", node.config.NodePort))
	if err != nil {
		log.Fatalf("Error listening on port %d: %s", node.config.NodePort, err)
		return
	}

	// Finally, listen on rpc.
	log.Printf("Listening on port %d for incoming RPC connections...", node.config.NodePort)
	rpcServer.Accept(listener)

}
