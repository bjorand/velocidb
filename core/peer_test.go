package core

import (
	"testing"
	"time"
)

func TestPeering(t *testing.T) {
	var (
		err error
		q   *Query
	)
	client1 := setup()

	input := []byte("0\r\n*2\r\n$6\r\nid=foo\r\n$5\r\nbar\r\n\r\n")
	q, err = client1.vqlTCPServer.Peer.ParsePeerQuery(client1, input)
	if err != nil {
		t.Fatal(err)
	}
	outputS := q.id
	expectedS := "foo"
	if outputS != expectedS {
		t.Fatalf("want %+v, got %+v", expectedS, outputS)
	}
	output := q.raw
	expected := []byte("bar\r\n")
	if string(output) != string(expected) {
		t.Fatalf("want %+v, got %+v", expected, output)
	}

	outputInt := len(client1.vqlTCPServer.Peer.Mesh.Peers)
	expectedInt := 0
	if expectedInt != outputInt {
		t.Errorf("want %+v, got %+v", expectedInt, outputInt)
	}

	p2, err := NewPeer("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	go p2.Run()
	vqlTCPServer2, err := NewVQLTCPServer(p2, "localhost", 0)
	if err != nil {
		panic(err)
	}
	go vqlTCPServer2.Run()
	client2 := NewVQLClient(1, "test-client-2", nil, vqlTCPServer2)
	time.Sleep(100 * time.Millisecond)
	expectedP2, err := client1.vqlTCPServer.Peer.ConnectToPeerAddr(p2.connString())
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 50; i++ {
		if expectedP2.ConnectionStatus() == PEER_STATUS_CONNECTED {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	expectedInt = PEER_STATUS_CONNECTED
	outputInt = expectedP2.ConnectionStatus()
	if expectedInt != outputInt {
		t.Fatalf("want %+v, got %+v", expectedInt, outputInt)
	}

	outputInt = len(client1.vqlTCPServer.Peer.Mesh.Peers)
	expectedInt = 1
	if expectedInt != outputInt {
		t.Fatalf("want %+v, got %+v", expectedInt, outputInt)
	}
	outputInt = len(client2.vqlTCPServer.Peer.Mesh.Peers)
	expectedInt = 1
	if expectedInt != outputInt {
		t.Fatalf("want %+v, got %+v", expectedInt, outputInt)
	}

	input = []byte("incr a\r\n")
	q, err = client1.ParseRawQuery(input)
	if err != nil {
		t.Fatalf("Cannot parse raw query: %+v", err)
	}
	_, err = q.Execute()
	if err != nil {
		t.Fatalf("Cannot execute query: %+v", err)
	}
	time.Sleep(400 * time.Millisecond)
	input = []byte("get a\r\n")
	q2, err := client2.ParseRawQuery(input)
	if err != nil {
		t.Fatalf("Cannot parse raw query: %+v", err)
	}
	r, err := q2.Execute()
	if err != nil {
		t.Fatalf("Cannot execute query: %+v", err)
	}
	expectedB := "$1\r\n1\r\n"
	outputB := r.FormattedPayload()
	if expectedB != string(outputB) {
		t.Fatalf("want %+v, got %+v", expectedB, string(outputB))
	}

}
