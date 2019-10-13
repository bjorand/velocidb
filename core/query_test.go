package core

import (
	"fmt"
	"testing"
)

func TestReadInt(t *testing.T) {
	input := []byte("33994\r\n$99")
	output, outputCursor := readInt(input, 0)
	expected := 33994
	expectedCursor := 7
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	if expectedCursor != outputCursor {
		t.Errorf("want %+v, got %+v", expectedCursor, outputCursor)
	}
}

func TestPeerQueryEncodeSimple(t *testing.T) {
	q := &Query{}
	q.raw = []byte("PING\r\n")
	q.id = "foo"
	output := q.PeerQueryEncode()
	expected := []byte("0\r\n*2\r\n$6\r\nid=foo\r\n$6\r\nPING\r\n\r\n")
	if string(output) != string(expected) {
		fmt.Println(string(output))
		t.Fatalf("want %+v, got %+v", expected, output)
	}
}

func TestPeerQueryEncodeFormatted(t *testing.T) {
	q := &Query{}
	q.raw = []byte("*1\r\n$3\r\nbar\r\n")
	q.id = "foo"
	output := q.PeerQueryEncode()
	expected := []byte("0\r\n*2\r\n$6\r\nid=foo\r\n$13\r\n*1\r\n$3\r\nbar\r\n\r\n")
	if string(output) != string(expected) {
		fmt.Println(string(output))
		t.Fatalf("want %s, got %s", expected, output)
	}
}
