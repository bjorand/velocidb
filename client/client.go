package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	vql "github.com/bjorand/velocidb/vql"
)

const (
	initalReconnectDelay = 0
)

var (
	server               = flag.String("server", "127.0.0.1:4300", "Server host:port")
	disableAutoReconnect = flag.Bool("no-auto", false, "Disable auto-reconnect to server")
)

type Client struct {
	prompt string
}

func reconnect(server string, next int64) {
	if next < 0 {
		return
	}
	next = next * 2
	if next > 0 {
		fmt.Printf("Reconnecting in %ds...\n", next)
	} else {
		fmt.Println("Reconnecting...")
	}
	time.Sleep(time.Duration(next) * time.Second)
	connect(server, next, false)
}

func connect(server string, next int64, firstConnection bool) {
	conn, err := net.Dial("tcp4", server)
	if err != nil {
		fmt.Println(err.Error())
		if firstConnection {
			return
		}
		reconnect(server, next)
	}
	next = initalReconnectDelay
	defer conn.Close()
	for {
		// read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("\r%s> ", conn.RemoteAddr())
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}
		if strings.Trim(text, " \n") == "" {
			continue
		}
		// send to socket
		fmt.Fprintf(conn, text+"\n")
		// listen for reply
		reply := make([]byte, 1024)
		n, err := conn.Read(reply)
		if err != nil {
			println("Write to server failed:", err.Error())
			conn.Close()
			reconnect(server, next)
		}
		next = initalReconnectDelay
		resp, err := vql.ParseRawResponse(reply[:n])
		if err != nil {
			fmt.Println(err)
			continue
		}
		if resp.DisconnectSignal {
			break
		}
		fmt.Println(resp.Payload)
	}
}

func main() {
	flag.Parse()
	var next int64
	if !*disableAutoReconnect {
		next = initalReconnectDelay
	}
	connect(*server, next, true)
}
