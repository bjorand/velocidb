package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"time"

	tcp "github.com/bjorand/velocidb/tcp"

	logger "github.com/bjorand/velocidb/logger"
	storagePkg "github.com/bjorand/velocidb/storage"
	utils "github.com/bjorand/velocidb/utils"

	"github.com/google/uuid"
)

const (
	// config
	TCP_WORKERS_PER_PEER = 4
	QUERY_TIMEOUT        = 3

	PEER_STATUS_NO_CONNECTION = 0
	PEER_STATUS_CONNECTED     = 1
)

var (
	PEER_STATUS_TEXT = map[int]string{
		PEER_STATUS_NO_CONNECTION: "No connection",
		PEER_STATUS_CONNECTED:     "Connected",
	}
	PEER_QUERY_TYPE    = []byte("0")
	PEER_RESPONSE_TYPE = []byte("1")
)

type Stats struct {
	BytesIn                      int64
	BytesOut                     int64
	connectionReadFailureCounter int64
	connectionLastError          error
}

type Peer struct {
	ID                  string
	Tags                []string
	RemoteConn          net.Conn
	Protocol            string
	Height              int64
	ListenPort          int64
	ListenAddr          string
	Stats               *Stats
	RemoveSignal        bool
	Name                string
	Mesh                *Mesh
	gotRawQueryFromPeer chan []byte
	// queryResponseSendQueue chan []byte
	queryResponseReceived chan []byte
	responseQueueToSend   chan *Response
	broadcastVQLQuery     chan *Query
	queryWaiting          map[string]chan *Response
	storage               *storagePkg.MemoryStorage
	walWriter             *storagePkg.WalFileWriter
	tcpServer             *tcp.TCPServer
	vqlTCPServer          *VQLTCPServer
	updateTrigger         chan bool
	l                     *logger.Logger
}

