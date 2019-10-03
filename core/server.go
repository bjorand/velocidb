package core

import (
	"fmt"
	"net"
	"sort"

	tcp "github.com/bjorand/velocidb/tcp"
)

var (
	lastConnectionID int64
)

type VQLTCPServer struct {
	Peer       *Peer
	ListenAddr string
	ListenPort int64
	clients    map[*VQLClient]bool
}

func NewVQLTCPServer(peer *Peer, listenAddr string, listenPort int64) (*VQLTCPServer, error) {

	v := &VQLTCPServer{
		Peer:       peer,
		ListenAddr: listenAddr,
		ListenPort: listenPort,
		clients:    make(map[*VQLClient]bool),
	}
	peer.vqlTCPServer = v
	return v, nil
}

func (v *VQLTCPServer) clientNextID() int64 {
	lock.Lock()
	defer lock.Unlock()
	lastConnectionID++
	return lastConnectionID
}

func (v *VQLTCPServer) Run() {
	s, err := tcp.NewTCPServer(v.ListenAddr, v.ListenPort)
	if err != nil {
		panic(err)
	}
	s.Run("vql", v.HandleVQLRequest)
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
			fmt.Println("[vql] Error reading:", err.Error())
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
			query, err = v.Peer.ParseRawQuery(client, buf[:reqLen])
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

	fmt.Println("[vql] shutdown")
}

func infoStorage(v *VQLTCPServer) (info []string) {
	info = append(info, "# Keyspace")
	info = append(info, fmt.Sprintf("db0:keys=%d", len(v.Peer.storage.Keys("*"))))
	return info
}

func infoWal(v *VQLTCPServer) (info []string) {
	walFilesize, _ := v.Peer.walWriter.WalFile.Size()
	info = append(info, "# Wal")
	info = append(info, fmt.Sprintf("current_wal_file:%s", v.Peer.walWriter.WalFile.Path()))
	info = append(info, fmt.Sprintf("current_wal_file_size_bytes:%d", walFilesize))
	info = append(info, fmt.Sprintf("write_bytes:%d", v.Peer.walWriter.BytesWritten))
	info = append(info, fmt.Sprintf("write_ops:%d", v.Peer.walWriter.WriteOps))
	return info
}

func infoServer(peer *Peer) (info []string) {
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

func infoVQL(v *VQLTCPServer) (info []string) {
	info = append(info, "# VQL")
	info = append(info, fmt.Sprintf("connected_clients:%d", len(v.clients)))
	return info
}
