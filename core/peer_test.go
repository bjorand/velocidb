package core

import (
	"fmt"
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
	outputInt := len(p1.Mesh.Peers)
	expectedInt := 0
	if expectedInt != outputInt {
		t.Errorf("want %+v, got %+v", expectedInt, outputInt)
	}

	p2, err := NewPeer("127.0.0.1", 64002)
	if err != nil {
		t.Error(err)
	}
	go p2.Run()
	err = p1.ConnectToPeerAddr("127.0.0.1:64002")
	if err != nil {
		t.Error(err)
	}
	// time.Sleep(10 * time.Second)
	output := p1.ConnectionStatus()
	expected := PEER_STATUS_CONNECTED
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	output = p2.ConnectionStatus()
	expected = PEER_STATUS_CONNECTED
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	fmt.Println(p1.Mesh.Peers)
	fmt.Println(p2.Mesh.Peers)

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
