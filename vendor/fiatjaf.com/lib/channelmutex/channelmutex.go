package channelmutex

import "time"

// Mutex is a channel-based mutex that can be used in select statements.
//
// call .New() to create a usable Mutex.
type Mutex struct {
	ch          chan struct{}
	invalidated chan struct{}
}

// New creates a new mutex ready to use.
func New() *Mutex {
	m := &Mutex{
		ch:          make(chan struct{}, 1),
		invalidated: make(chan struct{}),
	}
	m.ch <- struct{}{}
	return m
}

// Lock acquires the mutex.
func (m *Mutex) Lock() {
	<-m.ch
}

// Unlock releases the mutex.
func (m *Mutex) Unlock() {
	select {
	case <-m.invalidated:
		return
	default:
	}

	select {
	case <-m.invalidated:
	case m.ch <- struct{}{}:
	default:
		panic("channelmutex: unlock of unlocked mutex")
	}
}

// TryLock attempts to acquire the mutex without blocking.
func (m *Mutex) TryLock() bool {
	select {
	case <-m.ch:
		return true
	default:
		return false
	}
}

// C returns a receive-only channel that can be used in select{} statements.
//
// When a receive succeeds, the mutex is held and must be released with Unlock.
func (m *Mutex) C() <-chan struct{} {
	return m.ch
}

// Invalidate invalidates the mutex by closing its channel once.
func (m *Mutex) Invalidate() {
	select {
	case <-m.invalidated:
		return
	default:
	}

	close(m.invalidated)
	go func() {
		time.Sleep(time.Millisecond * 100)
		close(m.ch)
	}()
}
