package storage

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/gobwas/glob"
)

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

func (m *MemoryStorage) FlushData() {
	m.data = make(map[string][]byte)
}

func (m *MemoryStorage) Set(k string, v []byte) {
	lock.Lock()
	// TODO we could implement metrics to get locked time
	m.data[k] = v
	lock.Unlock()
}

func (m *MemoryStorage) Get(k string) []byte {
	lock.RLock()
	d := m.data[k]
	lock.RUnlock()
	return d
}

func (m *MemoryStorage) Incr(k string) ([]byte, error) {
	lock.RLock()
	s, ok := m.data[k]
	lock.RUnlock()
	if ok {
		i, err := strconv.Atoi(string(s))
		if err != nil {
			return nil, err
		}
		i = i + 1
		m.Set(k, []byte(fmt.Sprintf("%d", i)))
		return []byte(fmt.Sprintf("%d", i)), nil
	}
	m.Set(k, []byte("1"))
	return []byte("1"), nil
}

func (m *MemoryStorage) Decr(k string) ([]byte, error) {
	lock.RLock()
	s, ok := m.data[k]
	lock.RUnlock()
	if ok {
		i, err := strconv.Atoi(string(s))
		if err != nil {
			return nil, err
		}
		i = i - 1
		m.Set(k, []byte(fmt.Sprintf("%d", i)))
		return []byte(fmt.Sprintf("%d", i)), nil
	}
	m.Set(k, []byte("-1"))
	return []byte("-1"), nil
}

func (m *MemoryStorage) Del(k string) bool {
	lock.Lock()
	defer lock.Unlock()
	_, ok := m.data[k]
	if ok {
		delete(m.data, k)
		return true
	}
	return false
}

func (m *MemoryStorage) Keys(filter string) (keys []string) {
	var g glob.Glob
	g = glob.MustCompile(filter)
	for k := range m.data {
		if g.Match(k) {
			keys = append(keys, k)
		}
	}
	return keys
}
