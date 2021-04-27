// from https://github.com/gotify/server/blob/3454dcd60226acf121009975d947f05d41267283/api/stream/once.go
package push

import (
	"sync"
	"sync/atomic"
)

// Modified version of sync.Once (https://github.com/golang/go/blob/master/src/sync/once.go)
// This version unlocks the mutex early and therefore doesn't hold the lock while executing func f().
type once struct {
	m    sync.Mutex
	done uint32
}

func (o *once) Do(f func()) {
	if atomic.LoadUint32(&o.done) == 1 {
		return
	}
	if o.mayExecute() {
		f()
	}
}

func (o *once) mayExecute() bool {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		atomic.StoreUint32(&o.done, 1)
		return true
	}
	return false
}
