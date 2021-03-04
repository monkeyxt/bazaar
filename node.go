package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"

	"gopkg.in/yaml.v2"
)

// BazaarNode contains the state for the node.
type BazaarNode struct {
	config        NodeConfig
	sellerChannel chan Peer
}

// BazaarServer exposes methods for letting a node listen for RPC
type BazaarServer struct {
	node *BazaarNode
}

// CreateNodeFromConfigFile loads initial node state from a config file passed as
// bytes.
func CreateNodeFromConfigFile(configFile []byte) (*BazaarNode, error) {
	// load from YAML at the desired path
	var node BazaarNode
	err := yaml.Unmarshal(configFile, &node.config)
	if err != nil {
		return nil, err
	}

	// warn and return an error if the current node id is in the peer list.
	for peer := range node.config.Peers {
		if peer == node.config.NodeID {
			return nil, fmt.Errorf("the current node [%d] is in the peers list! Why buy from yourself? This could cause a cycle and peers would get duplicate lookup messages if this were allowed", peer)
		}
	}

	// return an error if there are more peers in the peerlist than the max
	// peers allowed.
	if len(node.config.Peers) > node.config.MaxPeers {
		return nil, fmt.Errorf("too many peers in the peers list. There are %d, the maximum is %d", len(node.config.Peers), node.config.MaxPeers)
	}

	// NOTE: project wasnt specific on how to select seller items, so we pick at
	// random
	// set the sellertarget depending on the available items
	availableItems, err := GetAvailableItems(&node)
	if err != nil {
		return nil, fmt.Errorf("error getting available items when loading config: %s", err)
	}

	// if available items is empty just pick from len(items)
	var randItemIdx int
	if len(availableItems) == 0 {
		randItemIdx = rand.Intn(len(node.config.Items))
	} else {
		// pick item at random from the list of available items
		randItemIdx = rand.Intn(len(availableItems))
	}

	// set the seller target to the randomly selected item
	node.config.SellerTarget = node.config.Items[randItemIdx].Item

	// initialize the seller channel, just have 100 max for now
	node.sellerChannel = make(chan Peer, 100)

	return &node, nil
}

// GetAvailableItems returns all items available for the given node.
// It returns an error if the list of items is empty.
func GetAvailableItems(bnode *BazaarNode) ([]string, error) {
	if len(bnode.config.Items) == 0 {
		return nil, fmt.Errorf("bazaar node has no items, so none can be available")
	}

	var items []string
	for _, item := range bnode.config.Items {
		if item.Unlimited || item.Amount > 0 {
			items = append(items, item.Item)
		}
	}

	return items, nil
}

// CreateNodeFromConfigPath loads initial node state from a config at a certain
// path.
func CreateNodeFromConfigPath(path string) (*BazaarNode, error) {
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return CreateNodeFromConfigFile(configFile)
}

// LookupArgs contains the RPC arguments for lookup, which is a product name,
// hopcount, and buyerid to be passed.
type LookupArgs struct {
	ProductName string
	HopCount    int
	BuyerID     int
	Route       []Peer
}

// LookupResponse is empty because no response is required for lookup.
type LookupResponse struct {
}

// Lookup runs the lookup command.
func (bnode *BazaarNode) Lookup(args LookupArgs, reply *LookupResponse) error {
	log.Printf("Node %d is looking for %d with lookup for %s", bnode.config.NodeID, args.BuyerID, args.ProductName)
	return bnode.lookupProduct(args.Route, args.ProductName, args.HopCount, args.BuyerID)
}

// lookupProduct takes in a product name and hopcount, and runs the lookup procedure.
func (bnode *BazaarNode) lookupProduct(route []Peer, productName string, hopcount int, buyerID int) error {

	if bnode.config.Role == "seller" && bnode.config.SellerTarget == productName {
		go bnode.reply(route, bnode.config.NodeID)
	}

	log.Printf("Node %d received lookup request from %d\n", bnode.config.NodeID, buyerID)
	if hopcount == 0 {
		log.Printf("Node %d is discarding lookup request for %s\n", bnode.config.NodeID, productName)
		return nil
	}

	log.Printf("Node %d flooding peers with lookup requests for %s from %d...\n", bnode.config.NodeID, productName, buyerID)
	for peer, addr := range bnode.config.Peers {
		if peer == bnode.config.NodeID {
			// if this is us, then don't worry about doing a lookup
			continue
		}
		// need to add this node to the route, and call lookup on this
		portStr := net.JoinHostPort("", strconv.Itoa(bnode.config.NodePort))
		route = append(route, Peer{bnode.config.NodeID, portStr})

		log.Printf("Node %d is flooding peer %d for lookup\n", bnode.config.NodeID, peer)
		go func(peerAddr string) {

			con, err := rpc.Dial("tcp", peerAddr)
			if err != nil {
				log.Fatalln("dailing error in lookup: ", err)
			}

			req := LookupArgs{
				ProductName: productName,
				HopCount:    hopcount - 1,
				BuyerID:     buyerID,
				Route:       route,
			}
			var res LookupResponse

			err = con.Call("node.Lookup", req, &res)
			if err != nil {
				log.Fatalln("reply error: ", err)
			}

			log.Printf("Lookup to %s finished\n", peerAddr)
		}(addr)

	}

	log.Printf("Node %d is done flooding peers with lookups for %s from %d...\n", bnode.config.NodeID, productName, buyerID)

	return nil
}

// Reply relays the message back to the buyer
func (bnode *BazaarNode) Reply(args ReplyArgs, reply *ReplyResponse) error {
	if len(args.RouteList) == 1 {
		log.Printf("Message at final hop: node %v with message from seller node %d", args.RouteList[len(args.RouteList)-1], args.SellerID)
	} else {
		log.Printf("Forward reply to node %v with message from seller node %d", args.RouteList[len(args.RouteList)-2], args.SellerID)
	}

	return bnode.reply(args.RouteList, args.SellerID)
}

