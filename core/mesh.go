package core

type Mesh struct {
	Peers      map[*Peer]bool
	register   chan *Peer
	deregister chan *Peer
}

func newMesh() *Mesh {
	return &Mesh{
		Peers:      make(map[*Peer]bool),
		register:   make(chan *Peer),
		deregister: make(chan *Peer),
	}
}

func (m *Mesh) registrator() {
	for {
		select {
		case p := <-m.register:
			m.Peers[p] = true
			// go p.getRemoteID(remotePeer)
		case p := <-m.deregister:
			delete(m.Peers, p)
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
