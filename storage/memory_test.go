package storage

import "testing"

func TestMemoryStorage(t *testing.T) {
	m := NewMemoryStorage()
	m.Set("key", []byte(" hello\tfoobar\t "))
	output := m.Get("key")
	expected := []byte(" hello\tfoobar\t ")
	if string(expected) != string(output) {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}
