package peering

import (
	"fmt"
	"log"
	"net"
	"time"

	tcp "github.com/bjorand/velocidb/tcp"
	utils "github.com/bjorand/velocidb/utils"
	"github.com/google/uuid"
)

const (
	PEER_STATUS_DISCONNECTED = 1
	PEER_STATUS_CONNECTED    = 1
)

type Stats struct {
	bytesIn                      int64
	bytesOut                     int64
	connectionReadFailureCounter int64
}

type Peer struct {
	ID         string
	Tags       []string
	RemoteConn net.Conn
	Protocol   string
	Height     int64
	Peering    *Peering
	ListenPort int64
	ListenAddr string
	Stats      *Stats
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
		Stats:      &Stats{},
	}, nil
}

func (p *Peer) connString() string {
	return fmt.Sprintf("%s:%d", p.ListenAddr, p.ListenPort)

}

func (p *Peer) ConnectToPeerAddr(peerConnString string) error {
	peerAddr, peerPort, err := utils.SplitHostPort(peerConnString)
	if err != nil {
		return err
	}
	newPeer := &Peer{
		ListenAddr: peerAddr,
		ListenPort: peerPort,
	}
	go p.ConnectToPeer(newPeer)
	return nil
}

func (p *Peer) ConnectToPeer(newPeer *Peer) {
	initialPause := 2
	maxPause := 60
	pause := initialPause
	p.Peering.Peers = append(p.Peering.Peers, newPeer)
	for {
		time.Sleep(time.Duration(pause) * time.Second)
		conn, err := net.Dial("tcp4", newPeer.connString())
		if err != nil {
			fmt.Printf("[peer %s] %s\n", newPeer.connString(), err)
			if pause < maxPause {
				pause += 2
			}
			continue
		}
		fmt.Printf("[peer %s] Connected\n", newPeer.connString())
		newPeer.RemoteConn = conn
		pause = initialPause
		defer conn.Close()
		for {
			reply := make([]byte, 1024)
			n, err := conn.Read(reply)
			if err != nil {
				fmt.Printf("[peer %s] Read from peer failed: %s\n", newPeer.connString(), err.Error())
				conn.Close()
				p.Stats.connectionReadFailureCounter++
				pause = initialPause
				break
			}
			fmt.Printf("[peer %s] %s\n", newPeer.connString(), reply[:n])
			// resp, err := vql.ParseRawResponse(reply[:n])
			// if err != nil {
			// 	fmt.Println(err)
			// 	continue
			// }
			// if resp.DisconnectSignal {
			// 	break
			// }
			// fmt.Println(resp.Payload)
		}
	}
}

func (p *Peer) Run() {
	s, err := tcp.NewTCPServer(p.ListenAddr, p.ListenPort)
	if err != nil {
		panic(err)
	}
	s.Run("peer", p.HandlePeerRequest)
}

func (p *Peer) HandlePeerRequest(s *tcp.TCPServer, conn net.Conn) {
	fmt.Printf("[peer] Serving %s\n", conn.RemoteAddr().String())
	// Make a buffer to hold incoming data.
	for {
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("[peer] Error reading:", err.Error())
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
	fmt.Printf("[peer] Connection closed %s\n", conn.RemoteAddr().String())
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
