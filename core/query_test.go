package core

import "testing"

func TestReadInt(t *testing.T) {
	input := []byte("33994\r\n$99")
	output, outputCursor := readInt(input, 0)
	expected := 33994
	expectedCursor := 7
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
	if expectedCursor != outputCursor {
		t.Errorf("want %+v, got %+v", expectedCursor, outputCursor)
	}
}
