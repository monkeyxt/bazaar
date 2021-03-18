package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rjected/bazaar/nodeconfig"
	"gopkg.in/yaml.v2"
)

// BazaarNode contains the state for the node.
type BazaarNode struct {
	config        nodeconfig.NodeConfig
	sellerChannel chan nodeconfig.Peer

	// peerClients is a map from a peerID to an rpc Client that we use for
	// communicating with that peer.
	peerClients    map[int]*rpc.Client
	peerClientLock *sync.Mutex
	VerboseLogging bool
	PerfLogger     *log.Logger
	lookupUUID     int
	uuidLock       *sync.Mutex
	perfMap        map[int][]time.Time
	perfLock       *sync.Mutex
}

// BazaarServer exposes methods for letting a node listen for RPC
type BazaarServer struct {
	node *BazaarNode
}

// SeedMathRand uses crypto randomness to seed the math prng. This is so we can
// seem more random when not testing, and be deterministic when testing.
// rand.Seed uses a seed of 1 when not explicitly seeded.
func SeedMathRand() error {
	var readInt [8]byte
	_, err := crand.Read(readInt[:])
	if err != nil {
		return fmt.Errorf("error reading from crypto rand for seeding: %s", err)
	}

	seed := binary.BigEndian.Uint64(readInt[:])
	rand.Seed(int64(seed))

	return nil
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

	if node.config.Role == "random" {
		randRole := rand.Intn(4)
		switch randRole {
		case 0:
			node.config.Role = "buyer"
			node.config.BuyerOptionList = GenerateRandomItems()
		case 1:
			node.config.Role = "seller"
			node.config.Items = CreateRandomSellerList(64)
		case 2:
			node.config.Role = "both"
			node.config.BuyerOptionList = GenerateRandomItems()
			node.config.Items = CreateRandomSellerList(64)
		case 3:
			node.config.Role = "none"
		}

	}

	if node.config.Role == "seller" || node.config.Role == "both" {
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
			log.Println("NO ITEMS AVAILABLE! Picking no items...")
		} else {
			// pick item at random from the list of available items
			randItemIdx = rand.Intn(len(availableItems))
			node.config.SellerTarget = availableItems[randItemIdx]
		}

	}

	// initialize the map for peer clients
	node.peerClients = make(map[int]*rpc.Client)
	node.peerClientLock = &sync.Mutex{}
	node.config.Mu = &sync.Mutex{}
	node.uuidLock = &sync.Mutex{}
	node.perfMap = make(map[int][]time.Time)
	node.perfLock = &sync.Mutex{}

	// initialize the seller channel, just have 100 max for now
	node.sellerChannel = make(chan nodeconfig.Peer, 100)

	return &node, nil
}

// CreateRandomSellerList creates a list of random items, with amounts, for a
// seller.
func CreateRandomSellerList(maxItems int) []nodeconfig.ItemAmount {
	possibleItems := []string{"salt", "fish", "boars"}

	// generates the items to potentially pick.
	pickItems := rand.Perm(len(possibleItems))

	// generates from [1,len(possibleItems)+1)
	numItems := rand.Intn(len(possibleItems)) + 1
	items := make([]nodeconfig.ItemAmount, numItems)

	for idx, item := range items {
		item.Item = possibleItems[pickItems[idx]]

		// max 64
		item.Amount = rand.Intn(maxItems)
		if rand.Uint64()%2 == 1 {
			item.Unlimited = true
		} else {
			item.Unlimited = false
		}
		items[idx] = item
	}

	return items

}

// GenerateRandomItems creates a list of random items for a buyer
func GenerateRandomItems() []string {

	possibleItems := []string{"salt", "fish", "boars"}

	// generates the items to potentially pick.
	pickItems := rand.Perm(len(possibleItems))

	// generates from [1,len(possibleItems))
	numItems := rand.Intn(len(possibleItems)) + 1
	items := make([]string, numItems)

	for idx := range items {
		items[idx] = possibleItems[pickItems[idx]]
	}

	return items
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
	Route       []nodeconfig.Peer
	UUID        int
}

