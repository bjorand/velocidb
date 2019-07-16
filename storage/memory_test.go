package storage

import "testing"

func TestMemoryStorage(t *testing.T) {
	var err error
	m := NewMemoryStorage()
	m.Set("key", []byte(" hello\tfoobar\t "))
	output := m.Get("key")
	expected := []byte(" hello\tfoobar\t ")
	if string(expected) != string(output) {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	outputK := m.Keys()[0]
	expectedK := "key"
	if string(expectedK) != string(outputK) {
		t.Errorf("want %+v, got %+v", expectedK, outputK)
	}
	outputD := m.Del("key")
	expectedD := true
	if expectedD != outputD {
		t.Errorf("want %+v, got %+v", expectedD, outputD)
	}
	outputD = m.Del("key")
	expectedD = false
	if expectedD != outputD {
		t.Errorf("want %+v, got %+v", expectedD, outputD)
	}
	output, err = m.Incr("key")
	if err != nil {
		t.Error(err)
	}
	expected = []byte("1")
	if string(expected) != string(output) {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	output, err = m.Incr("key")
	if err != nil {
		t.Error(err)
	}
	expected = []byte("2")
	if string(expected) != string(output) {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	m.Set("key", []byte("49"))
	output, err = m.Incr("key")
	if err != nil {
		t.Error(err)
	}
	expected = []byte("50")
	if string(expected) != string(output) {
		t.Errorf("want %s, got %s", expected, output)
	}
	m.Set("key", []byte("foobar"))
	_, err = m.Incr("key")
	if err == nil {
		t.Error("want an error here")
	}
}
