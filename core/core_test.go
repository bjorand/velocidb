package core

func setup() *VQLClient {
	var err error
	peer1, err := NewPeer("localhost", 0)
	if err != nil {
		panic(err)
	}
	go peer1.Run()

	vqlTCPServer, err := NewVQLTCPServer(peer1, "localhost", 0)
	if err != nil {
		panic(err)
	}
	go vqlTCPServer.Run()

	client := NewVQLClient(1, "test-client-1", nil, vqlTCPServer)

	return client
}
