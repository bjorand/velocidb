package utils

import "testing"

func TestSplitHostPort(t *testing.T) {
	input := "localhost:5000"
	outputHost, outputPort, err := SplitHostPort(input)
	if err != nil {
		t.Errorf("%+v", err)
	}
	expectedHost := "localhost"
	var expectedPort int64 = 5000
	if expectedHost != outputHost {
		t.Errorf("want %+v, got %+v", expectedHost, outputHost)
	}
	if expectedPort != outputPort {
		t.Errorf("want %+v, got %+v", expectedPort, outputPort)
	}
	badInput := "localhost"
	_, _, errInput := SplitHostPort(badInput)
	if errInput == nil {
		t.Errorf("should have get an error here")
	}
	badPort := "localhost:foobar"
	_, _, errPort := SplitHostPort(badPort)
	if errPort == nil {
		t.Errorf("should have get an error here")
	}
}
