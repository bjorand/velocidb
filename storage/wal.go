package storage

import (
	"log"
	"os"
)

const (
	WAL_DIR     = "/tmp"
	WAL_PATTERN = "%d.wal"
)

type walFile struct {
	size int
}

type WalFileWriter struct {
	data    chan ([]byte)
	walFile *walFile
}

func NewWalFileWriter() *WalFileWriter {

	w := &WalFileWriter{
		data: make(chan ([]byte)),
	}
	return w
}

func (writer *WalFileWriter) Run() {
	f, err := os.OpenFile("/tmp/0.wal", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Write([]byte("+CLOSED\r\n"))
		f.Close()
		log.Println("Wal file writer exited")
	}()
	log.Println("Wal file writer started")
	f.Write([]byte("$0\r\n"))
	for {
		data, closed := <-writer.data
		data = append(data, "\r\n"...)
		f.Write(data)
		if closed {
			return
		}
	}
}

func (writer *WalFileWriter) Close() {
	close(writer.data)
}
