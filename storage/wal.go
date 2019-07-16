package storage

import (
	"fmt"
	"log"
	"os"
)

type walFile struct {
	id   int
	size int
	wr   *WalFileWriter
}

type WalFileWriter struct {
	data          chan ([]byte)
	walFile       *walFile
	walDir        string
	WaitTerminate chan bool
}

func (w *walFile) path() string {
	return fmt.Sprintf("%s/%d.wal", w.wr.walDir, w.id)
}

func NewWalFileWriter(walDir string) *WalFileWriter {
	w := &WalFileWriter{
		walDir: walDir,
		data:   make(chan ([]byte)),
	}
	return w
}

func (writer *WalFileWriter) SyncWrite(data []byte) {
	// TODO get stats here
	writer.data <- data
}

func (writer *WalFileWriter) Run() {
	writer.WaitTerminate = make(chan bool)
	w := &walFile{
		wr: writer,
	}
	writer.walFile = w
	f, err := os.OpenFile(w.path(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Write([]byte("-CLOSED\r\n"))
		f.Close()
		close(writer.WaitTerminate)
		log.Println("Wal file writer exited")
	}()
	log.Println("Wal file writer started")
	f.Write([]byte("-WAL 0\r\n"))
	for {
		select {
		case data, more := <-writer.data:
			if !more {
				return
			}
			data = append(data, "\r\n"...)
			f.Write(data)
		}

	}
}

func (writer *WalFileWriter) Close() {
	close(writer.data)
}
