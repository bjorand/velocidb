package server

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

type TCPServer struct {
	Port int64
	Host string
}

func NewTCPServer(host string, port int64) (*TCPServer, error) {
	return &TCPServer{
		Host: host,
		Port: port,
	}, nil
}

func (s *TCPServer) Run(id string, handleRequesFunc func(*TCPServer, net.Conn)) {
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		fmt.Printf("[%s] Error listening: %s\n", id, err.Error())
		os.Exit(1)
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())
	fmt.Printf("[%s] Listening on:%d\n", id, s.Port)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("[%s] Error accepting: %s\n", id, err.Error())
			os.Exit(1)
		}
		go handleRequesFunc(s, conn)
	}
}
