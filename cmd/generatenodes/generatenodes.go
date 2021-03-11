package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rjected/bazaar/nodeconfig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	config           string
	netConf          NetworkConfig
	verbose          bool
	GenerateNodesCmd = &cobra.Command{
		Use:   "generatenodes",
		Short: "generatenodes generates a network configuration for the bazaar based on various constraints.",
		Long:  `Generatenodes takes as input a set of nodes, network constraints, a list of edges to include, and a list of edges to exclude. The output is a folder of bazaar configuration files which follows the network constrants and edge specifications.`,

		// Parse the config if one is provided.
		PersistentPreRun: func(ccmd *cobra.Command, args []string) {

			// if --config is passed, attempt to parse the config file
			if config != "" {

				// get the filepath
				abs, err := filepath.Abs(config)
				if err != nil {
					log.Fatal("Error reading filepath: ", err.Error())
				}

				// get the config name
				base := filepath.Base(abs)

				// get the path
				path := filepath.Dir(abs)

				// add config path to viper for searching
				viper.SetConfigName(strings.Split(base, ".")[0])
				viper.AddConfigPath(path)

				// Find and read the config file; Handle errors reading the config file
				if err := viper.ReadInConfig(); err != nil {
					log.Fatal("Failed to read config file: ", err.Error())
					os.Exit(1)
				}

				err = viper.UnmarshalExact(&netConf)
				if err != nil {
					log.Fatal("Unable to unmarshal network config: ", err.Error())
				}
			}
		},

		Run: func(ccmd *cobra.Command, args []string) {
		},
	}
)

func init() {
	var err error

	GenerateNodesCmd.PersistentFlags().StringVar(&config, "config", "", "config file (default is ./generatenodes.yaml)")
	GenerateNodesCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	GenerateNodesCmd.MarkFlagRequired("config")

	err = GenerateNodesCmd.Execute()
	if err != nil {
		log.Fatal("Error running generatenodes: ", err.Error())
	}

}

