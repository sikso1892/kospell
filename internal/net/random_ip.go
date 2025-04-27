package net

import (
	"crypto/rand"
	"net"
)

// RandV4 returns a pseudo-random IPv4 string.
func RandV4() string {
	var b [4]byte
	rand.Read(b[:])
	return net.IPv4(b[0], b[1], b[2], b[3]).String()
}
