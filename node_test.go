package main

import (
	"testing"
)

const testingConfig string = `
peers:
  0: localhost:10000
role: "buyer"
items:
  - item: "salt"
    amount: 10
    unlimited: false
  - item: "boars"
    amount: 0
    unlimited: true
  - item: "fish"
    amount: 1
    unlimited: false

maxpeers: 1
maxhops: 1
nodeid: 1
nodeport: 10001
`

// TestLookupSimple tests that a lookup connection can be made. It is also an
// example of how to call Lookup and initialize a node from a file.
func TestLookupSimple(t *testing.T) {

	// create testnode from test config
	testnode, err := CreateNodeFromConfigFile([]byte(testingConfig))
	if err != nil {
		t.Fatalf("Error configuring node for test rpc call: %s", err)
		return
	}

	args := LookupArgs{
		ProductName: "salt",
		HopCount:    0,
		SellerID:    0,
	}
	var rpcResponse LookupResponse
	err = testnode.Lookup(args, &rpcResponse)
	if err != nil {
		t.Fatalf("Error calling lookup on testnode: %s", err)
		return
	}

}

const nodeA string = `
peers:
  0: localhost:10000
role: "buyer"
items:
  - item: "salt"
    amount: 1
    unlimited: false
  - item: "boars"
    amount: 1
    unlimited: true
  - item: "fish"
    amount: 1
    unlimited: false

maxpeers: 1
maxhops: 1
nodeid: 1
nodeport: 10001
`

const nodeB string = `
peers:
  1: localhost:10001
role: "seller"
items:
  - item: "salt"
    amount: 1
    unlimited: false
  - item: "boars"
    amount: 1
    unlimited: true
  - item: "fish"
    amount: 1
    unlimited: false

maxpeers: 1
maxhops: 1
nodeid: 0
nodeport: 10000
`

// TestReplySimpleRPC tests that the reply method correctly responds through
// RPC.
func TestReplySimpleRPC(t *testing.T) {

	// create testnode from test config
	testNodeA, err := CreateNodeFromConfigFile([]byte(nodeA))
	if err != nil {
		t.Fatalf("Error configuring node A for test rpc call: %s", err)
		return
	}

	// create testnode from test config
	testNodeB, err := CreateNodeFromConfigFile([]byte(nodeB))
	if err != nil {
		t.Fatalf("Error configuring node B for test rpc call: %s", err)
		return
	}

	t.Logf("listening on rpc\n")

	// now we listen on the node's config port
	stopChan := make(chan bool, 1)
	serverB := &BazaarServer{node: testNodeB}
	go serverB.ListenRPC(stopChan)

	args := ReplyArgs{
		RouteList: []int{1},
		SellerID:  0,
	}

	var rpcResponse ReplyResponse
	err = testNodeA.Reply(args, &rpcResponse)
	if err != nil {
		t.Fatalf("error replying: %s", err)
		return
	}

	// clean up channel
	close(stopChan)
}

func TestBuySimple(t *testing.T) {

	// create testnode from test config
	testnode, err := CreateNodeFromConfigFile([]byte(testingConfig))
	if err != nil {
		t.Fatalf("Error configuring node for test rpc call: %s", err)
		return
	}

	transactionArgs := TransactionArgs{"salt"}
	var transactionResponse TransactionResponse
	err = testnode.Sell(transactionArgs, &transactionResponse)
	if err != nil {
		t.Fatalf("error selling: %s", err)
		return
	}
}
