package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const defaultConfig string = "bazaar.yml"

func main() {

	// catch signals so we can gracefully exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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

	if node.config.Role == "buyer" {
		go server.node.buyerLoop()
	}
	server.ListenRPC(stopChan, doneChan)

}

func (node *BazaarNode) buyerLoop() {
	// wait before starting the buyer loop
	time.Sleep(5 * time.Second)

	for {

		// Generate a buy request
		for targetID := range node.config.Items {
			if node.config.Items[targetID].Amount != 0 {
				node.config.Target = node.config.Items[targetID].Item
			}
		}
		log.Printf("Node %d plans to buy %s", node.config.NodeID, node.config.Target)

		// Lookup request to neighbours
		portStr := net.JoinHostPort("", strconv.Itoa(node.config.NodePort))
		args := LookupArgs{
			ProductName: node.config.Target,
			HopCount:    node.config.MaxHops,
			BuyerID:     node.config.NodeID,
			Route:       []Peer{{PeerID: node.config.NodeID, Addr: portStr}},
		}
		var rpcResponse LookupResponse
		node.Lookup(args, &rpcResponse)

		// Buy from the list of available sellers
		time.Sleep(5 * time.Second)
		var sellerList []Peer
		for i := 0; i < len(node.sellerChannel); i++ {
			sellerList = append(sellerList, <-node.sellerChannel)
		}

		if len(sellerList) != 0 {
			randomSeller := sellerList[rand.Intn(len(sellerList))]
			node.buy(randomSeller)
			log.Printf("Node %d buys %s from seller node %d", node.config.NodeID, node.config.Target, randomSeller.PeerID)
		}

	}

}
