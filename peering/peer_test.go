package peering

import (
	"testing"
)

func TestPeering(t *testing.T) {
	var (
		err error
	)

	p1, err := NewPeer("127.0.0.1", 64001)
	if err != nil {
		t.Error(err)
	}
	go p1.Run()
	output := len(p1.mesh.peers)
	expected := 0
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}

// 	p2, err := NewPeer("127.0.0.1", 64002)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	go p2.Run()
// 	time.Sleep(1 * time.Second)
// 	err = p1.ConnectToPeerAddr("127.0.0.1:64002")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	time.Sleep(5 * time.Second)
// 	output := p1.ConnectionStatus()
// 	expected := PEER_STATUS_TEXT[PEER_STATUS_CONNECTED]
// 	if expected != output {
// 		t.Errorf("want %+v, got %+v", expected, output)
// 	}
// 	output = p2.ConnectionStatus()
// 	expected = PEER_STATUS_TEXT[PEER_STATUS_CONNECTED]
// 	if expected != output {
// 		t.Errorf("want %+v, got %+v", expected, output)
// 	}
// }
