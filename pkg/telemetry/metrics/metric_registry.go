package metrics

import (
	"sync"
)

var (
	initialized bool
	initMutex   sync.Mutex
)

func EnsureInitialized() {
	initMutex.Lock()
	defer initMutex.Unlock()

	if !initialized {
		InitMetrics()
		initialized = true
	}
}