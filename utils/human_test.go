package utils

import (
	"testing"
)

func TestHumanSizeBytes(t *testing.T) {
	testCases := map[int64]string{
		80:                  "80B",
		8000:                "8KB",
		81000:               "81KB",
		343403940:           "343MB",
		34340394090:         "34GB",
		3434039409023434344: "3434PB",
	}
	for input, expected := range testCases {
		output := HumanSizeBytes(input)
		if output != expected {
			t.Errorf("Got: %s, want:%s", output, expected)
		}
	}
}
