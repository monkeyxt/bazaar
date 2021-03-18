package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const defaultConfig string = "bazaar.yml"
const defaultLogFile string = "log.txt"

func main() {

	// Seed math rand for more random looking results
	err := SeedMathRand()
	if err != nil {
		log.Fatalf("Error seeding math rand: %s\n", err)
		return
	}

	// Load config location from commandline flag
	var config string
	var logFileLocation string

	// output a lot
	var verbose bool
	flag.StringVar(&config, "config", defaultConfig, "The config used to define node behavior (default is bazaar.yml).")
	flag.StringVar(&logFileLocation, "logfile", defaultLogFile, "The file which logs should be written to (default is log.txt).")
	flag.BoolVar(&verbose, "verbose", false, "Add this flag if you want verbose logging output.")
	flag.Parse()

	// Create file to dump the node log
	logFile, err := os.OpenFile(logFileLocation, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
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
	if verbose {
		log.Printf("Loaded config for bazaar node. Node ID: %d\n", node.config.NodeID)
	}

	perfLogFile, err := os.OpenFile(fmt.Sprint("perflog", node.config.NodeID, ".txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	perflogger := log.New(perfLogFile, "", 0)
	node.PerfLogger = perflogger

	node.VerboseLogging = verbose

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
