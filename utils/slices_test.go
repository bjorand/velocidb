package utils

import (
	"strings"
	"testing"
)

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

func TestSliceStringsEquals(t *testing.T) {
	testCases := map[string]map[string]bool{
		"A,B": {
			"A,B":   true,
			"A":     false,
			"":      false,
			"B,A":   false,
			"A,B, ": false,
		},
	}

	for input1s, testCase := range testCases {
		input1 := strings.Split(input1s, ",")
		for input2s, expected := range testCase {
			input2 := strings.Split(input2s, ",")
			output := SliceEquals(input1, input2)
			if output != expected {
				t.Errorf("Got %+v, want %+v, input1: %+v, input2: %+v", output, expected, input1, input2)
			}
		}
	}
}
