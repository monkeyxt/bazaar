package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const defaultConfig string = "bazaar.yml"

func main() {

	// Load config location from commandline flag
	var config string
	flag.StringVar(&config, "config", defaultConfig, "")
	flag.Parse()

	// Create file to dump the node log
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error creating log: %s", err)
		return
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Catch signals so we can gracefully exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Create node based on specified configuration file
	node, err := CreateNodeFromConfigPath(config)
	if err != nil {
		log.Fatalf("Error creating node from config at %s: %s", defaultConfig, err)
		return
	}
	log.Printf("Loaded config for bazaar node. Node ID: %d\n", node.config.NodeID)

	// Finally, listen on rpc
	log.Printf("Listening on port %d for incoming RPC connections...", node.config.NodePort)
	stopChan := make(chan bool)
	doneChan := make(chan bool)

	// Closing listener on signal
	go func(nodeStop chan bool) {
		s := <-sigc
		log.Printf("Received signal %s, closing listener and stopping bazaar...\n", s.String())
		<-doneChan
		nodeStop <- true
		close(doneChan)
		close(stopChan)
	}(stopChan)

	// Initialize the node
	server := &BazaarServer{
		node: node,
	}
	go server.node.init()
	server.ListenRPC(stopChan, doneChan)

}
