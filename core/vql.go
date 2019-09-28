package core

var (
	verbs          = []string{"quit", "peer"}
	firstByteArray = []byte("*")
)

const (
	typeArray        = "*"
	typeInteger      = ":"
	typeSimpleString = "+"
	typeBulkString   = "$"
	typeError        = "-"
)
