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
	WalFile       *walFile
	walDir        string
	WaitTerminate chan bool
	BytesWritten  int
	WriteOps      int
}

func (w *walFile) Path() string {
	return fmt.Sprintf("%s/%d.wal", w.wr.walDir, w.id)
}

func (w *walFile) Size() (int64, error) {
	fi, err := os.Stat(w.Path())
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
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
	writer.WalFile = w
	f, err := os.OpenFile(w.Path(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
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
			lock.Lock()
			writer.BytesWritten += len(data)
			writer.WriteOps++
			lock.Unlock()
		}

	}
}

func (writer *WalFileWriter) Close() {
	close(writer.data)
}
