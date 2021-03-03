package main

import (
	"log"
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

	// Finally, listen on rpc.
	log.Printf("Listening on port %d for incoming RPC connections...", node.config.NodePort)
	stopChan := make(chan bool)
	doneChan := make(chan bool)
	server := &BazaarServer{
		node: node,
	}
	server.ListenRPC(stopChan, doneChan)

}
