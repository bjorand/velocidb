package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"

	peering "github.com/bjorand/velocidb/peering"
	utils "github.com/bjorand/velocidb/utils"
	vql "github.com/bjorand/velocidb/vql"
)

const (
	defaultListenPeer = "0.0.0.0:4301"
	defaultListenVQL  = "0.0.0.0:4300"
)

var (
	cpuprofile     = flag.String("cpuprofile", "", "write cpu profile to file")
	walDir         = flag.String("wal-dir", "/var/lib/velocidb/wals", "WAL storage directory")
	listenPeerFlag = flag.String("peer-listen", "", fmt.Sprintf("Peer server listen host:port (default: %s)", defaultListenPeer))
	listenVQLFlag  = flag.String("vql-listen", "", fmt.Sprintf("VQL server listen host:port (default: %s)", defaultListenVQL))
	peers          = flag.String("peers", "", "Lisf of peers addr:port,addr1:port")
	disableVQL     = flag.Bool("disable-vql", false, "Disable VQL server")
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
	signalChan := make(chan os.Signal, 1)
	quit := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGTERM)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGKILL)
	// signal.Notify(signalChan, os.)
	go func() {
		q := <-signalChan
		log.Printf("Signal %+v received", q)
		close(quit)
		return
	}()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		log.Println("CPU profiler enabled")
		defer pprof.StopCPUProfile()
	}
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

	if !*disableVQL {
		v, err := vql.NewVQLTCPServer(peer, hostVQL, portVQL)
		if err != nil {
			panic(err)
		}
		go v.Run()
		defer v.Shutdown()
	}
	<-quit
	log.Println("Clean shutdown done")
}
