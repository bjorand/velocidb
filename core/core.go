package core

import (
	"sync"
)

var (
	lock = sync.RWMutex{}
)
