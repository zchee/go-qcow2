// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import "encoding/binary"

// ToBigEndian32 convert the int32 type of varint(varying-length integer) to the binary data of big endian format byte order.
func ToBigEndian32(i int32) []byte {
	dst := [4]byte{}
	binary.BigEndian.PutUint32(dst[:], uint32(i))
	return dst[:]
}

// ToBigEndian64 convert the int64 type of varint(varying-length integer) to the binary data of big endian format byte order.
func ToBigEndian64(i int64) []byte {
	dst := [8]byte{}
	binary.BigEndian.PutUint64(dst[:], uint64(i))
	return dst[:]
}
