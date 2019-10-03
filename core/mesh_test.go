package core

import (
	"testing"
)

func TestMesh(t *testing.T) {
	m := newMesh()
	go m.registrator()
	p1, err := NewPeer("127.0.0.1", 0)
	if err != nil {
		t.Error(err)
	}
	p2, err := NewPeer("127.0.0.1", 0)
	if err != nil {
		t.Error(err)
	}
	m.register <- p1
	m.register <- p2
	output := len(m.Peers)
	expected := 2
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}

}
