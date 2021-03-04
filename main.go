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

func (bnode *BazaarNode) buyerLoop() {
	// wait before starting the buyer loop
	time.Sleep(2 * time.Second)

	for {

		// Generate a buy request
		for targetID := range bnode.config.Items {
			if bnode.config.Items[targetID].Amount != 0 {
				bnode.config.Target = bnode.config.Items[targetID].Item
			}
		}
		log.Printf("Node %d plans to buy %s", bnode.config.NodeID, bnode.config.Target)

		// Lookup request to neighbours
		portStr := net.JoinHostPort("", strconv.Itoa(bnode.config.NodePort))
		args := LookupArgs{
			ProductName: bnode.config.Target,
			HopCount:    bnode.config.MaxHops,
			BuyerID:     bnode.config.NodeID,
			Route:       []Peer{{PeerID: bnode.config.NodeID, Addr: portStr}},
		}
		var rpcResponse LookupResponse
		go bnode.Lookup(args, &rpcResponse)

		log.Printf("Waiting to retrieve sellers...")
		// Buy from the list of available sellers
		time.Sleep(200 * time.Millisecond)
		var sellerList []Peer
		for i := 0; i < len(bnode.sellerChannel); i++ {
			sellerList = append(sellerList, <-bnode.sellerChannel)
		}

		if len(sellerList) != 0 {
			randomSeller := sellerList[rand.Intn(len(sellerList))]
			go bnode.buy(randomSeller)
			log.Printf("Node %d buys %s from seller node %d", bnode.config.NodeID, bnode.config.Target, randomSeller.PeerID)
		}

	}

}
