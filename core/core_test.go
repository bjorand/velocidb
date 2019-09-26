package core

var (
	testPeer *Peer
)

func setup() {
	var err error
	testPeer, err = NewPeer("localhost", 26000)
	if err != nil {
		panic(err)
	}
	// var err error
	client = &VQLClient{}
	client.vqlTCPServer, err = NewVQLTCPServer(testPeer, "localhost", 26001)
	if err != nil {
		panic(err)
	}
}
