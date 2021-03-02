package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"strconv"

	"gopkg.in/yaml.v2"
)

// BazaarNode contains the state for the node.
type BazaarNode struct {
	config NodeConfig
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

	return &node, nil
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

// Lookup runs the lookup command.
func (bnode *BazaarNode) Lookup(args LookupArgs, reply *LookupResponse) error {
	log.Printf("Looking for %d with lookup for %s", args.SellerID, args.ProductName)
	return bnode.lookupProduct(args.ProductName, args.HopCount, args.SellerID)
}

// LookupArgs contains the RPC arguments for lookup, which is a product name,
// hopcount, and sellerid to be passed.
type LookupArgs struct {
	ProductName string
	HopCount    int
	SellerID    int
}

// LookupResponse is empty because no response is required for lookup.
type LookupResponse struct {
}

// lookupProduct takes in a product name and hopcount, and runs the lookup procedure.
func (bnode *BazaarNode) lookupProduct(productName string, hopcount int, sellerID int) error {

	if hopcount == 0 {
		if bnode.config.Role == "buyer" {
			return nil
		}

		// TODO: call reply(bnode.config.NodeID, sellerID)
	}

	// for peer, addr := range bnode.config.Peers {
	// TODO: call lookp rpc with hopcount - 1 and the product name
	// }

	return nil
}

// Reply relays the message back to the buyer
func (bnode *BazaarNode) Reply(args ReplyArgs, reply *ReplyResponse) error {
	if len(args.RouteList) == 1 {
		log.Printf("Message at final hop: node %d with message from seller node %d", args.RouteList[len(args.RouteList)-1], args.SellerID)
	} else {
		log.Printf("Forward reply to node %d with message from seller node %d", args.RouteList[len(args.RouteList)-2], args.SellerID)
	}

	return bnode.reply(args.RouteList, args.SellerID)
}

// ReplyArgs contains the RPC arguments for reply, which is the backtracking list
// and the sellerid to be returned
type ReplyArgs struct {
	RouteList []int
	SellerID  int
}

// ReplyResponse is empty because no response is required.
type ReplyResponse struct {
}

// Reply message with the peerId of the seller
func (bnode *BazaarNode) reply(routeList []int, sellerID int) error {

	// routeList: a list of ids to traverse back to the original sender in the format of
	//         [1, 5, 2, 6], so the reverse traversal path should be 6 --> 2 --> 5 --> 1

	// sellerID: id of the seller who responds

	if len(routeList) == 1 {

		// Reached original sender, add the sellerID to a list for the buyer to randomly
		// choose from.
		log.Printf("Node %d got a match reply from node %d ", bnode.config.NodeID, sellerID)

		var tempSellerList []int
		tempSellerList = append([]int{sellerID}, tempSellerList...)

		// Remove duplicate sellers from the list
		bnode.config.SellerList = unique(tempSellerList)
		log.Printf("Current seller list: %v", bnode.config.SellerList)

	} else {

		var recepientID int
		recepientID, routeList = routeList[len(routeList)-2], routeList[:len(routeList)-1]

		log.Printf("Current recepent ID: %d: ", recepientID)
		log.Printf("Current routing List %v: ", routeList)

		addr := bnode.config.Peers[recepientID]

		con, err := rpc.DialHTTP("tcp", addr)
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
func (bnode *BazaarNode) buy(sellerID int) error {

	log.Printf("Node %d buying from seller node %d", bnode.config.NodeID, sellerID)

	addr := bnode.config.Peers[sellerID]

	con, err := rpc.DialHTTP("tcp", addr)
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

	// In the case of unlimted item. Nothing to do, return nil.
	if bnode.config.Items[targetID].Unlimited == true {
		log.Printf("Seller node %d sold %s", bnode.config.NodeID, bnode.config.Items[targetID].Item)
		return nil
	}

	// Complete the transaction
	bnode.config.Mu.Lock()
	if bnode.config.Items[targetID].Amount > 0 {
		bnode.config.Items[targetID].Amount--
		log.Printf("Seller node %d sold %s, amount remaining %d", bnode.config.NodeID, bnode.config.Items[targetID].Item, bnode.config.Items[targetID].Amount)
	} else {

		// TODO: Item sold out. Pick another item to sell

	}
	bnode.config.Mu.Unlock()

	return nil

}

// ListenRPC listens on RPC for all methods on the desired listener. To stop
// listening, one passes a bool to the stopChannel or closes stopChannel. This
// method should be run in a goroutine. The listener passed will be closed if
// something stopChannel receives a message.
func (server *BazaarServer) ListenRPC(stopChannel chan bool) {

	addr := net.JoinHostPort("", strconv.Itoa(server.node.config.NodePort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Error listening for RPC: %s", err)
		return
	}

	rpcServer := rpc.NewServer()
	rpcServer.Register(server.node)

	defer func() {
		log.Printf("Closing %s listener for %s...\n", listener.Addr().Network(), listener.Addr().String())
		listener.Close()
	}()

	// listen in goroutine so we can block until receiving a message in
	// stopChannel
	go rpcServer.Accept(listener)

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