func main() {
	if verbose {
		log.Printf("The whole yaml: %v\n", netConf)
	}

	if len(netConf.StaticNodes) > netConf.N {
		log.Fatal("The number of pre configured nodes exceeds the maximum number of nodes, please change N or the number of staticNodes")
	}

	log.Printf("Creating %d nodes to satisfy N", netConf.N-len(netConf.StaticNodes))
	// for k, v := range netConf.StaticNodes {
	// 	log.Printf("node %d exists with values %v\n", k, v)
	// }

	remainingNodes := netConf.N - len(netConf.StaticNodes)

	var nodeName int
	var ok bool
	for i := 0; i < remainingNodes; i++ {
		// make sure that the nodeName is not already in the map
		_, ok = netConf.StaticNodes[nodeName]
		if !ok {
			nodeName++
		}
		for ok {
			nodeName++
			_, ok = netConf.StaticNodes[nodeName]
		}
		netConf.StaticNodes[nodeName] = nodeconfig.NodeConfig{
			NodeID:  nodeName,
			Role:    "none",
			MaxHops: netConf.MaxHops,
		}
	}

	// set ports starting at 100000, also make peer maps
	// also make sure nodeid is set
	nodePort := 10000
	for k := range netConf.StaticNodes {
		// TODO: change lock to &sync.Mutex
		temp := netConf.StaticNodes[k]
		temp.NodePort = nodePort
		temp.Peers = make(map[int]string)
		temp.NodeID = k
		nodePort++
		netConf.StaticNodes[k] = temp
	}

	// for k, v := range netConf.StaticNodes {
	// 	log.Printf("22222 node %d exists with values %v\n", k, v)
	// }

	var err error
	// make sure all edges make sense
	err = CheckEdges(netConf.IncludeEdges, netConf.StaticNodes)
	if err != nil {
		log.Fatalf("Error checking include edges: %s", err)
	}

	err = CheckEdges(netConf.ExcludeEdges, netConf.StaticNodes)
	if err != nil {
		log.Fatalf("Error checking exclude edges: %s", err)
	}

	// TODO: check that includeedges and excludeedges do not contain the same
	// edge

	// Start including edges, this is bidirectional
	for _, edge := range netConf.IncludeEdges {
		// TODO: configure host in config, for now just localhost
		node1 := netConf.StaticNodes[edge[0]]
		node2 := netConf.StaticNodes[edge[1]]
		node2.Peers[node1.NodeID] = net.JoinHostPort("localhost", strconv.Itoa(node1.NodePort))
		node1.Peers[node2.NodeID] = net.JoinHostPort("localhost", strconv.Itoa(node2.NodePort))
		netConf.StaticNodes[edge[0]] = node1
		netConf.StaticNodes[edge[1]] = node2
	}

	// create an exclusion map
	excluded := make(map[[2]int]bool)
	for _, edge := range netConf.ExcludeEdges {
		excluded[edge] = true
		// flipped edge as well
		excluded[[2]int{edge[1], edge[0]}] = true
	}

	if verbose {
		for k, v := range netConf.StaticNodes {
			log.Printf("node %d exists with values %v\n", k, v)
		}
	}

	// connect edges. for this, we are going to pick nodes which don't have
	// enough peers (<k) and fill them up, making sure to avoid edges we have
	// excluded.
	for id1, node1 := range netConf.StaticNodes {
		// if it has enough peers we can move on
		if len(node1.Peers) >= netConf.K {
			continue
		}

		// otherwise, add edges
		for id2, node2 := range netConf.StaticNodes {
			// don't add self
			if id1 == id2 {
				continue
			}

			// if we added enough edges to node1 we can break
			if len(node1.Peers) >= netConf.K {
				break
			}

			// if has too many edges already, skip
			if len(node2.Peers) >= netConf.K {
				continue
			}
			// if excluded, skip
			_, ok := excluded[[2]int{id1, id2}]
			if ok {
				continue
			}

			// we are all set to create this edge then
			// TODO: turn this monolith into something stateful
			node2.Peers[node1.NodeID] = net.JoinHostPort("localhost", strconv.Itoa(node1.NodePort))
			node1.Peers[node2.NodeID] = net.JoinHostPort("localhost", strconv.Itoa(node2.NodePort))
			netConf.StaticNodes[id1] = node1
			netConf.StaticNodes[id2] = node2
		}
	}

	if verbose {
		for k, v := range netConf.StaticNodes {
			log.Printf("node %d exists with values %v\n", k, v)
		}
	}

	log.Printf("Writing files...")
	// create list of static nodes
	var nodes []nodeconfig.NodeConfig
	for k := range netConf.StaticNodes {
		nodes = append(nodes, netConf.StaticNodes[k])
	}
	writeFiles(nodes, netConf.OutputDir)
	log.Printf("Network generation has is done!")

}

// writeFiles writes the nodes to a file in the specified folder.
func writeFiles(nodelist []nodeconfig.NodeConfig, folder string) {
	for _, node := range nodelist {
		yamlString, err := yaml.Marshal(&node)
		if err != nil {
			log.Fatalf("yaml.Marshal failed with '%s'\n", err)
		}
		file, err := os.Create(fmt.Sprintf("%s/node%d.yml", folder, node.NodeID))
		if err != nil {
			log.Fatalf("os.Create failed with '%s'\n", err)
		}
		_, err = file.Write(yamlString)
		if err != nil {
			log.Fatalf("Write failed with '%s'\n", err)
		}
	}
}

// CheckEdges checks that edges exist in the static node list
func CheckEdges(edges [][2]int, staticNodes map[int]nodeconfig.NodeConfig) error {
	var ok bool
	for _, edge := range edges {
		_, ok = staticNodes[edge[0]]
		if !ok {
			return fmt.Errorf("the node %d, in edge %v, does not exist in the static node list", edge[0], edge)
		}
		_, ok = staticNodes[edge[1]]
		if !ok {
			return fmt.Errorf("the node %d, in edge %v, does not exist in the static node list", edge[0], edge)
		}
	}
	return nil
}
