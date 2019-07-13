package storage

import (
	"io/ioutil"
	"testing"
	"time"
)

func TestWalFileWriter(t *testing.T) {
	wfw := NewWalFileWriter()
	go wfw.Run()
	wfw.data <- []byte("toto")
	time.Sleep(1 * time.Second)
	wfw.Close()
	output, err := ioutil.ReadFile("/tmp/0.wal")
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("$0\r\ntoto\r\n+CLOSED\r\n")
	if string(expected) != string(output) {
		t.Errorf("want %s, got %s", expected, output)
	}
}
