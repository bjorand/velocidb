package core

import (
	"bytes"
	"fmt"
)

var (
	verbs          = []string{"quit", "peer"}
	firstByteArray = []byte("*")
)

const (
	typeArray        = "*"
	typeInteger      = ":"
	typeSimpleString = "+"
	typeBulkString   = "$"
	typeError        = "-"
)

func infoStorage() (info []string) {
	info = append(info, "# Keyspace")
	info = append(info, fmt.Sprintf("db0:keys=%d", len(storage.Keys("*"))))
	return info
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
