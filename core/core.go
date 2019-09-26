package core

import (
	"sync"

	storagePkg "github.com/bjorand/velocidb/storage"
)

var (
	lock    = sync.RWMutex{}
	storage *storagePkg.MemoryStorage
)
