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
	log.Printf("Replying to the previous node %d with message from seller %d", args.RouteList[0], args.SellerID)
	return bnode.replyBuyer(args.RouteList, args.SellerID)
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
func (bnode *BazaarNode) replyBuyer(routeList []int, sellerID int) error {

	// idList: a list of ids to traverse back to the original sender
	// sellerID: id of the seller who responds

	if len(routeList) == 1 {

		// Reached original sender
		log.Printf("%d got a match reply from %d ", routeList[0], sellerID)

		// TODO: add sellerID to list of sellers for the buyer to randomly pick from

	} else {

		var recepientID int
		recepientID, routeList = routeList[len(routeList)-1], routeList[:len(routeList)-1]

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
