package main

import (
	"github.com/rjected/bazaar/nodeconfig"
)

type NetworkConfig struct {
	N int `yaml:"N"`
	K int `yaml:"K"`

	// maxhops for the whole network
	MaxHops   int    `yaml:"maxHops"`
	OutputDir string `yaml:"outputDir"`

	// includeEdges is a list of edges to include in the network, from nodes
	// that have already been named and specified.
	// excludeEdges is a list of edges to specifically exclude, and not connect
	// in the network.
	ExcludeEdges [][2]int                      `yaml:"excludeEdges,inline"`
	IncludeEdges [][2]int                      `yaml:"includeEdges,inline"`
	StaticNodes  map[int]nodeconfig.NodeConfig `yaml:"staticNodes,inline"`
	Hosts        []string                      `yaml:"hosts"`
}
