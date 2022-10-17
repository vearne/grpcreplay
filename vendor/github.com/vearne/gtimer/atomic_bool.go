package gtimer

import "sync/atomic"

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock() {}

// atomicBool is a wrapper around uint32 for usage as a boolean value with
// atomic access.
type AtomicBool struct {
	_noCopy noCopy
	value   uint32
}

func NewAtomicBool(flag bool) *AtomicBool {
	ab := &AtomicBool{}
	ab.Set(flag)
	return ab
}

// IsSet returns whether the current boolean value is true
func (ab *AtomicBool) IsSet() bool {
	return atomic.LoadUint32(&ab.value) > 0
}

// Set sets the value of the bool regardless of the previous value
func (ab *AtomicBool) Set(value bool) {
	if value {
		atomic.StoreUint32(&ab.value, 1)
	} else {
		atomic.StoreUint32(&ab.value, 0)
	}
}

func (ab *AtomicBool) IsTrue() bool {
	return ab.IsSet()
}
