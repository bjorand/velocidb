package core

import (
	"fmt"
	"strings"
)

type Response struct {
	Payload          [][]byte
	DisconnectSignal bool
	Type             string
	q                *Query
}

func NewResponse(q *Query) *Response {
	return &Response{
		q:       q,
		Payload: make([][]byte, 1024),
	}
}

func NewPeerResponseError(q *Query, err error) *Response {
	r := NewResponse(q)
	r.Payload = append(r.Payload, []byte(fmt.Sprintf("-%s\r\n", err.Error())))
	return r
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
		// payload = []byte(fmt.Sprintf("*%d\r\n", len(r.Payload)))
		// for i := 0; i < len(r.Payload); i++ {
		// 	if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
		// 		payload = append(payload, []byte(fmt.Sprintf("$%d\r\n", len(r.Payload[i])))...)
		// 	}
		// 	payload = append(payload, r.Payload[i]...)
		// 	if !bytes.HasPrefix(r.Payload[i], []byte("*")) {
		// 		payload = append(payload, []byte("\r\n")...)
		// 	}
		// }
		payload = formattedArray(r.Payload)
	}
	return payload
}

func (r *Response) PeerResponseEncode() []byte {
	var qid string
	switch {
	case r.q == nil:
		qid = "-1"
	default:
		qid = r.q.id
	}
	var data [][]byte
	data = append(data, []byte(fmt.Sprintf("id=%s", qid)))
	data = append(data, r.FormattedPayload())
	payload := append(PEER_RESPONSE_TYPE, controlByte...)
	payload = append(payload, formattedArray(data)...)
	return payload
}
