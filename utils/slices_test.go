package utils

import "testing"

func TestLpopString(t *testing.T) {
	arr := []string{"A", "B"}
	output, _ := lpopString(&arr)
	expected := "A"
	if output != expected {
		t.Errorf("Got %+v, want %v", output, expected)
	}
	outputLength := len(arr)
	expectedLength := 1
	if outputLength != expectedLength {
		t.Errorf("Got %d, want %d", outputLength, expectedLength)
	}
	output, _ = lpopString(&arr)
	expected = "B"
	if output != expected {
		t.Errorf("Got %+v, want %v", output, expected)
	}
	outputLength = len(arr)
	expectedLength = 0
	if outputLength != expectedLength {
		t.Errorf("Got %d, want %d", outputLength, expectedLength)
	}
	outputErr, err := lpopString(&arr)
	if err == nil {
		t.Errorf("Should have got an error here")
	}
	expected = ""
	if outputErr != expected {
		t.Errorf("Got %+v, want %v", outputErr, expected)
	}
	outputLength = len(arr)
	expectedLength = 0
	if outputLength != expectedLength {
		t.Errorf("Got %d, want %d", outputLength, expectedLength)
	}
}
