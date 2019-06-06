package vql

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	input := " TEST   \r\n\n   \n"
	output := Sanitize([]byte(input))
	expected := "TEST"
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}

func TestParseResponseQuit(t *testing.T) {
	payload := "+ATH0"
	resp, _ := ParseRawResponse([]byte(payload))
	output := resp.DisconnectSignal
	expected := true
	if expected != output {
		t.Errorf("want %+v, got %+v", expected, output)
	}
}
