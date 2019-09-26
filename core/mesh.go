package core

type Mesh struct {
	Peers      map[*Peer]bool
	register   chan *Peer
	deregister chan *Peer
}

func newMesh() *Mesh {
	return &Mesh{
		Peers:      make(map[*Peer]bool),
		register:   make(chan *Peer, 2),
		deregister: make(chan *Peer, 2),
	}
}

func (m *Mesh) registrator() {
	for {
		select {
		case p := <-m.register:
			m.Peers[p] = true

		case p := <-m.deregister:
			delete(m.Peers, p)
		default:
		}
	}
}

func (m *Mesh) GetPeerByKey(key string) *Peer {
	for p := range m.Peers {
		if p.Key() == key {
			return p
		}
	}
	return nil
}
