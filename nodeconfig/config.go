// Package nodeconfig contains the configuration structures for bazaar nodes.
package nodeconfig

import (
	"sync"
)

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

	// NodeIP is the host address of the current node
	NodeIP string `yaml:"nodeip"`

	// NodeID is the ID of the node
	NodeID int `yaml:"nodeid"`

	// NodePort is the port for the node to listen on for RPC
	NodePort int `yaml:"nodeport"`

	// SellerList is a list of sellers for the buyer to choose from
	SellerList []int `yaml:"-"`

	// BuyerOptionList is a list of items for the buyer to choose from
	BuyerOptionList []string `yaml:",flow"`

	// BuyerTarget is the item that the buyer wishes to buy
	BuyerTarget string `yaml:"-"`

	// Mu is the mutex lock for the current node
	Mu *sync.Mutex `yaml:"-"`

	// SellerTarget is the item that the seller is currently selling
	SellerTarget string `yaml:"-"`

	// Latency is the culmulative response time for all requests
	LatencyLookup float64 `yaml:"-"`

	// RequestCount is the number of RPC calls submitted by the client
	RequestCountLookup int `yaml:"-"`

	// Latency is the culmulative response time for all requests
	LatencyRemote float64 `yaml:"-"`

	// RequestCount is the number of RPC calls submitted by the client
	RequestCountRemote int `yaml:"-"`

	// Latency is the culmulative response time for all requests
	LatencyLocal float64 `yaml:"-"`

	// RequestCount is the number of RPC calls submitted by the client
	RequestCountLocal int `yaml:"-"`
}

// ItemAmount is an item, associated amount, and an Unlimited setting. If
// unlimited is set to true, then the amount is ignored and the item is treated
// as unlimited.
type ItemAmount struct {
	Item      string `yaml:"item"`
	Amount    int    `yaml:"amount"`
	Unlimited bool   `yaml:"unlimited"`
}

// Peer holds a peerID and an address.
type Peer struct {
	PeerID int
	Addr   string
}