func (p *Peer) ParseRawQuery(c *VQLClient, data []byte) (*Query, error) {
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
		p:   p,
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

func NewPeer(listenAddr string, port int64) (*Peer, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	walDir, err := ioutil.TempDir("/tmp", fmt.Sprintf("wal-%s", id))
	if err != nil {
		return nil, err
	}
	peerID := id.String()
	return &Peer{
		ID:                peerID,
		ListenAddr:        listenAddr,
		ListenPort:        port,
		Stats:             &Stats{},
		Mesh:              newMesh(),
		broadcastVQLQuery: make(chan *Query, 1024),
		queryWaiting:      make(map[string]chan *Response),
		storage:           storagePkg.NewMemoryStorage(),
		walWriter:         storagePkg.NewWalFileWriter(walDir),
		l:                 logger.NewLogger(logger.Fields{"peer": peerID, "self": true}),
	}, nil
}

func NewRemotePeer(listenAddr string, port int64) (*Peer, error) {
	return &Peer{
		ListenAddr:            listenAddr,
		ListenPort:            port,
		Stats:                 &Stats{},
		Mesh:                  newMesh(),
		broadcastVQLQuery:     make(chan *Query, 1024),
		gotRawQueryFromPeer:   make(chan []byte, 1024),
		queryWaiting:          make(map[string]chan *Response),
		responseQueueToSend:   make(chan *Response, 1024),
		queryResponseReceived: make(chan []byte, 1024),
		updateTrigger:         make(chan bool, 1024),
	}, nil
}

func (p *Peer) connString() string {
	if p.tcpServer != nil {
		return fmt.Sprintf("%s:%d", p.tcpServer.Host, p.tcpServer.Port)
	}
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

func (p *Peer) ConnectionStatus() int {
	if p.RemoteConn != nil {
		return PEER_STATUS_CONNECTED
	}
	return PEER_STATUS_NO_CONNECTION
}

func (p *Peer) ConnectToPeerAddr(peerConnString string) (*Peer, error) {
	peerAddr, peerPort, err := utils.SplitHostPort(peerConnString)
	if err != nil {
		return nil, err
	}
	newPeer, err := NewRemotePeer(peerAddr, peerPort)
	if err != nil {
		return nil, err
	}
	go p.connectToPeer(newPeer)
	return newPeer, nil
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
	p.Mesh.register <- newPeer
	fmt.Printf("[mesh] register peer %s\n", newPeer.ID)
	// initialPause := 0
	// maxPause := 60
	// pause := initialPause
	// for {
	// 	if newPeer.RemoveSignal {
	// 		break
	// 	}
	// 	p.connectToPeer(remotePeer)
	// }

	fmt.Printf("[peer] Connecting to peer %s\n", newPeer.connString())
	newPeer.RemoteConn = nil
	// time.Sleep(time.Duration(pause) * time.Second)
	conn, err := net.Dial("tcp4", newPeer.connString())
	if err != nil {
		fmt.Printf("[peer %s] %s\n", newPeer.connString(), err)
		// if pause < maxPause {
		// 	pause += 2
		// }
		return
	}
	newPeer.RemoteConn = conn
	fmt.Printf("[peer %s] Connected\n", newPeer.connString())
	// pause = initialPause

	c := NewVQLClient(-1, fmt.Sprintf("peer-%s", p.ID), conn, p.vqlTCPServer)

	for i := 0; i < TCP_WORKERS_PER_PEER; i++ {
		go p.ResponseReader(i, newPeer, c)
		go p.ResponseWriter(newPeer, c)
		go p.QueryReader(newPeer, c)
		go p.QueryWriter(newPeer, c)
	}
	go p.Updater(newPeer)

	defer func() {
		defer conn.Close()
		close(newPeer.queryResponseReceived)
		close(newPeer.responseQueueToSend)
		close(newPeer.broadcastVQLQuery)
		close(newPeer.gotRawQueryFromPeer)
		close(newPeer.updateTrigger)
	}()

	for {
		if newPeer.RemoveSignal {
			break
		}
		// fmt.Fprintf(conn, text+"\n")
		reply := make([]byte, 1024)
		n, err := conn.Read(reply)
		newPeer.Stats.BytesIn += int64(n)
		if err != nil {
			fmt.Printf("[peer %s] Read from peer failed: %s\n", newPeer.connString(), err.Error())
			conn.Close()
			p.Stats.connectionReadFailureCounter++
			// pause = initialPause
			break
		}
		switch {
		case bytes.HasPrefix(reply[:n], PEER_QUERY_TYPE):
			select {
			case newPeer.gotRawQueryFromPeer <- reply[:n]:
			default:
			}

		case bytes.HasPrefix(reply[:n], PEER_RESPONSE_TYPE):
			select {
			case newPeer.queryResponseReceived <- reply[:n]:
			default:
			}

		}
		// // fmt.Printf("[peer %s] %s\n", newPeer.connString(), reply[:n])
		// resp, err := p.ParsePeerResponse(newPeer, reply[:n])
		// if err != nil {
		// 	fmt.Println(err)
		// 	continue
		// }
		// if resp.DisconnectSignal {
		// 	break
		// }
	}
	// }
}

type PeerResponse struct {
	r                *Response
	Error            error
	DisconnectSignal bool
}

func infoPeer(p *Peer) (info []string) {
	info = append(info, "# VQL")
	info = append(info, fmt.Sprintf("id:%s", p.ID))
	info = append(info, fmt.Sprintf("listen_addr:%s", p.ListenAddr))
	info = append(info, fmt.Sprintf("listen_port:%d", p.ListenPort))
	return info
}

type PeerRequest struct {
	Payload []byte
}

func (r *PeerRequest) Execute() error {
	return nil
}

func (p *Peer) ParsePeerResponse(c *VQLClient, input []byte) (*Response, error) {
	if !bytes.HasPrefix(input, PEER_RESPONSE_TYPE) {
		return nil, fmt.Errorf("invalid byte response")
	}
	input = bytes.TrimLeft(input, string(append(PEER_RESPONSE_TYPE, controlByte...)))
	var q *Query
	var err error
	q, err = p.ParseRawQuery(c, input)
	if err != nil {
		return nil, err
	}
	if len(q.parsed) < 2 {
		fmt.Println(string(bytes.Join(q.parsed, []byte(""))))
		return nil, fmt.Errorf("no response found")
	}
	if !bytes.HasPrefix(q.parsed[0], []byte("id=")) {
		return nil, fmt.Errorf("invalid response id")
	}
	rid := string(q.parsed[0][3:])
	if rid == "" {
		return nil, fmt.Errorf("no response id found")
	}
	q = &Query{}
	q.id = rid
	r := NewResponse(q)
	r.Payload = make([][]byte, 1)
	// r.Payload[0] = q.parsed[1]
	return r, nil

}
func (p *Peer) ParsePeerQuery(c *VQLClient, input []byte) (*Query, error) {
	if !bytes.HasPrefix(input, PEER_QUERY_TYPE) {
		return nil, fmt.Errorf("invalid byte query")
	}
	input = bytes.TrimLeft(input, string(append(PEER_QUERY_TYPE, controlByte...)))
	var q *Query
	var err error
	q, err = p.ParseRawQuery(c, input)
	if err != nil {
		return nil, err
	}
	if len(q.parsed) <= 1 {
		fmt.Println(string(bytes.Join(q.parsed, []byte(""))))
		return nil, fmt.Errorf("no query found")
	}
	if !bytes.HasPrefix(q.parsed[0], []byte("id=")) {
		return nil, fmt.Errorf("invalid query id")
	}
	qid := string(q.parsed[0][3:])
	if qid == "" {
		return nil, fmt.Errorf("no query id found")
	}

	q, err = p.ParseRawQuery(c, q.parsed[1])
	if err != nil {
		return nil, err
	}
	q.id = qid
	return q, nil
}

func (p *Peer) RemoteExecute(remotePeer *Peer, q *Query) (*Response, error) {
	lock.Lock()
	remotePeer.queryWaiting[q.id] = make(chan *Response)
	lock.Unlock()
	defer delete(remotePeer.queryWaiting, q.id)

	select {
	case remotePeer.broadcastVQLQuery <- q:
	default:
	}
	fmt.Println("waiting query", q.id)
	select {
	case resp := <-remotePeer.queryWaiting[q.id]:
		fmt.Println("got response", resp.q.id)
		return resp, nil
	case <-time.After(QUERY_TIMEOUT * time.Second):
		return nil, fmt.Errorf("timeout waiting response for query %s", q.id)
	}

}

func (p *Peer) ResponseReader(id int, remotePeer *Peer, c *VQLClient) {
	l := p.l.NewLogger(logger.Fields{"resp_reader": id, "remote_peer": remotePeer.connString()})
	l.Debug(nil, "Starting response reader")
	defer func() {
		l.Debug(nil, "Exited response reader")
	}()
	for {
		select {
		case data, more := <-remotePeer.queryResponseReceived:
			if !more {
				return
			}
			r, err := p.ParsePeerResponse(nil, data)
			if err != nil {
				fmt.Println("Unable to read peer response", string(data))
				continue
			}
			fmt.Println("Received response for query", r.q.id)
			select {
			case remotePeer.queryWaiting[r.q.id] <- r:
			default:
			}
		}
	}
}

func (p *Peer) QueryReader(remotePeer *Peer, c *VQLClient) {
	fmt.Println("Starting query reader for peer", remotePeer.ID)
	defer func() {
		fmt.Println("Exited query reader for peer", remotePeer.ID)
	}()
	for {
		select {
		case d, more := <-remotePeer.gotRawQueryFromPeer:
			if !more {
				return
			}
			query, err := p.ParsePeerQuery(c, d)
			if err != nil {
				fmt.Println("query parse error", err)
				select {
				case remotePeer.responseQueueToSend <- NewPeerResponseError(query, err):
				default:
				}
				continue
			}
			query.FromPeer = true
			fmt.Println("Query from peer:", query.words())
			resp, err := query.Execute()
			fmt.Println("query executed", query.id)
			if err != nil {
				fmt.Println("query execution error", err)
				select {
				case remotePeer.responseQueueToSend <- NewPeerResponseError(query, err):
				default:
				}
				continue
			}
			fmt.Println("queuing response for query", query.id)
			select {
			case remotePeer.responseQueueToSend <- resp:
			default:
			}

			fmt.Println("response queued for query", query.id)
		}
	}
}

func (p *Peer) QueryWriter(remotePeer *Peer, c *VQLClient) {
	fmt.Println("Starting query writer for peer", remotePeer.ID)
	defer func() {
		fmt.Println("Exited query writer for peer", remotePeer.ID)
	}()
	for {
		select {
		case q, more := <-remotePeer.broadcastVQLQuery:
			if !more {
				return
			}
			data := q.PeerQueryEncode()
			_, err := remotePeer.RemoteConn.Write(data)
			if err != nil {
				fmt.Println(err)
				return
			}
			// select {
			// case resp := <-remotePeer.queryWaiting[q.id]:
			// 	fmt.Println("got response", resp.q.id)
			// case <-time.After(QUERY_TIMEOUT * time.Second):
			// 	fmt.Println("timeout waiting response for query", q.id)
			// }
		}
	}
}

func (p *Peer) ResponseWriter(remotePeer *Peer, c *VQLClient) {
	fmt.Println("Starting response writer for peer", remotePeer.ID)
	defer func() {
		fmt.Println("Exited response writer for peer", remotePeer.ID)
	}()
	for {
		select {
		case r, more := <-remotePeer.responseQueueToSend:
			if !more {
				return
			}
			data := r.PeerResponseEncode()
			_, err := remotePeer.RemoteConn.Write(data)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("response sent", r.q.id)
		}
	}
}

func (p *Peer) Run() {
	s, err := tcp.NewTCPServer(p.ListenAddr, p.ListenPort)
	if err != nil {
		panic(err)
	}
	go p.Mesh.registrator()
	go p.walWriter.Run()
	defer func() {
		p.walWriter.Close()

	}()
	p.tcpServer = s
	s.Run("peer", p.HandlePeerRequest)
}

func (q *Query) waitResponse() (*Response, error) {

	return nil, fmt.Errorf("Timeout waiting response for query %s", q.raw)
}

// func (p *Peer) getRemoteID() string {
// 	for {
// 		q := &Query{
// 			raw: []byte("PEER ID\r\n"),
// 		}
// 		p.broadcastVQLQuery <- q
// 		resp, err := p.waitResponse(q)
// 		if err != nil {
// 			fmt.Println(err)
// 			time.Sleep(time.Second)
// 			continue
// 		}
// 		p.ID = string(resp.Payload[0])
// 		time.Sleep(time.Second)
// 	}
// }

func (p *Peer) HandlePeerRequest(s *tcp.TCPServer, conn net.Conn) {
	fmt.Printf("[peer] Serving %s\n", conn.RemoteAddr().String())
	c := NewVQLClient(-1, fmt.Sprintf("peer-%s", p.ID), conn, p.vqlTCPServer)
	remotePeerAddr, remotePeerPort, err := utils.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		fmt.Println(err)
		return
	}
	remotePeer, err := NewRemotePeer(remotePeerAddr, remotePeerPort)
	if err != nil {
		fmt.Println(err)
		return
	}
	remotePeer.RemoteConn = conn

	for i := 0; i < TCP_WORKERS_PER_PEER; i++ {
		go p.ResponseReader(i, remotePeer, c)
		go p.ResponseWriter(remotePeer, c)
		go p.QueryReader(remotePeer, c)
		go p.QueryWriter(remotePeer, c)
	}

	defer func() {
		p.Mesh.deregister <- remotePeer
		conn.Close()
		close(remotePeer.queryResponseReceived)
		close(remotePeer.responseQueueToSend)
		close(remotePeer.broadcastVQLQuery)
		close(remotePeer.gotRawQueryFromPeer)
	}()
	p.Mesh.register <- remotePeer
	for {
		buf := make([]byte, 1024)
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("[peer] Error reading:", err.Error())
			break
		}
		data := buf[:reqLen]
		switch {
		case bytes.HasPrefix(data, PEER_QUERY_TYPE):
			fmt.Println("queing incoming query")
			select {
			case remotePeer.gotRawQueryFromPeer <- data:
			default:
			}
			fmt.Println("incoming query queued")
		case bytes.HasPrefix(data, PEER_RESPONSE_TYPE):
			fmt.Println("queing incoming response")
			select {
			case remotePeer.queryResponseReceived <- data:
			default:
			}
			fmt.Println("incoming response queued")
		default:
			fmt.Println("Unknown data received:", string(data))
		}
	}
	fmt.Printf("[peer] Connection closed %s\n", conn.RemoteAddr().String())
}

