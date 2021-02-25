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

	return
}
