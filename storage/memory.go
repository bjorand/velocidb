package storage

import "sync"

var (
	lock = sync.RWMutex{}
)

type MemoryStorage struct {
	data map[string][]byte
}

func NewMemoryStorage() *MemoryStorage {
	m := &MemoryStorage{}
	m.data = make(map[string][]byte)
	return m
}

func (m *MemoryStorage) Set(k string, v []byte) {
	lock.Lock()
	m.data[k] = v
	lock.Unlock()
}

func (m *MemoryStorage) Get(k string) []byte {
	return m.data[k]
}
