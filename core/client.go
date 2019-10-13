package core

import (
	"net"
)

var (
	endByte = []byte("\r\n")
)

type VQLClient struct {
	id           int64
	name         string
	conn         net.Conn
	vqlTCPServer *VQLTCPServer
}

func NewVQLClient(id int64, name string, conn net.Conn, v *VQLTCPServer) *VQLClient {
	return &VQLClient{
		id:           id,
		name:         name,
		vqlTCPServer: v,
		conn:         conn,
	}
}

func (c *VQLClient) ParseRawQuery(input []byte) (*Query, error) {
	return c.vqlTCPServer.Peer.ParseRawQuery(c, input)
}
