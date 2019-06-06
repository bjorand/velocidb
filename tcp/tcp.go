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

func (s *TCPServer) Run(handleRequesFunc func(*TCPServer, net.Conn)) {
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer l.Close()
	rand.Seed(time.Now().Unix())
	fmt.Printf("Listening on %s:%d\n", s.Host, s.Port)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		// go s.handleRequest(conn)

		go handleRequesFunc(s, conn)
	}
}
