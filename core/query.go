package core

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bjorand/velocidb/utils"
)

type Query struct {
	raw         []byte
	id          string
	parsed      [][]byte
	c           *VQLClient
	hasMoreData int
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
	q.c.vqlTCPServer.Peer.storage.Set(key, value)
	q.WalWrite()

}

func (q *Query) Incr(key string) ([]byte, error) {
	v, err := q.c.vqlTCPServer.Peer.storage.Incr(key)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (q *Query) Decr(key string) ([]byte, error) {
	v, err := q.c.vqlTCPServer.Peer.storage.Decr(key)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (q *Query) Get(key string) []byte {
	return q.c.vqlTCPServer.Peer.storage.Get(key)
}

func (q *Query) Del(keys ...string) []byte {
	var deletedCount int
	for _, key := range keys {
		deleted := q.c.vqlTCPServer.Peer.storage.Del(key)
		if deleted {
			deletedCount = deletedCount + 1
		}
	}
	return []byte(fmt.Sprintf("%d", deletedCount))
}

func (q *Query) WalWrite() {
	q.c.vqlTCPServer.Peer.walWriter.SyncWrite(q.raw)
	q.c.vqlTCPServer.Peer.PublishVQL(q.raw)
}

func (q *Query) Execute() (*Response, error) {
	r := NewResponse()
	args := q.args()
	syntax := map[string]map[string]func() error{
		"peer": {
			"list": func() error {
				for peer := range q.c.vqlTCPServer.Peer.Mesh.Peers {
					r.Payload = append(r.Payload, []byte(fmt.Sprintf("*%s\t%s:%d\tConnection:%s\tBytesIn:%s\n",
						peer.ID,
						peer.ListenAddr,
						peer.ListenPort,
						PEER_STATUS_TEXT[peer.ConnectionStatus()],
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
				peer := q.c.vqlTCPServer.Peer.Mesh.GetPeerByKey(args[1])
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
				r.PayloadString([]byte(fmt.Sprintf("%s\r\n", strings.Join(infoStorage(q.c.vqlTCPServer), "\r\n"))))
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
				info = append(info, infoStorage(q.c.vqlTCPServer)...)
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
				q.c.vqlTCPServer.Peer.storage.FlushData()
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
				for _, k := range q.c.vqlTCPServer.Peer.storage.Keys(args[0]) {
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
				err, cursor, keys = q.c.vqlTCPServer.Peer.storage.Scan(cursor, count, match, typeFilter)
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