func (p *Peer) Shutdown() {
	p.walWriter.Close()
	<-p.walWriter.WaitTerminate
	fmt.Println("Peer shutdown")
}

func (p *Peer) Ready() bool {
	if p.ConnectionStatus() != PEER_STATUS_CONNECTED ||
		p.ID == "" {
		return false
	}
	return true
}

func (p *Peer) Updater(remotePeer *Peer) {
	fmt.Println("Peer updater started")
	defer func() {
		fmt.Println("Peer updater exited")
	}()
	if remotePeer.ID == "" {
		select {
		case remotePeer.updateTrigger <- true:
		default:
		}
	}
	for {
		select {
		case _, more := <-remotePeer.updateTrigger:
			if !more {
				return
			}
			fmt.Println("Querying peer info of", remotePeer.connString())
			q := NewSimpleQuery("peer id\r\n")

			resp, err := p.RemoteExecute(remotePeer, q)
			if err != nil {
				remotePeer.RemoteConn.Close()
				return
			}
			fmt.Println("-----------", string(resp.FormattedPayload()))

			remotePeer.ID = string(resp.FormattedPayload())
		}
	}
	// q := NewSimpleQuery("PEER ID")
	// select {
	// case p.broadcastVQLQuery <- q:
	// 	fmt.Println("vql published to peer", p.ID)
	// default:
	// }
	// // resp := <-remotePeer.queryWaiting[r.q.id]
	// return nil
}

func (p *Peer) PublishVQL(query *Query) {
	// TODO: publish vql to the Peer Leader only who will dispatch query to
	// followers.
	// We can implement a region ID per cluster in order to we send queries to the
	// leader of the other regions.
	// It reduces network usage in high latency networks

	for p := range p.Mesh.Peers {
		if !p.Ready() {
			continue
		}
		select {
		case p.broadcastVQLQuery <- query:
			fmt.Println("vql published to peer", p.ID)
		}
		p.Stats.BytesOut += int64(len(query.raw))
	}
}
