package main

import "sync"

// NodeConfig includes a list of peers, node role, a list of items (with
// amounts), and a maximum number of peers to connect to.
type NodeConfig struct {
	// Peers is a map from a peer ID to an address
	// TODO: consider the case when the node can be both the buyer & seller
	Peers    map[int]string `yaml:"peers,omitempty"`
	Role     string         `yaml:"role"`
	Items    []ItemAmount   `yaml:",flow"`
	MaxPeers int            `yaml:"maxpeers"`
	MaxHops  int            `yaml:"maxhops"`

	// NodeID is the ID of the node
	NodeID int `yaml:"nodeid"`

	// NodePort is the port for the node to listen on for RPC
	NodePort int `yaml:"nodeport"`

	// SellerList is a list of sellers for the buyer to choose from
	SellerList []int

	// Mu is the mutex lock for the current node
	Mu sync.Mutex

	// Target is the item that the buyer wishes to buy
	Target string

	// SellerTarget is the item that the seller is currently selling
	SellerTarget string
}

// ItemAmount is an item, associated amount, and an Unlimited setting. If
// unlimited is set to true, then the amount is ignored and the item is treated
// as unlimited.
type ItemAmount struct {
	Item      string `yaml:"item"`
	Amount    int    `yaml:"amount"`
	Unlimited bool   `yaml:"unlimited"`
}
