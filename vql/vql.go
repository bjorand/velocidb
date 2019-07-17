package vql

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/bjorand/velocidb/peering"
	storagePkg "github.com/bjorand/velocidb/storage"
	tcp "github.com/bjorand/velocidb/tcp"
	"github.com/bjorand/velocidb/utils"
	"github.com/google/uuid"
)

var (
	verbs          = []string{"quit", "peer"}
	firstByteArray = []byte("*")
	endByte        = []byte("\r\n")
	storage        *storagePkg.MemoryStorage
	walWriter      *storagePkg.WalFileWriter
)

const (
	typeArray        = "*"
	typeInteger      = ":"
	typeSimpleString = "+"
	typeBulkString   = "$"
	typeError        = "-"
)

type Query struct {
	raw  []byte
	id   string
	text string
	v    *VQLTCPServer
}

type Response struct {
	Payload          [][]byte
	DisconnectSignal bool
	Type             string
}

func NewResponse() *Response {
	return &Response{
		// Payload: make([][]byte),
	}
}

func (r *Response) PayloadString(s []byte) {
	r.Payload = make([][]byte, 1)
	r.Payload[0] = s
}

func (r *Response) OK() {
	r.PayloadString([]byte("OK"))
}

func SanitizeTextInput(data []byte) string {
	d := string(data)
	d = strings.Trim(d, " \r\n")
	return d
}

func Sanitize(data []byte) []byte {
	// d := string(data)
	// return strings.Trim(d, " \r\n")
	return data
}

func ParseRawResponse(data []byte) (*Response, error) {
	r := NewResponse()
	if len(r.Payload) > 0 {
		r.Payload[0] = Sanitize(data)
	}
	// if r.Payload == "+ATH0" {
	// 	r.DisconnectSignal = true
	// }
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
			text = SanitizeTextInput(data)
		}
		break

	}
	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}
	return &Query{
		raw:  data,
		id:   id.String(),
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
	storage.Set(key, value)
}

