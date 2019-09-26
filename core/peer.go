package core

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"time"

	tcp "github.com/bjorand/velocidb/tcp"
	utils "github.com/bjorand/velocidb/utils"
	"github.com/google/uuid"
)

const (
	PEER_STATUS_NO_CONNECTION = 0
	PEER_STATUS_CONNECTED     = 1
)

var (
	PEER_STATUS_TEXT = map[int64]string{
		PEER_STATUS_NO_CONNECTION: "No connection",
		PEER_STATUS_CONNECTED:     "Connected",
	}
)

type Stats struct {
	BytesIn                      int64
	BytesOut                     int64
	connectionReadFailureCounter int64
	connectionLastError          error
}

type Peer struct {
	ID               string
	Tags             []string
	RemoteConn       net.Conn
	Protocol         string
	Height           int64
	ListenPort       int64
	ListenAddr       string
	Stats            *Stats
	RemoveSignal     bool
	Name             string
	Mesh             *Mesh
	broadcastBulkVQL chan []byte
	vqlClient        *VQLClient
}

func NewPeer(listenAddr string, port int64) (*Peer, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	return &Peer{
		ID:               id.String(),
		ListenAddr:       listenAddr,
		ListenPort:       port,
		Stats:            &Stats{},
		Mesh:             newMesh(),
		broadcastBulkVQL: make(chan []byte, 1024),
	}, nil
}

func (p *Peer) connString() string {
	return fmt.Sprintf("%s:%d", p.ListenAddr, p.ListenPort)

}

func (p *Peer) Info() (info map[string]interface{}) {
	info = make(map[string]interface{})
	info["id"] = p.ID

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info["heap_usage"] = utils.HumanSizeBytes(int64(m.Alloc))
	info["total_heap_allocated"] = utils.HumanSizeBytes(int64(m.TotalAlloc))
	info["memory_usage"] = utils.HumanSizeBytes(int64(m.Sys))
	info["goroutines"] = runtime.NumGoroutine()
	return info

}

func (p *Peer) ConnectionStatus() int64 {
	if p.RemoteConn != nil {
		return PEER_STATUS_CONNECTED
	}
	return PEER_STATUS_NO_CONNECTION
}

func (p *Peer) ConnectToPeerAddr(peerConnString string) error {
	peerAddr, peerPort, err := utils.SplitHostPort(peerConnString)
	if err != nil {
		return err
	}
	newPeer := &Peer{
		ListenAddr: peerAddr,
		ListenPort: peerPort,
		Stats:      &Stats{},
	}
	go p.connectToPeer(newPeer)
	return nil
}

func (p *Peer) Key() string {
	return p.connString()
}

func (p *Peer) RemovePeer(dead *Peer) {
	dead.RemoveSignal = true
	if dead.RemoteConn != nil {
		dead.RemoteConn.Close()
	}
}

func (p *Peer) connectToPeer(newPeer *Peer) {
	defer func() {
		p.Mesh.deregister <- newPeer
	}()
	initialPause := 2
	maxPause := 60
	pause := initialPause
	for {
		if newPeer.RemoveSignal {
			break
		}
		newPeer.RemoteConn = nil
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
		p.Mesh.register <- newPeer
		pause = initialPause
		defer conn.Close()
		for {
			if newPeer.RemoveSignal {
				break
			}
			// if p.ID != "" {
			// 	fmt.Fprintf(conn, "PEER ID")
			// }
			reply := make([]byte, 1024)
			n, err := conn.Read(reply)
			newPeer.Stats.BytesIn += int64(n)
			if err != nil {
				fmt.Printf("[peer %s] Read from peer failed: %s\n", newPeer.connString(), err.Error())
				conn.Close()
				p.Stats.connectionReadFailureCounter++
				pause = initialPause
				break
			}

			// fmt.Printf("[peer %s] %s\n", newPeer.connString(), reply[:n])
			resp, err := p.ParsePeerResponse(newPeer, reply[:n])
			if err != nil {
				fmt.Println(err)
				continue
			}
			if resp.DisconnectSignal {
				break
			}
		}
	}
}

type PeerResponse struct {
	DisconnectSignal bool
}

func (p *Peer) ParsePeerResponse(from *Peer, input []byte) (*PeerResponse, error) {
	s := string(input)
	if strings.HasPrefix(s, "+ID") {
		idArr := strings.Split(s, " ")
		from.ID = idArr[1]
	}
	return &PeerResponse{}, nil
}

type PeerRequest struct {
	Payload []byte
}

func (r *PeerRequest) Execute() error {
	return nil
}

func (p *Peer) ParsePeerRequest(input []byte) (*PeerRequest, error) {
	// switch string(input) {
	// case "PEER ID":
	// 	return &PeerRequest{
	// 		Payload: []byte("+ID " + p.ID),
	// 	}, nil
	// default:
	// 	fmt.Println("Unknown peer request:", string(input))
	// }
	q, err := p.vqlClient.ParseRawQuery(input)
	if err != nil {
		fmt.Printf("[peer] Cannot parse vql query: %+v", err)
	}
	_, err = q.Execute()
	if err != nil {
		fmt.Printf("Cannot execute query: %+v", err)
	}
	return &PeerRequest{}, nil
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
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("[peer] Error reading:", err.Error())
			break
		}
		peerRequest, err := p.ParsePeerRequest(buf[:reqLen])
		if err != nil {
			conn.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
			break
		}
		conn.Write([]byte(peerRequest.Payload))

		// if resp.DisconnectSignal {
		// 	break
		// }
	}
	conn.Close()
	fmt.Printf("[peer] Connection closed %s\n", conn.RemoteAddr().String())
}

func (p *Peer) Shutdown() {
}

func (p *Peer) PublishVQL(query []byte) {
	for p := range p.Mesh.Peers {
		if p.ConnectionStatus() == PEER_STATUS_CONNECTED {
			p.RemoteConn.Write(query)
		}
	}
}
