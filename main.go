package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const defaultConfig string = "bazaar.yml"

func main() {

	var config string
	flag.StringVar(&config, "config", defaultConfig, "")
	flag.Parse()

	// catch signals so we can gracefully exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	node, err := CreateNodeFromConfigPath(config)
	if err != nil {
		log.Fatalf("Error creating node from config at %s: %s", defaultConfig, err)
		return
	}
	log.Printf("Loaded config for bazaar node. Node ID: %d\n", node.config.NodeID)

	// Finally, listen on rpc.
	log.Printf("Listening on port %d for incoming RPC connections...", node.config.NodePort)
	stopChan := make(chan bool)
	doneChan := make(chan bool)

	// closing listener on signal
	go func(nodeStop chan bool) {
		s := <-sigc
		log.Printf("Received signal %s, closing listener and stopping bazaar...\n", s.String())
		<-doneChan
		nodeStop <- true
		close(doneChan)
		close(stopChan)
	}(stopChan)

	server := &BazaarServer{
		node: node,
	}

	go server.node.init()

	server.ListenRPC(stopChan, doneChan)

}
