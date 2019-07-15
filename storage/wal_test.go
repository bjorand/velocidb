package storage

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var walDirTest string

func setup() {
	var err error
	walDirTest, err = ioutil.TempDir("/tmp", "testWal")
	if err != nil {
		panic(err)
	}
}

func teardown() {
	defer os.RemoveAll(walDirTest)
}

func TestWalFileWriter(t *testing.T) {
	setup()
	defer teardown()
	wfw := NewWalFileWriter(walDirTest)
	go wfw.Run()
	wfw.SyncWrite([]byte("foobar"))

	time.Sleep(1 * time.Second)
	output, err := ioutil.ReadFile(wfw.walFile.path())
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte("$0\r\nfoobar\r\n")
	if string(expected) != string(output) {
		t.Fatalf("want %s, got %s", expected, output)
	}
	wfw.SyncWrite([]byte("data"))

	wfw.Close()
	output, err = ioutil.ReadFile(wfw.walFile.path())
	if err != nil {
		t.Fatal(err)
	}
	expected = []byte("$0\r\nfoobar\r\ndata\r\n+CLOSED\r\n")
	if string(expected) != string(output) {
		t.Fatalf("want %s, got %s", expected, output)
	}
}
