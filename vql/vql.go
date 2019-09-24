package vql

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bjorand/velocidb/peering"
	storagePkg "github.com/bjorand/velocidb/storage"
	tcp "github.com/bjorand/velocidb/tcp"
	"github.com/bjorand/velocidb/utils"
	"github.com/google/uuid"
)

var (
	verbs            = []string{"quit", "peer"}
	firstByteArray   = []byte("*")
	endByte          = []byte("\r\n")
	storage          *storagePkg.MemoryStorage
	walWriter        *storagePkg.WalFileWriter
	lock             = sync.RWMutex{}
	lastConnectionID int64
)

const (
	typeArray        = "*"
	typeInteger      = ":"
	typeSimpleString = "+"
	typeBulkString   = "$"
	typeError        = "-"
)

type Query struct {
	raw         []byte
	id          string
	parsed      [][]byte
	c           *VQLClient
	hasMoreData int
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

func (c *VQLClient) ParseRawQuery(data []byte) (*Query, error) {
	// text := Sanitize(data)
	var readCur int
	var bytesToRead int
	var elementsCount int
	var initialRow int

	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}
	q := &Query{
		raw: data,
		id:  id.String(),
		c:   c,
	}

	for i, d := range data {
		if d == '*' {
			readCur = i + 1
			elementsCount, readCur = readInt(data[readCur:], readCur)
			for row := initialRow; row < elementsCount; row++ {
				if data[readCur] == '$' {
					readCur++
					bytesToRead, readCur = readInt(data[readCur:], readCur)
					if len(data) >= readCur+bytesToRead {
						q.parsed = append(q.parsed, data[readCur:readCur+bytesToRead])
					} else {
						q.parsed = append(q.parsed, data[readCur:])
						q.hasMoreData = bytesToRead - len(data) - readCur
						break
					}
					readCur += bytesToRead + len(endByte)
				}
			}
		} else {
			for _, w := range bytes.Split(data, []byte(" ")) {
				w = bytes.Trim(w, string(endByte))
				q.parsed = append(q.parsed, []byte(w))
			}
			break
		}
		break

	}

	return q, nil
}

func (q *Query) words() []string {
	words := []string{}
	for _, w := range q.parsed {
		words = append(words, string(w))
	}
	return words
}

func (q *Query) verb() string {
	if len(q.words()) > 0 {
		return strings.ToLower(string(q.parsed[0]))
	}
	return ""
}

