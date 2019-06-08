package vql

import (
	"fmt"
	"net"
	"strings"

	"github.com/bjorand/velocidb/peering"
	tcp "github.com/bjorand/velocidb/tcp"
	utils "github.com/bjorand/velocidb/utils"
)

var (
	verbs = []string{"quit", "peer"}
)

type Query struct {
	text string
	v    *VQLTCPServer
}

type Response struct {
	Payload          string
	DisconnectSignal bool
}

func Sanitize(data []byte) string {
	d := string(data)
	return strings.Trim(d, " \r\n")
}

func ParseRawResponse(data []byte) (*Response, error) {
	r := &Response{}
	r.Payload = Sanitize(data)
	if r.Payload == "+ATH0" {
		r.DisconnectSignal = true
	}
	return r, nil
}

func (v *VQLTCPServer) ParseRawQuery(data []byte) (*Query, error) {
	text := Sanitize(data)

	return &Query{
		text: text,
		v:    v,
	}, nil
}

func (q *Query) words() []string {
	return strings.Split(q.text, " ")
}

func (q *Query) verb() string {
	return q.words()[0]
}

func (q *Query) Execute() (*Response, error) {
	r := &Response{}
	switch q.verb() {
	case "quit":
		r.DisconnectSignal = true
		r.Payload = "+ATH0"
		return r, nil
	case "help":
		return nil, fmt.Errorf(Help(""))
	case "peer":
		if len(q.words()) == 1 {
			return nil, fmt.Errorf(Help("peer"))
		}
		arg := q.words()[1]
		switch arg {
		case "connect":
			if len(q.words()) < 3 {
				return nil, fmt.Errorf(Help("peer"))
			}
			host, port, err := utils.SplitHostPort(q.words()[2])
			if err != nil {
				return nil, err
			}
			go q.v.Peer.ConnectToPeerAddr(q.words()[2])
			r.Payload = fmt.Sprintf("Connecting to peer %s:%d\n", host, port)
			return r, nil
		case "list":
			for _, peer := range q.v.Peer.Peering.Peers {
				r.Payload += fmt.Sprintf("%s:%d\n", peer.ListenAddr, peer.ListenPort)
			}
			return r, nil
		default:
			return nil, fmt.Errorf(Help("peer"))
		}
	default:
		return nil, fmt.Errorf("-ERR unknown command '%s'", q.verb())
	}
}

type VQLTCPServer struct {
	Peer       *peering.Peer
	ListenAddr string
	ListenPort int64
}

func NewVQLTCPServer(peer *peering.Peer, listenAddr string, listenPort int64) (*VQLTCPServer, error) {
	return &VQLTCPServer{
		Peer:       peer,
		ListenAddr: listenAddr,
		ListenPort: listenPort,
	}, nil
}

func (v *VQLTCPServer) Run() {
	s, err := tcp.NewTCPServer(v.ListenAddr, v.ListenPort)
	if err != nil {
		panic(err)
	}
	s.Run("vql", v.HandleVQLRequest)
}

func (v *VQLTCPServer) HandleVQLRequest(s *tcp.TCPServer, conn net.Conn) {
	fmt.Printf("[vql] Serving %s\n", conn.RemoteAddr().String())
	// Make a buffer to hold incoming data.
	for {
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("[vql] -Error reading:", err.Error())
			break
		}
		query, err := v.ParseRawQuery(buf[:reqLen])
		if err != nil {
			conn.Write([]byte(fmt.Sprintf("-%s\n", err.Error())))
			continue
		}
		resp, err := query.Execute()
		if err != nil {
			conn.Write([]byte(fmt.Sprintf("-%s\n", err.Error())))
			continue
		}
		conn.Write([]byte(fmt.Sprintf("%s\n", resp.Payload)))
		if resp.DisconnectSignal {
			break
		}
	}
	conn.Close()
	fmt.Printf("[vql] Connection closed %s\n", conn.RemoteAddr().String())
}

func (v *VQLTCPServer) Shutdown() {
	fmt.Println("[vql] shutdown")
}
