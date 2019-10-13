package core

import (
	"testing"
)

func TestPeerResponseEncode(t *testing.T) {
	client := setup()
	q, err := client.ParseRawQuery([]byte("ping\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	q.id = "foo"
	r, err := q.Execute()
	if err != nil {
		t.Fatal(err)
	}
	output := r.PeerResponseEncode()
	expected := []byte("1\r\n*2\r\n$6\r\nid=foo\r\n$7\r\n+PONG\r\n\r\n")
	if string(output) != string(expected) {
		t.Fatalf("want %+v, got %+v", expected, output)
	}
}