func (q *Query) args() []string {
	if len(q.words()) > 0 {
		return q.words()[1:]
	}
	return []string{}
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

func infoStorage() (info []string) {
	info = append(info, "# Keyspace")
	info = append(info, fmt.Sprintf("db0:keys=%d", len(storage.Keys("*"))))
	return info
}

func infoWal(v *VQLTCPServer) (info []string) {
	walFilesize, _ := v.walWriter.WalFile.Size()
	info = append(info, "# Wal")
	info = append(info, fmt.Sprintf("current_wal_file:%s", v.walWriter.WalFile.Path()))
	info = append(info, fmt.Sprintf("current_wal_file_size_bytes:%d", walFilesize))
	info = append(info, fmt.Sprintf("write_bytes:%d", v.walWriter.BytesWritten))
	info = append(info, fmt.Sprintf("write_ops:%d", v.walWriter.WriteOps))
	return info
}

func infoVQL(v *VQLTCPServer) (info []string) {
	info = append(info, "# VQL")
	info = append(info, fmt.Sprintf("connected_clients:%d", len(v.clients)))
	return info
}

func infoServer(peer *peering.Peer) (info []string) {
	info = append(info, "# Server")
	peerInfo := peer.Info()
	names := make([]string, 0, len(peerInfo))
	for name := range peerInfo {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info = append(info, fmt.Sprintf("%s:%+v", name, peerInfo[name]))
	}
	return info
}

func (q *Query) Execute() (*Response, error) {
	r := NewResponse()
	args := q.args()
	syntax := map[string]map[string]func() error{
		"peer": {
			"list": func() error {
				for _, peer := range q.c.vqlTCPServer.Peer.Peers {
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
				go func() {
					q.c.vqlTCPServer.Peer.ConnectToPeerAddr(args[1])
				}()
				r.OK()
				return nil
			},
			"remove": func() error {
				if len(args) < 2 {
					return fmt.Errorf(Help("peer"))
				}
				peer := q.c.vqlTCPServer.Peer.Peers[args[1]]
				if peer != nil {
					q.c.vqlTCPServer.Peer.RemovePeer(peer)
					r.OK()
					return nil
				}
				return fmt.Errorf("Peer %s not found in peer list", args[1])
			},
		},
		"client": {
			"list": func() error {
				r.Type = typeBulkString
				var clients []string
				for c := range q.c.vqlTCPServer.clients {
					clients = append(clients, fmt.Sprintf("id=%d addr=%s name=%s", c.id, c.conn.RemoteAddr().String(), c.name))
				}
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(clients, "\r\n"))))
				return nil
			},
			"setname": func() error {
				if len(args) > 2 {
					return fmt.Errorf("Too many arguments")
				}
				q.c.name = args[1]
				r.OK()
				return nil
			},
			"getname": func() error {
				if len(args) > 1 {
					return fmt.Errorf("Too many arguments")
				}
				r.PayloadString([]byte(q.c.name))
				r.Type = typeBulkString
				return nil
			},
			"kill": func() error {
				if len(args) > 2 {
					return fmt.Errorf("Too many arguments")
				}
				if len(args) == 1 || args[1] == "" {
					return fmt.Errorf("syntax error")
				}
				// find client with "host:port"
				for c := range q.c.vqlTCPServer.clients {
					if c.conn.RemoteAddr().String() == args[1] {
						r.DisconnectSignal = true
						r.OK()
						return nil
					}
				}
				return fmt.Errorf("No such client")
			},
		},
		"info": {
			"server": func() error {
				r.Type = typeBulkString
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(infoServer(q.c.vqlTCPServer.Peer), "\r\n"))))
				return nil
			},
			"keyspace": func() error {
				r.Type = typeBulkString
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(infoStorage(), "\r\n"))))
				return nil
			},
			"vql": func() error {
				r.Type = typeBulkString
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(infoVQL(q.c.vqlTCPServer), "\r\n"))))
				return nil
			},
			"wal": func() error {
				r.Type = typeBulkString
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(infoWal(q.c.vqlTCPServer), "\r\n"))))
				return nil
			},
			"": func() error {
				var info []string
				info = append(info, infoServer(q.c.vqlTCPServer.Peer)...)
				info = append(info, infoStorage()...)
				info = append(info, infoVQL(q.c.vqlTCPServer)...)
				info = append(info, infoWal(q.c.vqlTCPServer)...)
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(info, "\r\n"))))
				r.Type = typeBulkString
				return nil
			},
		},
		"ping": {
			"": func() error {
				r.PayloadString([]byte("PONG"))
				return nil
			},
			"*": func() error {
				if len(args) > 1 {
					return fmt.Errorf("Too many arguments")
				}
				r.Type = typeBulkString
				r.PayloadString([]byte(args[0]))
				return nil
			},
		},
		"flushdb": {
			"": func() error {
				storage.FlushData()
				q.WalWrite()
				r.OK()
				return nil
			},
		},
		"time": {
			"": func() error {
				t := time.Now()
				r.Payload = append(
					r.Payload,
					[]byte(fmt.Sprintf("%d", t.Unix())),
					[]byte(fmt.Sprintf("%d", t.UnixNano()%int64(time.Second)/int64(time.Microsecond))))
				r.Type = typeArray
				return nil
			},
		},
		"set": {
			"*": func() error {
				if len(args) < 2 {
					return fmt.Errorf("Too few arguments")
				}
				q.Set(args[0], q.parsed[2])
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
				for _, k := range storage.Keys(args[0]) {
					r.Payload = append(r.Payload, []byte(k))
				}
				r.Type = typeArray
				return nil
			},
		},
		"scan": {
			"*": func() error {
				if len(args) > 7 {
					return fmt.Errorf("Too many arguments")
				}
				if len(args)%2 != 1 {
					return fmt.Errorf("Too few arguments")
				}
				cursor := 0
				match := "*"
				count := 10
				typeFilter := "string"
				for i, arg := range args {
					if strings.ToLower(arg) == "match" && args[i+1] != "" {
						match = args[i+1]
					}
					if strings.ToLower(arg) == "type" && args[i+1] != "" {
						typeFilter = args[i+1]
					}
					if strings.ToLower(arg) == "count" && args[i+1] != "" {
						c, err := strconv.Atoi(args[i+1])
						if err != nil {
							return fmt.Errorf("Value is not an integer or out of range")
						}
						count = c
					}
				}
				var keys []string
				var err error
				err, cursor, keys = storage.Scan(cursor, count, match, typeFilter)
				if err != nil {
					return err
				}

				r.Payload = append(r.Payload, []byte(fmt.Sprintf("%d", cursor)))
				var keysB [][]byte
				for _, key := range keys {
					keysB = append(keysB, []byte(key))
				}

				f := formattedArray(keysB)
				r.Payload = append(r.Payload, f)
				r.Type = typeArray
				return nil
			},
		},
		"type": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				// TODO we need to support more types
				r.PayloadString([]byte("string"))
				r.Type = typeSimpleString
				return nil
			},
		},
		"ttl": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				// TODO we need to support TTL
				r.PayloadString([]byte("-1"))
				r.Type = typeInteger
				return nil
			},
		},
		"select": {
			"*": func() error {
				if len(args) != 1 {
					return fmt.Errorf("Too many arguments")
				}
				// TODO we have to implement multi db support
				if args[0] != "0" {
					return fmt.Errorf("invalid DB index")
				}
				r.OK()
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

type VQLClient struct {
	id           int64
	name         string
	conn         net.Conn
	vqlTCPServer *VQLTCPServer
}

type VQLTCPServer struct {
	Peer       *peering.Peer
	ListenAddr string
	ListenPort int64
	walWriter  *storagePkg.WalFileWriter
	clients    map[*VQLClient]bool
}

func NewVQLTCPServer(peer *peering.Peer, listenAddr string, listenPort int64) (*VQLTCPServer, error) {

	v := &VQLTCPServer{
		Peer:       peer,
		ListenAddr: listenAddr,
		ListenPort: listenPort,
		clients:    make(map[*VQLClient]bool),
	}
	storage = v.StorageInit()
	walWriter = storagePkg.NewWalFileWriter("/tmp")
	go walWriter.Run()
	v.walWriter = walWriter
	return v, nil
}

func (v *VQLTCPServer) StorageInit() *storagePkg.MemoryStorage {
	return storagePkg.NewMemoryStorage()
}

func (v *VQLTCPServer) clientNextID() int64 {
	lock.Lock()
	defer lock.Unlock()
	lastConnectionID++
	return lastConnectionID
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

func formattedArray(items [][]byte) []byte {
	payload := []byte(fmt.Sprintf("*%d\r\n", len(items)))
	for i := 0; i < len(items); i++ {
		if !bytes.HasPrefix(items[i], []byte("*")) {
			payload = append(payload, []byte(fmt.Sprintf("$%d\r\n", len(items[i])))...)
		}
		payload = append(payload, items[i]...)
		payload = append(payload, []byte("\r\n")...)
	}
	return payload
}

func (r *Response) FormattedPayload() []byte {
	var payload []byte

	if len(r.Payload) == 1 && !r.isArray() {

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

	} else {
		payload = []byte(fmt.Sprintf("*%d\r\n", len(r.Payload)))
		for i := 0; i < len(r.Payload); i++ {

			if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
				payload = append(payload, []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[i])))...)
			}
			payload = append(payload, r.Payload[i]...)
			if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
				payload = append(payload, []byte("\r\n")...)
			}
		}

	}
	return payload
}

