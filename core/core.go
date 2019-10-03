package core

import (
	"sync"
)

var (
	lock = sync.Mutex{}
	// client *VQLClient
)