// ReplyArgs contains the RPC arguments for reply, which is the backtracking list
// and the sellerid to be returned
type ReplyArgs struct {
	RouteList []Peer
	SellerID  int
}

// ReplyResponse is empty because no response is required.
type ReplyResponse struct {
}

// Reply message with the peerId of the seller
func (bnode *BazaarNode) reply(routeList []Peer, sellerID int) error {

	// routeList: a list of ids to traverse back to the original sender in the format of
	//         [1, 5, 2, 6], so the reverse traversal path should be 6 --> 2 --> 5 --> 1

	// sellerID: id of the seller who responds

	if len(routeList) == 1 {

		// Reached original sender, add the sellerID to a list for the buyer to randomly
		// choose from.
		log.Printf("Node %d got a match reply from node %d ", bnode.config.NodeID, sellerID)

		bnode.sellerChannel <- routeList[0]
		log.Printf("Added %v to seller channel", routeList[0])

	} else {

		var recipient Peer
		recipient, routeList = routeList[len(routeList)-2], routeList[:len(routeList)-1]

		log.Printf("Current recepent ID: %d: ", recipient.PeerID)
		log.Printf("Current routing List %v: ", routeList)

		con, err := rpc.Dial("tcp", recipient.Addr)
		if err != nil {
			log.Fatalln("dailing error: ", err)
		}

		req := ReplyArgs{routeList, sellerID}
		var res ReplyResponse

		err = con.Call("node.Reply", req, &res)
		if err != nil {
			log.Fatalln("reply error: ", err)
		}

	}

	return nil
}

// TransactionArgs contains the RPC arguments for buy. CurrentTarget is the
// what the buyer wishes to buy during this transaction
type TransactionArgs struct {
	CurrentTarget string
}

// TransactionResponse is empty for now
type TransactionResponse struct {
}

// Buy item directly from the seller with RCP call
func (bnode *BazaarNode) buy(seller Peer) error {

	log.Printf("Node %d buying from seller node %d", bnode.config.NodeID, seller.PeerID)

	con, err := rpc.Dial("tcp", seller.Addr)
	if err != nil {
		log.Fatalln("dailing error: ", err)
	}

	req := TransactionArgs{bnode.config.Target}
	var res TransactionResponse

	err = con.Call("node.Sell", req, &res)
	if err != nil {
		log.Fatalln("reply error: ", err)
	}

	return nil

}

// Sell runs the sell command
func (bnode *BazaarNode) Sell(args TransactionArgs, reply *TransactionResponse) error {
	log.Printf("Seller node %d selling item %s", bnode.config.NodeID, args.CurrentTarget)
	return bnode.sell(args.CurrentTarget)
}

func (bnode *BazaarNode) sell(target string) error {

	// target: the requested item by the buyer

	// Extract the itemID for the requested item
	var targetID int
	for itemID := range bnode.config.Items {
		if bnode.config.Items[itemID].Item == target {
			targetID = itemID
		}
	}

	// Complete the transaction
	bnode.config.Mu.Lock()
	if bnode.config.Items[targetID].Amount > 0 {

		bnode.config.Items[targetID].Amount--
		log.Printf("Seller node %d sold %s, amount remaining %d", bnode.config.NodeID, bnode.config.Items[targetID].Item, bnode.config.Items[targetID].Amount)

	} else {

		// If the item is defined to be unlimited in the YAML file restock the item and purchase again
		if bnode.config.Items[targetID].Unlimited == true {

			bnode.config.Items[targetID].Amount += 10
			log.Printf("Seller node %d restocked %s", bnode.config.NodeID, bnode.config.Items[targetID].Item)

			bnode.config.Items[targetID].Amount--
			log.Printf("Seller node %d sold %s, amount remaining %d", bnode.config.NodeID, bnode.config.Items[targetID].Item, bnode.config.Items[targetID].Amount)

		} else {

			// Item sold out. Pick another item randomly to sell
			var commodity []string
			for itemID := range bnode.config.Items {
				if bnode.config.Items[itemID].Amount != 0 {
					commodity = append(commodity, bnode.config.Items[itemID].Item)
				}
			}

			bnode.config.SellerTarget = commodity[rand.Intn(len(commodity))]
			log.Printf("Seller node %d now selling %s", bnode.config.NodeID, bnode.config.SellerTarget)
		}

	}
	bnode.config.Mu.Unlock()

	return nil

}

// ListenRPC listens on RPC for all methods on the desired listener. To stop
// listening, one passes a bool to the stopChannel or closes stopChannel. This
// method should be run in a goroutine. The listener passed will be closed if
// something stopChannel receives a message. The doneListening channel will be
// sent a bool when the server is ready to accept connections.
func (server *BazaarServer) ListenRPC(stopChannel chan bool, doneListening chan bool) {

	addr := net.JoinHostPort("", strconv.Itoa(server.node.config.NodePort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		doneListening <- true
		log.Fatalf("Error listening for RPC: %s", err)
		return
	}

	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("node", server.node)

	defer func() {
		log.Printf("Closing %s listener for %s...\n", listener.Addr().Network(), listener.Addr().String())
		listener.Close()
	}()

	log.Printf("Node %d listening for rpc on address %s\n", server.node.config.NodeID, addr)
	// listen in goroutine so we can block until receiving a message in
	// stopChannel
	go rpcServer.Accept(listener)
	doneListening <- true

	// wait until something is in stopchannel or it is closed
	<-stopChannel

	return
}

// unique is a function to de-duplicate the seller list
func unique(intSlice []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
