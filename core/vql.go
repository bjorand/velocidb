package core

var (
	verbs          = []string{"quit", "peer"}
	firstByteArray = []byte("*")
	controlByte    = []byte("\r\n")
)

const (
	typeArray        = "*"
	typeInteger      = ":"
	typeSimpleString = "+"
	typeBulkString   = "$"
	typeError        = "-"
)
