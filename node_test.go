package main

import (
	"net"
	"strconv"
	"testing"

	"github.com/rjected/bazaar/nodeconfig"
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

	portStr := net.JoinHostPort("", strconv.Itoa(testnode.config.NodePort))
	args := LookupArgs{
		ProductName: "salt",
		HopCount:    0,
		BuyerID:     0,
		Route:       []nodeconfig.Peer{{PeerID: testnode.config.NodeID, Addr: portStr}},
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

	// the reply args use test node B instead of A because test node B would
	// have called the lookup method.
	portStr := net.JoinHostPort("", strconv.Itoa(testNodeB.config.NodePort))
	args := ReplyArgs{
		RouteList:  []nodeconfig.Peer{{PeerID: testNodeB.config.NodeID, Addr: portStr}},
		SellerInfo: nodeconfig.Peer{testNodeB.config.NodeID, net.JoinHostPort("", strconv.Itoa(testNodeB.config.NodePort))},
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

	portStr := net.JoinHostPort("", strconv.Itoa(testFirstNode.config.NodePort))
	args := LookupArgs{
		ProductName: "salt",
		HopCount:    testFirstNode.config.MaxHops - 1,
		BuyerID:     testFirstNode.config.NodeID,
		Route:       []nodeconfig.Peer{{PeerID: testFirstNode.config.NodeID, Addr: portStr}},
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

// TestMilestoneOne tests the cases required for milestone one. This includes
// three tests:
//
// 1. Assign one peer to be a buyer of fish and another to be a seller of fish.
// Ensure that all fish is sold and restocked forever.
//
// 2. Assign one peer to be a buyer of fish and another to be a seller of boar.
// Ensure that nothing is sold.
//
// 3. Randomly assign buyer and seller roles. Ensure that items keep being sold
// throughout.
// NOTE: in order for items to be sold throughout, there need to be matching
// buyers and sellers which either never run out of items (and do not change),
// or the restocking logic (both on the buyer and seller side) must be such that
// restocking also changes the item sold.
//
func TestMilestoneOne(t *testing.T) {
	// fishBuyerForever := `
	// peers:
	//   0: localhost:44444
	// role: "buyer"
	// items:
	//   - item: "fish"
	// 	amount: 1
	// 	unlimited: true

	// maxpeers: 1
	// maxhops: 1
	// nodeid: 1
	// nodeport: 33333
	// `
	// fishSellerForever := `
	// peers:
	//   1: localhost:33333
	// role: "seller"
	// items:
	//   - item: "fish"
	// 	amount: 1
	// 	unlimited: true

	// maxpeers: 1
	// maxhops: 1
	// nodeid: 0
	// nodeport: 44444
	// `
	// // boarSellerForever is connected to the fish buyer, reuse fish buyer
	// boarSellerForever := `
	// peers:
	//   1: localhost:33333
	// role: "seller"
	// items:
	//   - item: "boar"
	// 	amount: 1
	// 	unlimited: true

	// maxpeers: 1
	// maxhops: 1
	// nodeid: 0
	// nodeport: 43334
	// `

}