func (q *Query) Incr(key string) ([]byte, error) {
	v, err := storage.Incr(key)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (q *Query) Decr(key string) ([]byte, error) {
	v, err := storage.Decr(key)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (q *Query) Get(key string) []byte {
	return storage.Get(key)
}

func (q *Query) Del(keys ...string) []byte {
	var deletedCount int
	for _, key := range keys {
		deleted := storage.Del(key)
		if deleted {
			deletedCount = deletedCount + 1
		}
	}
	return []byte(fmt.Sprintf("%d", deletedCount))

}

func (q *Query) WalWrite() {
	walWriter.SyncWrite(q.raw)
}

func (q *Query) Execute() (*Response, error) {
	r := NewResponse()
	args := q.args()
	syntax := map[string]map[string]func() error{
		"peer": {
			"list": func() error {
				for _, peer := range q.v.Peer.Peers {
					r.Payload = append(r.Payload, []byte(fmt.Sprintf("*%s\t%s:%d\tConnection:%s\tBytesIn:%s\n",
						peer.ID,
						peer.ListenAddr,
						peer.ListenPort,
						peer.ConnectionStatus(),
						utils.HumanSizeBytes(peer.Stats.BytesIn),
					)))
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
				r.Payload[0] = []byte(fmt.Sprintf("Connecting to peer %s:%d\n", host, port))
				return nil
			},
			"remove": func() error {
				if len(args) < 2 {
					return fmt.Errorf(Help("peer"))
				}
				peer := q.v.Peer.Peers[args[1]]
				if peer != nil {
					q.v.Peer.RemovePeer(peer)
					r.OK()
					return nil
				}
				return fmt.Errorf("Peer %s not found in peer list", args[1])
			},
		},
		"info": {
			"": func() error {
				for k, v := range q.v.Peer.Info() {
					r.Payload = append(r.Payload, []byte(fmt.Sprintf("%s\t%+v\n", k, v)))
				}
				return nil
			},
		},
		"ping": {
			"": func() error {
				r.PayloadString([]byte("PONG"))
				return nil
			},
		},
		"flushdb": {
			"": func() error {
				storage.FlushData()
				r.OK()
				return nil
			},
		},
		"set": {
			"*": func() error {
				if len(args) < 2 {
					return fmt.Errorf("Too few arguments")
				}
				q.Set(args[0], []byte(strings.Join(args[1:], " ")))
				q.WalWrite()
				r.OK()
				return nil
			},
		},
		"get": {
			"*": func() error {
				if len(args) < 1 {
					return fmt.Errorf("Too many arguments")
				}
				r.PayloadString([]byte(q.Get(args[0])))
				r.Type = typeBulkString
				return nil
			},
		},
		"del": {
			"*": func() error {
				if len(args) < 1 {
					return fmt.Errorf("Too many arguments")
				}
				r.PayloadString([]byte(q.Del(args...)))
				q.WalWrite()
				r.Type = typeInteger
				return nil
			},
		},
		"incr": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				v, err := q.Incr(args[0])
				if err != nil {
					return err
				}
				r.PayloadString([]byte(v))
				q.WalWrite()
				r.Type = typeInteger
				return nil
			},
		},
		"decr": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				v, err := q.Decr(args[0])
				if err != nil {
					return err
				}
				r.PayloadString([]byte(v))
				q.WalWrite()
				r.Type = typeInteger
				return nil
			},
		},
		"keys": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				for _, k := range storage.Keys() {
					r.Payload = append(r.Payload, []byte(k))
				}
				r.Type = typeArray
				return nil
			},
		},
		"quit": {
			"": func() error {
				r.DisconnectSignal = true
				r.OK()
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
				return r, err
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
	v := &VQLTCPServer{
		Peer:       peer,
		ListenAddr: listenAddr,
		ListenPort: listenPort,
	}
	storage = v.StorageInit()
	walWriter = storagePkg.NewWalFileWriter("/tmp")
	go walWriter.Run()
	return v, nil
}

func (v *VQLTCPServer) StorageInit() *storagePkg.MemoryStorage {
	return storagePkg.NewMemoryStorage()
}

func (v *VQLTCPServer) Run() {
	defer walWriter.Close()
	s, err := tcp.NewTCPServer(v.ListenAddr, v.ListenPort)
	if err != nil {
		panic(err)
	}
	s.Run("vql", v.HandleVQLRequest)
}

func (r *Response) Size() int {
	return len(r.Payload)
}

func (r *Response) isBulkString() bool {
	if r.Type == typeBulkString {
		return true
	}
	return false
}

func (r *Response) isArray() bool {
	if r.Type == typeArray {
		return true
	}
	return false
}

func (r *Response) isInteger() bool {
	if r.Type == typeInteger {
		return true
	}
	return false
}

func (r *Response) isNullBulkString() bool {
	if r.Type == typeBulkString && len(r.Payload[0]) == 0 {
		return true
	}
	return false
}

func (r *Response) FormattedPayload() []byte {
	var payload []byte

	if len(r.Payload) == 1 {
		if !r.isArray() {
			if r.isBulkString() {
				if r.isNullBulkString() {
					payload = []byte("$-1")
				} else {
					payload = []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[0])))
					payload = append(payload, r.Payload[0]...)
				}
			} else if r.isInteger() {
				payload = []byte(fmt.Sprintf(":%s", r.Payload[0]))
			} else {
				payload = []byte(fmt.Sprintf("+%s", r.Payload[0]))
			}
			payload = append(payload, "\r\n"...)
		}
	} else {
		payload = []byte(fmt.Sprintf("*%d\r\n", len(r.Payload)))
		for i := 0; i < len(r.Payload); i++ {
			payload = append(payload, []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[i])))...)
			payload = append(payload, r.Payload[i]...)
			payload = append(payload, []byte("\r\n")...)
		}

	}
	return payload
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
			conn.Write([]byte(fmt.Sprintf("-%s\r\n", err.Error())))
			continue
		}
		resp, err := query.Execute()
		if err != nil {
			conn.Write([]byte(fmt.Sprintf("-%s\r\n", err.Error())))
			continue
		}
		conn.Write(resp.FormattedPayload())
		if resp.DisconnectSignal {
			break
		}
	}
	conn.Close()
	fmt.Printf("[vql] Connection closed %s\n", conn.RemoteAddr().String())
}

func (v *VQLTCPServer) Shutdown() {
	walWriter.Close()
	<-walWriter.WaitTerminate
	fmt.Println("[vql] shutdown")
}
