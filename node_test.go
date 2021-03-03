package main

import (
	"testing"
)

const testingConfig string = `
peers:
  0: localhost:99999
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
nodeport: 30001
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
		BuyerID:     0,
		Route:       []int{0},
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
  0: localhost:20000
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
nodeport: 20001
`

const nodeB string = `
peers:
  1: localhost:20001
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
nodeport: 20000
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
	doneChan := make(chan bool)
	serverB := &BazaarServer{node: testNodeB}
	go serverB.ListenRPC(stopChan, doneChan)
	<-doneChan

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
	close(doneChan)
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

// firstNode wants to buy one thing - salt. max hops is 4.
const firstNode string = `
peers:
  1: localhost:10001
role: "buyer"
items:
  - item: "salt"
    amount: 1
    unlimited: false
  - item: "boars"
    amount: 0
    unlimited: true
  - item: "fish"
    amount: 0
    unlimited: false

maxpeers: 1
maxhops: 4
nodeid: 0
nodeport: 10000
`

// secondNode wants to buy nothing, it's just an intermediate node that listens
// on rpc.
const secondNode string = `
peers:
  2: localhost:10002
role: "seller"
items:
  - item: "salt"
    amount: 0
    unlimited: false
  - item: "boars"
    amount: 0
    unlimited: false
  - item: "fish"
    amount: 0
    unlimited: false

maxpeers: 1
maxhops: 4
nodeid: 1
nodeport: 10001
`

// thirdNode is selling one unit of salt.
const thirdNode string = `
peers:
role: "seller"
items:
  - item: "salt"
    amount: 1
    unlimited: false
  - item: "boars"
    amount: 0
    unlimited: false
  - item: "fish"
    amount: 0
    unlimited: false

maxpeers: 1
maxhops: 4
nodeid: 2
nodeport: 10002
`

// TestLookupLinearRPC sets up a small network of 3 nodes, and is intended to
// test the backward routing of reply as well as the flooding during Lookup.
// In this test, the max number of hops will not be reached.
func TestLookupLinearRPC(t *testing.T) {

	// create firstnode from test config
	testFirstNode, err := CreateNodeFromConfigFile([]byte(firstNode))
	if err != nil {
		t.Fatalf("Error configuring firstNode for test rpc call: %s", err)
		return
	}

	// create secondnode from test config
	testSecondNode, err := CreateNodeFromConfigFile([]byte(secondNode))
	if err != nil {
		t.Fatalf("Error configuring secondNode for test rpc call: %s", err)
		return
	}

	// create thirdnode from test config
	testThirdNode, err := CreateNodeFromConfigFile([]byte(thirdNode))
	if err != nil {
		t.Fatalf("Error configuring thirdNode for test rpc call: %s", err)
		return
	}

	// firstnode rpc listen
	t.Logf("listening on rpc for secondnode\n")
	firstStopChan := make(chan bool, 1)
	firstDoneChan := make(chan bool)
	firstServer := &BazaarServer{node: testFirstNode}
	go firstServer.ListenRPC(firstStopChan, firstDoneChan)

	// secondnode rpc listen
	t.Logf("listening on rpc for secondnode\n")
	secondStopChan := make(chan bool, 1)
	secondDoneChan := make(chan bool)
	secondServer := &BazaarServer{node: testSecondNode}
	go secondServer.ListenRPC(secondStopChan, secondDoneChan)

	// thirdnode rpc listen
	t.Logf("listening on rpc for thirdnode\n")
	thirdStopChan := make(chan bool, 1)
	thirdDoneChan := make(chan bool)
	thirdServer := &BazaarServer{node: testThirdNode}
	go thirdServer.ListenRPC(thirdStopChan, thirdDoneChan)

	// make sure servers are ready to accept connections
	<-firstDoneChan
	<-secondDoneChan
	<-thirdDoneChan

	args := LookupArgs{
		ProductName: "salt",
		HopCount:    testFirstNode.config.MaxHops - 1,
		BuyerID:     testFirstNode.config.NodeID,
		Route:       []int{testFirstNode.config.NodeID},
	}

	var rpcResponse LookupResponse
	err = testFirstNode.Lookup(args, &rpcResponse)
	if err != nil {
		t.Fatalf("error with lookup: %s", err)
		return
	}

	// clean up channels
	close(firstStopChan)
	close(secondStopChan)
	close(thirdStopChan)

	close(firstDoneChan)
	close(secondDoneChan)
	close(thirdDoneChan)
}
