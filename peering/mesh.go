package peering

type mesh struct {
	peers      map[*Peer]bool
	register   chan *Peer
	deregister chan *Peer
}

func newMesh() *mesh {
	return &mesh{
		peers:      make(map[*Peer]bool),
		register:   make(chan *Peer, 2),
		deregister: make(chan *Peer, 2),
	}
}

func (m *mesh) registrator() {
	for {
		select {
		case p := <-m.register:
			m.peers[p] = true

		case p := <-m.deregister:
			delete(m.peers, p)
		default:
		}
	}
}
