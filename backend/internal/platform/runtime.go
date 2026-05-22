package platform

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type RandomIDGenerator struct{}

func (RandomIDGenerator) NewID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