func (v *VQLTCPServer) HandleVQLRequest(s *tcp.TCPServer, conn net.Conn) {
	// Make a buffer to hold incoming data.
	var hasMoreData int
	var query *Query
	client := &VQLClient{
		id:           v.clientNextID(),
		conn:         conn,
		vqlTCPServer: v,
	}
	fmt.Printf("[vql] Serving addr=%s\n", conn.RemoteAddr().String())
	lock.Lock()
	v.clients[client] = true
	lock.Unlock()
	defer func() {
		lock.Lock()
		conn.Close()
		fmt.Printf("[vql] Connection closed addr=%s\n", conn.RemoteAddr().String())
		delete(v.clients, client)
		lock.Unlock()
	}()
	for {
		buf := make([]byte, 1024)
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("[vql] -Error reading:", err.Error())
			break
		}
		if hasMoreData > 0 {
			query.parsed[len(query.parsed)-1] = append(query.parsed[len(query.parsed)-1], buf[:reqLen]...)
			hasMoreData = hasMoreData - reqLen
			if hasMoreData > 0 {
				continue
			} else {
				hasMoreData = -1
			}
		}
		// extended read finished
		if hasMoreData == 0 {
			query, err = client.ParseRawQuery(buf[:reqLen])
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("-%s\r\n", err.Error())))
				continue
			}
			hasMoreData = query.hasMoreData
			if hasMoreData > 0 {
				continue
			}
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
}

func (v *VQLTCPServer) Shutdown() {
	walWriter.Close()
	<-walWriter.WaitTerminate
	fmt.Println("[vql] shutdown")
}