// LookupResponse is empty because no response is required for lookup.
type LookupResponse struct {
}

// Lookup runs the lookup command.
func (bnode *BazaarNode) Lookup(args LookupArgs, reply *LookupResponse) error {
	// log.Printf("Node %d is looking for %d with lookup for %s", bnode.config.NodeID, args.BuyerID, args.ProductName)
	return bnode.lookupProduct(args.Route, args.ProductName, args.HopCount, args.BuyerID, args.UUID)
}

// lookupProduct takes in a product name and hopcount, and runs the lookup procedure.
func (bnode *BazaarNode) lookupProduct(route []nodeconfig.Peer, productName string, hopcount int, buyerID int, uuid int) error {

	// Add the current node to the routelist
	portStr := net.JoinHostPort(bnode.config.NodeIP, strconv.Itoa(bnode.config.NodePort))
	route = append(route, nodeconfig.Peer{PeerID: bnode.config.NodeID, Addr: portStr})

	// Reached a seller with the desired product. Send a reply.
	if (bnode.config.Role == "seller" || bnode.config.Role == "both") && (bnode.config.SellerTarget == productName) {
		if bnode.VerboseLogging {
			log.Printf("Seller has found a buyer! Replying to %d along route %v\n", buyerID, route)
		}
		go bnode.reply(route, nodeconfig.Peer{PeerID: bnode.config.NodeID, Addr: net.JoinHostPort(bnode.config.NodeIP, strconv.Itoa(bnode.config.NodePort))}, uuid)
	}

	// log.Printf("Node %d received lookup request from %d\n", bnode.config.NodeID, buyerID)
	if hopcount == 0 {
		if bnode.VerboseLogging {
			log.Printf("Node %d is discarding lookup request for %s\n", bnode.config.NodeID, productName)
		}
		return nil
	}

	// log.Printf("Node %d flooding peers with lookup requests for %s from %d...\n", bnode.config.NodeID, productName, buyerID)
	for peer, addr := range bnode.config.Peers {

		// Make sure that we are not flooding the node where we came from
		peerInRoute := false
		for _, routePeer := range route {
			if peer == routePeer.PeerID && hopcount < bnode.config.MaxHops {
				peerInRoute = true
				break
			}
		}
		if peerInRoute {
			continue
		}

		// Flood the other peers
		// log.Printf("Node %d is flooding peer %d for lookup\n", bnode.config.NodeID, peer)
		go bnode.callLookupRPC(route, nodeconfig.Peer{PeerID: peer, Addr: addr}, productName, hopcount, buyerID, uuid)

	}

	// log.Printf("Node %d is done flooding peers with lookups for %s from %d...\n", bnode.config.NodeID, productName, buyerID)

	return nil
}

// Reply relays the message back to the buyer
func (bnode *BazaarNode) Reply(args ReplyArgs, reply *ReplyResponse) error {
	// if len(args.RouteList) == 1 {
	// 	log.Printf("Message at final hop: node %v with message from seller node %d", args.RouteList[len(args.RouteList)-1], args.SellerInfo.PeerID)
	// } else {
	// 	log.Printf("Forward reply to node %v with message from seller node %d", args.RouteList[len(args.RouteList)-2], args.SellerInfo.PeerID)
	// }

	return bnode.reply(args.RouteList, args.SellerInfo, args.LookupUUID)
}

// ReplyArgs contains the RPC arguments for reply, which is the backtracking list
// and the sellerid to be returned
type ReplyArgs struct {
	RouteList  []nodeconfig.Peer
	SellerInfo nodeconfig.Peer
	LookupUUID int
}

// ReplyResponse is empty because no response is required.
type ReplyResponse struct {
}

