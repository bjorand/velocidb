package core

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestSanitizeTextInput(t *testing.T) {
	input := " TEST   \r\n\n   \n"
	output := SanitizeTextInput([]byte(input))
	expected := "TEST"
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}

func TestVQLTCPServerParseRawQuery(t *testing.T) {
	client := setup()
	input := []byte("ping\r\n")
	q, err := client.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected := "ping"
	output := q.words()[0]
	if expected != output {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
	input = []byte("peer connect 192.168.0.2\r\n")
	q, err = client.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected = "peer connect 192.168.0.2"
	output = strings.Join(q.words(), " ")
	if expected != output {
		t.Errorf("want %s, got %s", []byte(expected), []byte(output))
	}
	input = []byte("*3\r\n$4\r\npeer\r\n$7\r\nconnect\r\n$11\r\n192.168.0.2\r\n")
	q, err = client.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	expected = "peer connect 192.168.0.2"
	output = strings.Join(q.words(), " ")
	if expected != output {
		t.Errorf("want %s, got %s", []byte(expected), []byte(output))
	}
}

func TestVQLPing(t *testing.T) {
	client := setup()
	var r *Response
	var q *Query
	var err error

	input := []byte("ping\r\n")
	q, err = client.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	r, err = q.Execute()
	if err != nil {
		t.Errorf("Cannot execute query: %+v", err)
	}
	expected := "+PONG\r\n"
	output := r.FormattedPayload()
	if expected != string(output) {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}

	input = []byte("ping foobar\r\n")
	q, err = client.ParseRawQuery(input)
	if err != nil {
		t.Errorf("Cannot parse raw query: %+v", err)
	}
	r, err = q.Execute()
	if err != nil {
		t.Errorf("Cannot execute query: %+v", err)
	}
	expected = "$6\r\nfoobar\r\n"
	output = r.FormattedPayload()
	if expected != string(output) {
		t.Errorf("want %+v, got %+v", []byte(expected), []byte(output))
	}
}

func TestVQLQuit(t *testing.T) {
	client := setup()
	input := []byte("quit\r\n")
	q, err := client.ParseRawQuery(input)
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

// func TestVQLScan(t *testing.T) {
// 	client := setup()
// 	var err error
// 	var q *Query
// 	var r *Response
// 	input := []byte("scan 0\r\n")
// 	q, err = client.ParseRawQuery(input)
// 	if err != nil {
// 		t.Errorf("Cannot parse raw query: %+v", err)
// 	}
// 	r, err = q.Execute()
// 	if err != nil {
// 		t.Errorf("Cannot execute query: %+v", err)
// 	}
// 	expected := "*2\r\n$1\r\n0\r\n*0\r\n"
// 	output := r.FormattedPayload()
// 	if expected != string(output) {
// 		t.Errorf("want %s, got %s", []byte(expected), []byte(output))
// 	}
// 	for i := 0; i < 10; i++ {
// 		q, err = client.ParseRawQuery([]byte(fmt.Sprintf("incr a-%d\r\n", i)))
// 		if err != nil {
// 			t.Errorf("Cannot parse raw query: %+v", err)
// 		}
// 		_, err = q.Execute()
// 		if err != nil {
// 			t.Errorf("Cannot execute query: %+v", err)
// 		}
// 	}
// 	input = []byte("scan 0\r\n")
// 	q, err = client.ParseRawQuery(input)
// 	if err != nil {
// 		t.Errorf("Cannot parse raw query: %+v", err)
// 	}
// 	r, err = q.Execute()
// 	if err != nil {
// 		t.Errorf("Cannot execute query: %+v", err)
// 	}
// 	expected = "*2\r\n$1\r\n0\r\n*10"
// 	output = r.FormattedPayload()
// 	if !strings.HasPrefix(string(output), expected) {
// 		t.Errorf("want %s, got %s", []byte(expected), []byte(output))
// 	}
//
// }

func TestVQLQueries(t *testing.T) {
	b := make([]byte, 1000)
	// TODO generate a binary file here
	f, err := ioutil.ReadFile("/Users/meister/consul_1.5.1_darwin_amd64.zip")
	if err != nil {
		panic(err)
	}
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
		fmt.Sprintf("*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$%d\r\n%x\r\n", len(b)*2, b), "+OK\r\n",
		"get key", fmt.Sprintf("$%d\r\n%x\r\n", len(b)*2, b),
		fmt.Sprintf("*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$%d\r\n%x\r\n", len(f)*2, f), "+OK\r\n",
		"client setname foobar", "+OK\r\n",
		"client getname", "$6\r\nfoobar\r\n",
		"ping foobar", "$6\r\nfoobar\r\n",
	}
	client := setup()
	for i := 0; i < len(suites); i++ {
		input := []byte(suites[i])
		expected := suites[i+1]
		q, errP := client.ParseRawQuery(input)
		if errP != nil {
			t.Errorf("Cannot parse raw query: %+v", errP)
		}
		r, errQ := q.Execute()
		if errQ != nil {
			t.Fatalf("Cannot execute query: %+v", errQ)
		}
		output := r.FormattedPayload()
		if expected != string(output) {
			t.Errorf("want %s, got %s", []byte(expected), []byte(output))
		}
		i++
	}
}
