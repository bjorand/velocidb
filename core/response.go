package core

import (
	"bytes"
	"fmt"
	"strings"
)

type Response struct {
	Payload          [][]byte
	DisconnectSignal bool
	Type             string
}

func NewResponse() *Response {
	return &Response{
		// Payload: make([][]byte),
	}
}

func (r *Response) PayloadString(s []byte) {
	r.Payload = make([][]byte, 1)
	r.Payload[0] = s
}

func (r *Response) OK() {
	r.PayloadString([]byte("OK"))
}

func SanitizeTextInput(data []byte) string {
	d := string(data)
	d = strings.Trim(d, " \r\n")
	return d
}

func Sanitize(data []byte) []byte {
	// d := string(data)
	// return strings.Trim(d, " \r\n")
	return data
}

func ParseRawResponse(data []byte) (*Response, error) {
	r := NewResponse()
	if len(r.Payload) > 0 {
		r.Payload[0] = Sanitize(data)
	}
	// if r.Payload == "+ATH0" {
	// 	r.DisconnectSignal = true
	// }
	return r, nil
}

func (r *Response) Size() int {
	return len(r.Payload)
}

func (r *Response) isBulkString() bool {
	if r.Type == typeBulkString {
		return true
	}
	return false
}

func (r *Response) isArray() bool {
	if r.Type == typeArray {
		return true
	}
	return false
}

func (r *Response) isInteger() bool {
	if r.Type == typeInteger {
		return true
	}
	return false
}

func (r *Response) isNullBulkString() bool {
	if r.Type == typeBulkString && len(r.Payload[0]) == 0 {
		return true
	}
	return false
}

func (r *Response) FormattedPayload() []byte {
	var payload []byte

	if len(r.Payload) == 1 && !r.isArray() {

		if r.isBulkString() {
			if r.isNullBulkString() {
				payload = []byte("$-1")
			} else {
				payload = []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[0])))
				payload = append(payload, r.Payload[0]...)
			}
		} else if r.isInteger() {
			payload = []byte(fmt.Sprintf(":%s", r.Payload[0]))
		} else {
			payload = []byte(fmt.Sprintf("+%s", r.Payload[0]))
		}
		payload = append(payload, "\r\n"...)

	} else {
		payload = []byte(fmt.Sprintf("*%d\r\n", len(r.Payload)))
		for i := 0; i < len(r.Payload); i++ {

			if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
				payload = append(payload, []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[i])))...)
			}
			payload = append(payload, r.Payload[i]...)
			if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
				payload = append(payload, []byte("\r\n")...)
			}
		}

	}
	return payload
}
