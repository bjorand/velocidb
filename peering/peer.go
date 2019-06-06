package peering

import (
	"fmt"
	"log"
	"net"

	tcp "github.com/bjorand/velocidb/tcp"
	"github.com/google/uuid"
)

type Peer struct {
	ID         string
	Tags       []string
	RemoteConn net.Conn
	Protocol   string
	Height     int64
	Peering    *Peering
	ListenPort int64
	ListenAddr string
}

type Peering struct {
	Peers []*Peer
}

func NewPeer(listenAddr string, port int64) (*Peer, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	return &Peer{
		ID: id.String(),
		Peering: &Peering{
			Peers: make([]*Peer, 0),
		},
		ListenAddr: listenAddr,
		ListenPort: port,
	}, nil
}

func (p *Peer) ConnectToPeer(newPeer *Peer) error {
	p.Peering.Peers = append(p.Peering.Peers, newPeer)
	return nil
}

func (p *Peer) Run() {
	s, err := tcp.NewTCPServer(p.ListenAddr, p.ListenPort)
	if err != nil {
		panic(err)
	}
	s.Run(p.HandlePeerRequest)
}

func (p *Peer) HandlePeerRequest(s *tcp.TCPServer, conn net.Conn) {
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())
	// Make a buffer to hold incoming data.
	for {
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			break
		}
		// query, err := vql.ParseRawQuery(buf[:reqLen])
		// if err != nil {
		// 	conn.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		// 	continue
		// }
		// resp, err := query.Execute()
		// if err != nil {
		// 	conn.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		// 	continue
		// }
		// conn.Write([]byte(fmt.Sprintf("%s\n", resp.Payload)))
		// if resp.DisconnectSignal {
		// 	break
		// }
	}
	conn.Close()
	fmt.Printf("Connection closed %s\n", conn.RemoteAddr().String())
}

func (p *Peer) Shutdown() {
}

func HandlePeerRequest(s *tcp.TCPServer, conn net.Conn, peer *Peer) {
	fmt.Printf("Serving %s\n", conn.RemoteAddr().String())
	// Make a buffer to hold incoming data.
	for {
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			break
		}
		// query, err := vql.ParseRawQuery(buf[:reqLen])
		// if err != nil {
		// 	conn.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		// 	continue
		// }
		// resp, err := query.Execute()
		// if err != nil {
		// 	conn.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		// 	continue
		// }
		// conn.Write([]byte(fmt.Sprintf("%s\n", resp.Payload)))
		// if resp.DisconnectSignal {
		// 	break
		// }
	}
	conn.Close()
	fmt.Printf("Connection closed %s\n", conn.RemoteAddr().String())
}
