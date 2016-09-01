// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import "encoding/binary"

func BEUint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func BEUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func BEUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// BEUvarint16 convert the int16 type of varint(varying-length integer) to the binary data of big endian format byte order.
func BEUvarint16(i uint16) []byte {
	dst := [2]byte{}
	binary.BigEndian.PutUint16(dst[:], uint16(i))
	return dst[:]
}

// BEUvarint32 convert the int32 type of varint(varying-length integer) to the binary data of big endian format byte order.
func BEUvarint32(i uint32) []byte {
	dst := [4]byte{}
	binary.BigEndian.PutUint32(dst[:], uint32(i))
	return dst[:]
}

// BEUvarint64 convert the int64 type of varint(varying-length integer) to the binary data of big endian format byte order.
func BEUvarint64(i uint64) []byte {
	dst := [8]byte{}
	binary.BigEndian.PutUint64(dst[:], uint64(i))
	return dst[:]
}
