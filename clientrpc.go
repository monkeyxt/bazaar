package main

import (
	"fmt"
	"log"
	"net/rpc"
	"sync"
	"time"

	"github.com/rjected/bazaar/nodeconfig"
)

// getClientForPeer either gets the client for the desired peer, or creates it
// and returns it if it does not exist. The client returned is also inserted
// into the node's client map, so future requests will not create a new client.
// This method is thread-safe.
func (bnode *BazaarNode) getClientForPeer(peer nodeconfig.Peer) (*rpc.Client, error) {

	bnode.peerClientLock.Lock()

	// unlock when we return
	defer func(lock *sync.Mutex) {
		lock.Unlock()
	}(bnode.peerClientLock)

	client, ok := bnode.peerClients[peer.PeerID]
	if !ok {
		// if the peer does not exist, dial the peer and keep the connection
		// open.

		newClient, err := rpc.Dial("tcp", peer.Addr)
		if err != nil {
			return nil, fmt.Errorf("dailing error in lookup: %s", err)
		}

		// now insert the client!
		bnode.peerClients[peer.PeerID] = newClient

		return newClient, nil
	}

	return client, nil

}

// callReplyRPC calls the reply RPC with the given routelist to the given peer
func (bnode *BazaarNode) callReplyRPC(replyPeer nodeconfig.Peer, routeList []nodeconfig.Peer, sellerInfo nodeconfig.Peer, lookupUUID int) {

	startTime := time.Now()

	// get client
	client, err := bnode.getClientForPeer(replyPeer)
	if err != nil {
		log.Fatalf("Error getting client during sell call: %s\n", err)
	}

	req := ReplyArgs{routeList, sellerInfo, lookupUUID}
	var res ReplyResponse

	err = client.Call("node.Reply", req, &res)
	if err != nil {
		log.Fatalln("reply error: ", err)
	}
	end := time.Now()
	bnode.reportRPCLatency(startTime, end, replyPeer.Addr)

}

// callSellRPC calls the sell RPC to the given node, and reports latency.
func (bnode *BazaarNode) callSellRPC(seller nodeconfig.Peer) {

	start := time.Now()

	// get client
	client, err := bnode.getClientForPeer(seller)
	if err != nil {
		log.Fatalf("Error getting client during sell call: %s\n", err)
	}

	req := TransactionArgs{CurrentTarget: bnode.config.BuyerTarget, BuyerID: bnode.config.NodeID}
	var res TransactionResponse

	err = client.Call("node.Sell", req, &res)
	if err != nil {
		log.Fatalln("sell call error: ", err)
	}

	end := time.Now()
	bnode.reportRPCLatency(start, end, seller.Addr)

}

// callLookupRPC is meant to be run in a goroutine and call the lookup RPC to the
// given peer. It will also take care of reporting latency.
func (bnode *BazaarNode) callLookupRPC(route []nodeconfig.Peer, lookupPeer nodeconfig.Peer, productName string, hopcount, buyerID int, uuid int) {

	start := time.Now()

	// get client
	client, err := bnode.getClientForPeer(lookupPeer)
	if err != nil {
		log.Fatalf("Error getting client during lookup call: %s\n", err)
	}

	req := LookupArgs{
		ProductName: productName,
		HopCount:    hopcount - 1,
		BuyerID:     buyerID,
		Route:       route,
		UUID:        uuid,
	}
	var res LookupResponse

	err = client.Call("node.Lookup", req, &res)
	if err != nil {
		log.Fatalln("lookup call error: ", err)
	}

	end := time.Now()
	bnode.reportRPCLatency(start, end, lookupPeer.Addr)

}