// Reply message with the peerId of the seller
func (bnode *BazaarNode) reply(routeList []nodeconfig.Peer, sellerInfo nodeconfig.Peer, lookupUUID int) error {

	// routeList: a list of ids to traverse back to the original sender in the format of
	//         [1, 5, 2, 6], so the reverse traversal path should be 6 --> 2 --> 5 --> 1

	// sellerID: id of the seller who responds

	if len(routeList) == 1 {

		// Reached original sender, add the sellerID to a list for the buyer to randomly
		// choose from.
		// log.Printf("Node %d got a match reply from node %d ", bnode.config.NodeID, sellerInfo.PeerID)

		// first seller
		bnode.sellerChannel <- nodeconfig.Peer{PeerID: sellerInfo.PeerID, Addr: sellerInfo.Addr}

	} else {

		var recipient nodeconfig.Peer
		recipient, routeList = routeList[len(routeList)-2], routeList[:len(routeList)-1]

		// log.Printf("Sending reply RPC to %s\n", recipient.Addr)
		go bnode.callReplyRPC(recipient, routeList, sellerInfo, lookupUUID)

	}

	return nil
}

// TransactionArgs contains the RPC arguments for buy. CurrentTarget is the
// what the buyer wishes to buy during this transaction
type TransactionArgs struct {
	CurrentTarget string
	BuyerID       int
}

// TransactionResponse is empty for now
type TransactionResponse struct {
}

// Buy item directly from the seller with RCP call
func (bnode *BazaarNode) buy(seller nodeconfig.Peer) error {

	// log.Printf("Node %d buying from seller node %d", bnode.config.NodeID, seller.PeerID)
	go bnode.callSellRPC(seller)

	return nil

}

// Sell runs the sell command
func (bnode *BazaarNode) Sell(args TransactionArgs, reply *TransactionResponse) error {
	if bnode.VerboseLogging {
		log.Printf("Seller node %d selling item %s", bnode.config.NodeID, args.CurrentTarget)
	}
	return bnode.sell(args.CurrentTarget, args.BuyerID)
}

