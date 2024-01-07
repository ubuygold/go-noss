package rander

import (
	"crypto/rand"
	"unsafe"
)

type Rander struct {
	charset []byte
}

func NewRander(charset []byte) *Rander {
	return &Rander{
		charset: charset,
	}
}

func (r *Rander) Rand(raw []byte) (nonce string) {
	_, _ = rand.Read(raw)
	for i := 0; i < len(raw); i++ {
		raw[i] = r.charset[int(raw[i])%len(r.charset)]
	}
	return bytesToString(raw)
}

func bytesToString(b []byte) string {
	// Ignore if your IDE shows an error here; it's a false positive.
	p := unsafe.SliceData(b)
	return unsafe.String(p, len(b))
}
