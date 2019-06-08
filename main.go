package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	peering "github.com/bjorand/velocidb/peering"
	utils "github.com/bjorand/velocidb/utils"
	vql "github.com/bjorand/velocidb/vql"
)

const (
	defaultListenPeer = "0.0.0.0:4301"
	defaultListenVQL  = "0.0.0.0:4300"
)

var (
	listenPeerFlag = flag.String("peer-listen", "", fmt.Sprintf("Peer server listen host:port (default: %s)", defaultListenPeer))
	listenVQLFlag  = flag.String("vql-listen", "", fmt.Sprintf("VQL server listen host:port (default: %s)", defaultListenVQL))
	peers          = flag.String("peers", "", "Lisf of peers addr:port,addr1:port")
)

type Config struct {
	listenPeer string
	listenVQL  string
	peersAddr  []string
}

func cleanPeersInput(input string) (peers []string) {
	for _, peerAddr := range strings.Split(input, ",") {
		peerAddr = strings.Trim(peerAddr, " ")
		peers = append(peers, peerAddr)
	}
	return peers
}

func (c *Config) SetDefault() {
	c.listenPeer = defaultListenPeer
	c.listenVQL = defaultListenVQL
}

func (c *Config) FromEnvironment() {
	for _, env := range os.Environ() {
		envArray := strings.Split(env, "=")
		envKey := envArray[0]
		envValue := envArray[1]
		// breakSwitch:
		switch envKey {
		case "PEER_LISTEN":
			c.listenPeer = envValue
		case "VQL_LISTEN":
			c.listenVQL = envValue
		case "PEERS":
			c.peersAddr = cleanPeersInput(envValue)
		}
	}
}

func (c *Config) FlagsOverride() {
	if *listenPeerFlag != "" {
		c.listenPeer = *listenPeerFlag
	}
	if *listenVQLFlag != "" {
		c.listenVQL = *listenVQLFlag
	}
	if *peers != "" {
		c.peersAddr = cleanPeersInput(*peers)
	}
}

func main() {
	flag.Parse()
	config := Config{}
	config.SetDefault()
	config.FromEnvironment()
	config.FlagsOverride()

	hostPeer, portPeer, err := utils.SplitHostPort(config.listenPeer)
	if err != nil {
		panic(err)
	}

	hostVQL, portVQL, err := utils.SplitHostPort(config.listenVQL)
	if err != nil {
		panic(err)
	}
	peer, err := peering.NewPeer(hostPeer, portPeer)
	if err != nil {
		panic(err)
	}
	go func() {
		for _, peerAddr := range config.peersAddr {
			peer.ConnectToPeerAddr(peerAddr)
		}

	}()

	go peer.Run()
	defer peer.Shutdown()
	v, err := vql.NewVQLTCPServer(peer, hostVQL, portVQL)
	if err != nil {
		panic(err)
	}
	v.Run()
	defer v.Shutdown()
}