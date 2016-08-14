// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

type QCow2 struct {
	Header *QCowHeader
}

// Qcow2 qcow2 image header.
type QCowHeader struct {
	Magic                 []byte      // [0:4] magic: QCOW magic string ("QFI\xfb")
	Version               Version     // [4:8] Version number
	BackingFileOffset     int64       // [8:16] Offset into the image file at which the backing file name is stored.
	BackingFileSize       int32       // [16:20] Length of the backing file name in bytes.
	ClusterBits           int32       // [20:24] Number of bits that are used for addressing an offset whithin a cluster.
	Size                  int64       // [24:32] Virtual disk size in bytes
	CryptMethod           CryptMethod // [32:36] Crypt method
	L1Size                int32       // [36:40] Number of entries in the active L1 table
	L1TableOffset         int64       // [40:48] Offset into the image file at which the active L1 table starts
	RefcountTableOffset   int64       // [48:56] Offset into the image file at which the refcount table starts
	RefcountTableClusters int32       // [56:60] Number of clusters that the refcount table occupies
	NbSnapshots           int32       // [60:64] Number of snapshots contained in the image
	SnapshotsOffset       int64       // [64:72] Offset into the image file at which the snapshot table starts

	// The following fields are only valid for version >= 3
	IncompatibleFeatures int64 // [72:80] Bitmask of incomptible feature
	CompatibleFeatures   int64 // [80:88] Bitmask of compatible feature
	AutoclearFeatures    int64 // [88:96] Bitmask of auto-clear feature

	RefcountOrder int32 // [96:100] Describes the width of a reference count block entry
	HeaderLength  int32 // [100:104] Length of the header structure in bytes

	// ExtensionHeader optional header extensions
	ExtensionHeader []ExtensionHeader
}

// Qcow2Magic QCOW magic string ("QFI\xfb")
var Qcow2Magic = []byte{0x51, 0x46, 0x49, 0xFB}

// Version version number of qcow image format. valid values are 2 and 3.
type Version int

const (
	// Version2 qcow2 image format version2
	Version2 Version = 2
	// Version3 qcow2 image format version3
	Version3 Version = 3

	// Version2HeaderSize is the image header at the beginning of the file
	Version2HeaderSize = 72
	// Version3HeaderSize is directly following the v2 header, up to 104
	Version3HeaderSize = 104
)

// CryptMethod is whether encrypted qcow2 image.
// 0 for no enccyption
// 1 for AES encryption
type CryptMethod int32

const (
	CryptNone CryptMethod = iota
	CryptAES
)

// String implementations of fmt.Stringer
func (cm CryptMethod) String() string {
	if cm == 1 {
		return "AES"
	}
	return "none"
}

type PreallocMode int

const (
	PreallocModeOff PreallocMode = iota
	PreallocModeMetadata
	PreallocModeFalloc
	PreallocModeFull
	PreallocModeMax
)

const (
	MinClusterBits = 9
	MaxClusterBits = 21
)

// HeaderExtensionType indicators the the entries in the optional header area
type HeaderExtensionType int64

const (
	// HdrExtEndOfArea End of the header extension area
	HdrExtEndOfArea HeaderExtensionType = 0x00000000
	// HdrExtBackingFileFormat Backing file format name
	HdrExtBackingFileFormat HeaderExtensionType = 0xE2792ACA
	// HdrExtFeatureNameTable Feature name table
	HdrExtFeatureNameTable HeaderExtensionType = 0x6803f857
	// HdrExtBitmapsExtension Bitmaps extension
	HdrExtBitmapsExtension HeaderExtensionType = 0x23852875

	// Safely ignored other unknown header extension
)

// ExtensionHeader qcow2 optional header extension
type ExtensionHeader struct {
	Type HeaderExtensionType // [:4] Header extension type
	Size int                 // [4:8] Length of the header extension data
	Data []byte              // [8:n] Header extension data
}

// FeatureNameTable optional header extension that contains the name for features used by the image.
type FeatureNameTable struct {
	Type        int // [:1] Type of feature
	BitNumber   int // [1:2] Bit number within the selected feature bitmap
	FeatureName int // [2:48] // Feature name. padded with zeros
}

// BitmapExtension optional header extension.
type BitmapExtension struct {
	NbBitmaps             int // [1:4] The number of bitmaps contained in the image. Must be greater than ro equal to 1
	Reserved              int // [4:8] Reserved, must be zero
	BitmapDirectorySize   int // [8:16] Size of the bitmap directory in bytes. It is the cumulative size of all (nb_bitmaps) bitmap headers
	BitmapDirectoryOffset int // [16:24] Offste into the image file at which the bitmap directory starts.
}

type QCowSnapshotHeader struct {
}

type QCowSnapshotExtraData struct {
}

type QCowSnapshot struct {
}

type Qcow2Cache struct{}

type Qcow2UnknownHeaderExtension struct {
	magic int32
	len   int32
	// next QLIST_ENTRY(Qcow2UnknownHeaderExtension)
	data []int8
}

type QCow2FeatType int

const (
	QCow2FeatTypeIncompatible QCow2FeatType = iota
	QCow2FeatTypeCompatible
	QCow2FeatTypeAutoclear
)

/* Incompatible feature bits */
const (
	QCow2IncompatDirtyBitNr   = 0
	QCow2IncompatCorruptBitNr = 1
	QCow2IncompatDirty        = 1 << QCow2IncompatDirtyBitNr
	QCow2IncompatCorrupt      = 1 << QCow2IncompatCorruptBitNr

	QCow2IncompatMask = QCow2IncompatDirty | QCow2IncompatCorrupt
)

type QCow2CompatLazyRefcountsBitNr int

const (
	QCow2CompatLazyRefcounts QCow2CompatLazyRefcountsBitNr = 1 << iota

	QCow2CompatFeatMask = QCow2CompatLazyRefcounts
)

type QCow2DiscardType int

const (
	QCow2DiscardNever QCow2DiscardType = iota
	QCow2DiscardAlways
	QCow2DiscardRequest
	QCow2DiscardSnapshot
	QCow2DiscardOther
	QCow2DiscardMax
)
