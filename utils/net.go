package utils

import (
	"net"
	"strconv"
)

// SplitHostPort splits a string built of an host and a port like 127.0.0.1:5000
func SplitHostPort(h string) (string, int64, error) {
	host, portS, err := net.SplitHostPort(h)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.ParseInt(portS, 10, 64)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
