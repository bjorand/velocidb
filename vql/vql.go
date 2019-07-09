package vql

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/bjorand/velocidb/peering"
	tcp "github.com/bjorand/velocidb/tcp"
	"github.com/bjorand/velocidb/utils"
)

var (
	verbs          = []string{"quit", "peer"}
	firstByteArray = []byte("*")
	endByte        = []byte("\r\n")
)

const ()

type Query struct {
	text string
	v    *VQLTCPServer
}

type Response struct {
	Payload          string
	DisconnectSignal bool
	SimpleString     bool
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

func readInt(data []byte, cursor int) (int, int) {
	for i, d := range data {
		if d == endByte[0] && data[i+1] == endByte[1] {
			eC := string(data[0:i])
			elementsCount, err := strconv.Atoi(eC)
			if err != nil {
				break
			}
			cursor += i + len(endByte)
			return elementsCount, cursor
		}
	}

	return -1, cursor
}

func (v *VQLTCPServer) ParseRawQuery(data []byte) (*Query, error) {
	// text := Sanitize(data)
	var text string
	var readCur int
	var bytesToRead int
	var elementsCount int
	var initialRow int

	for i, d := range data {

		if d == '*' {
			readCur = i + 1
			elementsCount, readCur = readInt(data[readCur:], readCur)
			for row := initialRow; row < elementsCount; row++ {
				if data[readCur] == '$' {
					readCur++
					bytesToRead, readCur = readInt(data[readCur:], readCur)
					if len(text) > 0 {
						text += " "
					}
					text += string(data[readCur : readCur+bytesToRead])
					readCur += bytesToRead + len(endByte)
				}
			}
		} else {
			text = Sanitize(data)
		}
		break

	}

	return &Query{
		text: text,
		v:    v,
	}, nil
}

func (q *Query) words() []string {
	return strings.Split(q.text, " ")
}

func (q *Query) verb() string {
	return strings.ToLower(q.words()[0])
}

func (q *Query) args() []string {
	return q.words()[1:]
}

func (q *Query) Set(key string, value []byte) {
	fmt.Println(key, value)
}

func (q *Query) Get(key string) []byte {
	return []byte("1")
}

func (q *Query) Execute() (*Response, error) {
	r := &Response{}
	args := q.args()
	syntax := map[string]map[string]func() error{
		"peer": {
			"list": func() error {
				for _, peer := range q.v.Peer.Peers {
					r.Payload += fmt.Sprintf("*%s\t%s:%d\tConnection:%s\tBytesIn:%s\n",
						peer.ID,
						peer.ListenAddr,
						peer.ListenPort,
						peer.ConnectionStatus(),
						utils.HumanSizeBytes(peer.Stats.BytesIn),
					)
				}
				return nil
			},
			"connect": func() error {
				if len(args) < 2 {
					return fmt.Errorf(Help("peer"))
				}
				host, port, err := utils.SplitHostPort(args[1])
				if err != nil {
					return err
				}
				go func() {
					q.v.Peer.ConnectToPeerAddr(args[1])
				}()
				r.Payload = fmt.Sprintf("Connecting to peer %s:%d\n", host, port)
				return nil
			},
			"remove": func() error {
				if len(args) < 2 {
					return fmt.Errorf(Help("peer"))
				}
				peer := q.v.Peer.Peers[args[1]]
				if peer != nil {
					q.v.Peer.RemovePeer(peer)
					r.Payload = "+OK"
					return nil
				}
				return fmt.Errorf("Peer %s not found in peer list", args[1])
			},
		},
		"info": {
			"": func() error {
				for k, v := range q.v.Peer.Info() {
					r.Payload += fmt.Sprintf("%s\t%+v\n", k, v)
				}
				return nil
			},
		},
		"ping": {
			"": func() error {
				r.Payload += "+PONG\r\n"
				r.SimpleString = true
				return nil
			},
		},
		"set": {
			"*": func() error {
				if len(args) < 2 {
					return fmt.Errorf("Too few arguments")
				}
				q.Set(args[0], []byte(strings.Join(args[1:], " ")))
				return nil
			},
		},
		"get": {
			"*": func() error {
				if len(args) < 1 {
					return fmt.Errorf("Too many arguments")
				}
				r.Payload = string(q.Get(args[0]))
				return nil
			},
		},
		"quit": {
			"": func() error {
				r.DisconnectSignal = true
				r.Payload = "+ATH0"
				return nil
			},
		},
		"help": {
			"": func() error {
				return fmt.Errorf(Help(""))
			},
		},
	}
	verb := syntax[q.verb()]

	if len(args) > 0 {
		if verb["*"] != nil {
			err := verb["*"]()
			if err != nil {
				return nil, err
			}
			return r, nil
		}
		f := verb[args[0]]
		if f != nil {
			err := f()
			return r, err
		}
		return nil, fmt.Errorf("ERR unknown command '%s %s'", q.verb(), args[0])
	}
	if verb[""] == nil {
		return nil, fmt.Errorf("ERR unknown command '%s'", q.verb())
	}
	if len(q.args()) == 0 {
		err := syntax[q.verb()][""]()
		return r, err
	}
	return nil, nil
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

func (r *Response) Size() int {
	return len(r.Payload)
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
		if !resp.SimpleString {
			conn.Write([]byte(fmt.Sprintf("$%d\r\n", resp.Size())))
		}
		conn.Write([]byte(resp.Payload))
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
