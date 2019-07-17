package vql

import (
	"testing"

	peering "github.com/bjorand/velocidb/peering"
)

var (
	testPeer *peering.Peer
)

func setup() {
	var err error
	testPeer, err = peering.NewPeer("localhost", 26000)
	if err != nil {
		panic(err)
	}
}

func TestSanitizeTextInput(t *testing.T) {
	input := " TEST   \r\n\n   \n"
	output := SanitizeTextInput([]byte(input))
	expected := "TEST"
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}

// func TestParseResponseQuit(t *testing.T) {
// 	payload := "+ATH0"
// 	resp, _ := ParseRawResponse([]byte(payload))
// 	output := resp.DisconnectSignal
// 	expected := true
// 	if expected != output {
// 		t.Errorf("want %+v, got %+v", expected, output)
// 	}
// }

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

func TestVQLTCPServerParseRawQuery(t *testing.T) {
	p, err := peering.NewPeer("localhost", 26000)
	if err != nil {
		t.Errorf("Cannot create peer: %+v", err)
	}
	v, err := NewVQLTCPServer(p, "localhost", 26001)
	if err != nil {
		t.Errorf("Cannot create VQL server: %+v", err)
	}
	input := []byte("ping\r\n")
	q, err := v.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected := "ping"
	output := q.text
	if expected != output {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
	input = []byte("peer connect 192.168.0.2\r\n")
	q, err = v.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected = "peer connect 192.168.0.2"
	output = q.text
	if expected != output {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
	input = []byte("*3\r\n$4\r\npeer\r\n$7\r\nconnect\r\n$11\r\n192.168.0.2\r\n")
	q, err = v.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected = "peer connect 192.168.0.2"
	output = q.text
	if expected != output {
		t.Errorf("want %s, got %s", []byte(expected), []byte(output))
	}
}

func TestVQLPing(t *testing.T) {
	setup()
	v, err := NewVQLTCPServer(testPeer, "localhost", 26001)
	if err != nil {
		t.Errorf("Cannot create VQL server: %+v", err)
	}
	input := []byte("ping\r\n")
	q, err := v.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	r, err := q.Execute()
	if err != nil {
		t.Errorf("Cannot execute query: %+v", err)
	}
	expected := "+PONG\r\n"
	output := r.FormattedPayload()
	if expected != string(output) {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
}

func TestVQLQuit(t *testing.T) {
	setup()
	v, err := NewVQLTCPServer(testPeer, "localhost", 26001)
	if err != nil {
		t.Errorf("Cannot create VQL server: %+v", err)
	}
	input := []byte("quit\r\n")
	q, err := v.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	r, err := q.Execute()
	if err != nil {
		t.Errorf("Cannot execute query: %+v", err)
	}
	expected := "+OK\r\n"
	output := r.FormattedPayload()
	if expected != string(output) {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
}

func TestVQLQueries(t *testing.T) {
	suites := []string{
		"*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$4\r\n1337\r\n", "+OK\r\n",
		"*2\r\n$3\r\nget\r\n$3\r\nkey\r\n", "$4\r\n1337\r\n",
		"*2\r\n$3\r\nget\r\n$3\r\nfoobar\r\n", "$-1\r\n",
		"*2\r\n$3\r\ndel\r\n$3\r\nkey\r\n", ":1\r\n",
		"del key key e", ":0\r\n",
		"incr key", ":1\r\n",
		"incr key", ":2\r\n",
		"set key 49", "+OK\r\n",
		"incr key", ":50\r\n",
		"decr key", ":49\r\n",
		"decr z", ":-1\r\n",
		"decr z", ":-2\r\n",
		"del z key", ":2\r\n",
	}
	setup()
	v, err := NewVQLTCPServer(testPeer, "localhost", 26001)
	if err != nil {
		t.Errorf("Cannot create VQL server: %+v", err)
	}
	for i := 0; i < len(suites); i++ {
		input := []byte(suites[i])
		expected := suites[i+1]
		q, errP := v.ParseRawQuery(input)
		if errP != nil {
			t.Errorf("Cannot parse raw query: %+v", errP)
		}
		r, errQ := q.Execute()
		if errQ != nil {
			t.Fatalf("Cannot execute query: %+v", errQ)
		}
		output := r.FormattedPayload()
		if expected != string(output) {
			t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
		}
		i++
	}
}
