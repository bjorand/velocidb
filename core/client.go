package core

import (
	"bytes"
	"net"

	"github.com/google/uuid"
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
