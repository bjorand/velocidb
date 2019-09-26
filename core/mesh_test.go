package core

import (
	"testing"
	"time"
)

func TestMesh(t *testing.T) {
	m := newMesh()
	go m.registrator()
	p1, err := NewPeer("127.0.0.1", 64001)
	if err != nil {
		t.Error(err)
	}
	p2, err := NewPeer("127.0.0.1", 64002)
	if err != nil {
		t.Error(err)
	}
	m.register <- p1
	m.register <- p2
	time.Sleep(100 * time.Millisecond)
	output := len(m.Peers)
	expected := 2
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}

}
