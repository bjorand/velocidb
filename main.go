package main

import (
	peering "github.com/bjorand/velocidb/peering"
	vql "github.com/bjorand/velocidb/vql"
)

func main() {
	peer, err := peering.NewPeer("0.0.0.0", 4301)
	if err != nil {
		panic(err)
	}
	go peer.Run()
	defer peer.Shutdown()
	v, err := vql.NewVQLTCPServer(peer, "127.0.0.1", 4300)
	if err != nil {
		panic(err)
	}
	v.Run()
	defer v.Shutdown()
}
