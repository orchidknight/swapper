package models

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

func NewID() uint64 {
	// use current timestamp in milliseconds for the high 48 bits
	//nolint
	now := uint64(time.Now().UnixMilli()) << 16

	// generate 16 bits of randomness for the lower part
	var rnd [2]byte
	_, _ = rand.Read(rnd[:]) // ignore error because entropy failure is extremely rare
	randomPart := uint64(binary.BigEndian.Uint16(rnd[:]))

	// combine timestamp and randomness into a unique 64-bit ID
	return now | randomPart
}
