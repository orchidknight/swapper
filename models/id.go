package models

import (
	"sync"
	"time"
)

const (
	idSequenceBits  = 16
	idSequenceMask  = 1<<idSequenceBits - 1
	idTimestampMask = 1<<(64-idSequenceBits) - 1
)

var newIDState struct {
	mu        sync.Mutex
	lastMilli uint64
	sequence  uint64
}

// NewID returns a process-local unique uint64 ID.
//
// Layout: high 48 bits are logical Unix milliseconds, low 16 bits are a
// per-millisecond sequence. This gives 65,536 IDs per logical millisecond with
// zero collision probability inside one process. If this limit is reached before
// the wall clock advances, the logical millisecond is advanced to preserve
// uniqueness, so generated IDs may briefly run ahead of wall time.
//
// NewID does not use randomness, so rand.Read entropy failures are not possible.
func NewID() uint64 {
	newIDState.mu.Lock()
	defer newIDState.mu.Unlock()

	now := currentUnixMilli()
	if now > newIDState.lastMilli {
		newIDState.lastMilli = now
		newIDState.sequence = 0
	} else if newIDState.sequence < idSequenceMask {
		newIDState.sequence++
	} else {
		newIDState.lastMilli = (newIDState.lastMilli + 1) & idTimestampMask
		newIDState.sequence = 0
	}

	return newIDState.lastMilli<<idSequenceBits | newIDState.sequence
}

func currentUnixMilli() uint64 {
	now := time.Now().UnixMilli()
	if now < 0 {
		return 0
	}

	return uint64(now) & idTimestampMask
}