func (bnode *BazaarNode) sell(target string, buyerID int) error {

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
		log.Printf("💰💰💰 Node %d sold %s to %d, amount remaining %d 💰💰💰", bnode.config.NodeID, target, buyerID, bnode.config.Items[targetID].Amount)

	} else {

		// If the item is defined to be unlimited in the YAML file restock the item and purchase again
		if bnode.config.Items[targetID].Unlimited {

			bnode.config.Items[targetID].Amount += 10
			if bnode.VerboseLogging {
				log.Printf("Seller node %d restocked %s", bnode.config.NodeID, bnode.config.Items[targetID].Item)
			}

			bnode.config.Items[targetID].Amount--
			log.Printf("💰💰💰 Node %d sold %s to %d, amount remaining %d 💰💰💰", bnode.config.NodeID, target, buyerID, bnode.config.Items[targetID].Amount)

		} else {

			// Item sold out. Pick another item randomly to sell
			var commodity []string
			for itemID := range bnode.config.Items {
				if bnode.config.Items[itemID].Amount > 0 {
					commodity = append(commodity, bnode.config.Items[itemID].Item)
				}
			}

			// only select from random if there are things to select
			if len(commodity) > 0 {
				bnode.config.SellerTarget = commodity[rand.Intn(len(commodity))]
				log.Printf("Seller node %d now selling %s", bnode.config.NodeID, bnode.config.SellerTarget)
			} else {
				log.Printf("Seller node %d is out of items!\n", bnode.config.NodeID)
			}
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

// init is the entrance point for all nodes
func (bnode *BazaarNode) init() {
	if bnode.config.Role == "buyer" || bnode.config.Role == "both" {
		bnode.buyerLoop()
	}
}

// buyerLoop is the lookup/buy loop for the buyer
func (bnode *BazaarNode) buyerLoop() {
	// wait before starting the buyer loop
	time.Sleep(5 * time.Second)

	for {

		// Generate a buy request
		if len(bnode.config.BuyerOptionList) != 0 {
			bnode.config.BuyerTarget = bnode.config.BuyerOptionList[rand.Intn(len(bnode.config.BuyerOptionList))]
			if bnode.VerboseLogging {
				log.Printf("Node %d plans to buy %s", bnode.config.NodeID, bnode.config.BuyerTarget)
			}

		}

		// Lookup request to neighbours
		// portStr := net.JoinHostPort(bnode.config.NodeIP, strconv.Itoa(bnode.config.NodePort))
		lookupUUID := bnode.GetLookupUUID()
		args := LookupArgs{
			ProductName: bnode.config.BuyerTarget,
			HopCount:    bnode.config.MaxHops,
			BuyerID:     bnode.config.NodeID,
			Route:       []nodeconfig.Peer{},
			UUID:        lookupUUID,
		}

		var rpcResponse LookupResponse
		startTime := time.Now()
		go bnode.Lookup(args, &rpcResponse)
		// log.Printf("Waiting to retrieve sellers...")

		// Buy from the list of available sellers
		time.Sleep(200 * time.Millisecond)

		// NOTE: we are assuming here that this is the corresponding reply for
		// the previously issued lookup request
		endTime, err := bnode.GetEarliestLookup(lookupUUID)
		if err == nil {
			bnode.reportLookupLatency(startTime, endTime)
		} else {
			// log.Println("Not reporting latency, no data")
		}

		var tempSellerList []nodeconfig.Peer
		for i := 0; i < len(bnode.sellerChannel); i++ {
			tempSellerList = append(tempSellerList, <-bnode.sellerChannel)
		}

		// dedupe seller list
		var sellerList []nodeconfig.Peer
		peerMap := make(map[int]nodeconfig.Peer)
		for _, peer := range tempSellerList {
			_, ok := peerMap[peer.PeerID]
			if !ok {
				peerMap[peer.PeerID] = peer
				sellerList = append(sellerList, peer)
			}
		}

		if len(sellerList) != 0 {
			replyString := fmt.Sprintf("Node %d Received replies from ", bnode.config.NodeID)
			for i, seller := range sellerList {
				if i == 0 {
					replyString += strconv.Itoa(seller.PeerID)
				} else if i == len(sellerList)-1 {
					replyString += ", and " + strconv.Itoa(seller.PeerID)
				} else {
					replyString += ", " + strconv.Itoa(seller.PeerID)
				}
			}
			log.Println(replyString)

			randomSeller := sellerList[rand.Intn(len(sellerList))]
			go bnode.buy(randomSeller)
			log.Printf("Node %d bought %s from seller node %d", bnode.config.NodeID, bnode.config.BuyerTarget, randomSeller.PeerID)
		}

	}

}

// reportLatency logs the average latency of RPC calls every 50 invocations
func (bnode *BazaarNode) reportRPCLatency(start time.Time, end time.Time, peer string) {

	peerIP := strings.Split(peer, ":")[0]

	if peerIP == bnode.config.NodeIP {

		durationFloat64 := end.Sub(start).Seconds()
		bnode.config.LatencyLocal += durationFloat64
		bnode.config.RequestCountLocal += 1

		if bnode.config.RequestCountLocal%500 == 0 {
			averageLatency := bnode.config.LatencyLocal / float64(bnode.config.RequestCountLocal)
			log.Printf("👽👽👽 Average Local RPC Latency of peer %d： %f 👽👽👽", bnode.config.NodeID, averageLatency)
		}

	} else {

		durationFloat64 := end.Sub(start).Seconds()
		bnode.config.LatencyRemote += durationFloat64
		bnode.config.RequestCountRemote += 1

		if bnode.config.RequestCountRemote%500 == 0 {
			averageLatency := bnode.config.LatencyRemote / float64(bnode.config.RequestCountRemote)
			log.Printf("👽👽👽 Average Remote RPC Latency of peer %d： %f 👽👽👽", bnode.config.NodeID, averageLatency)
		}

	}

}

// reportLatency logs the average latency of RPC calls every 50 invocations
func (bnode *BazaarNode) reportLookupLatency(start time.Time, end time.Time) {

	durationFloat64 := end.Sub(start).Seconds()
	bnode.config.LatencyLookup += durationFloat64
	bnode.config.RequestCountLookup += 1

	averageLatency := bnode.config.LatencyLookup / float64(bnode.config.RequestCountLookup)
	bnode.PerfLogger.Printf("%f", averageLatency)

}
