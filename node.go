package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// BazaarNode contains the state for the node.
type BazaarNode struct {
	config NodeConfig
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
