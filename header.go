// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

// QcowMagic QCOW magic string ("QFI\xfb")
//  #define QCOW_MAGIC (('Q' << 24) | ('F' << 16) | ('I' << 8) | 0xfb)
var QcowMagic = []byte{0x51, 0x46, 0x49, 0xFB}

// QCow2 represents a qemu QCow2 image format.
type QCow2 struct {
	Header *QCowHeader
}

// QCowHeader represents a header of qcow2 image format.
type QCowHeader struct {
	Magic                 []byte      //     [0:3] magic: QCOW magic string ("QFI\xfb")
	Version               Version     //     [4:7] Version number
	BackingFileOffset     int64       //    [8:15] Offset into the image file at which the backing file name is stored.
	BackingFileSize       int32       //   [16:19] Length of the backing file name in bytes.
	ClusterBits           int32       //   [20:23] Number of bits that are used for addressing an offset whithin a cluster.
	Size                  int64       //   [24:31] Virtual disk size in bytes
	CryptMethod           CryptMethod //   [32:35] Crypt method
	L1Size                int32       //   [36:39] Number of entries in the active L1 table
	L1TableOffset         int64       //   [40:47] Offset into the image file at which the active L1 table starts
	RefcountTableOffset   int64       //   [48:55] Offset into the image file at which the refcount table starts
	RefcountTableClusters int32       //   [56:59] Number of clusters that the refcount table occupies
	NbSnapshots           int32       //   [60:63] Number of snapshots contained in the image
	SnapshotsOffset       int64       //   [64:71] Offset into the image file at which the snapshot table starts
	IncompatibleFeatures  int64       //   [72:79] for version >= 3: Bitmask of incomptible feature
	CompatibleFeatures    int64       //   [80:87] for version >= 3: Bitmask of compatible feature
	AutoclearFeatures     int64       //   [88:95] for version >= 3: Bitmask of auto-clear feature
	RefcountOrder         int32       //   [96:99] for version >= 3: Describes the width of a reference count block entry
	HeaderLength          int32       // [100:103] for version >= 3: Length of the header structure in bytes

	// ExtensionHeader optional header extensions.
	ExtensionHeader []ExtensionHeader
}

// Version represents a version number of qcow image format.
// The valid values are 2 and 3.
type Version int

const (
	// Version2 qcow2 image format version2.
	Version2 Version = 2
	// Version3 qcow2 image format version3.
	Version3 Version = 3

	// Version2HeaderSize is the image header at the beginning of the file.
	Version2HeaderSize = 72
	// Version3HeaderSize is directly following the v2 header, up to 104.
	Version3HeaderSize = 104
)

// CryptMethod represents a whether encrypted qcow2 image.
// 0 for no enccyption
// 1 for AES encryption
type CryptMethod int32

const (
	// CryptNone no encryption.
	CryptNone CryptMethod = iota
	// CryptAES AES encryption.
	CryptAES
)

// String implementations of fmt.Stringer.
func (cm CryptMethod) String() string {
	if cm == 1 {
		return "AES"
	}
	return "none"
}

// PreallocationMode represents a mode of Pre-allocation feature.
type PreallocationMode int

const (
	// PreallocationOff turn off preallocation.
	PreallocationOff PreallocationMode = iota
	// PreallocationMetadata preallocation of metadata only mode.
	PreallocationMetadata
	// PreallocationFalloc preallocation of falloc only mode.
	PreallocationFalloc
	// PreallocationFull full preallocation mode.
	PreallocationFull
	// PreallocationMax preallocation maximum preallocation mode.
	PreallocationMax
)

const (
	// MinClusterBits minimum of cluster bits size.
	MinClusterBits = 9
	// MaxClusterBits maximum of cluster bits size.
	MaxClusterBits = 21
)

// HeaderExtensionType represents a indicators the the entries in the optional header area
type HeaderExtensionType int64

const (
	// HdrExtEndOfArea End of the header extension area.
	HdrExtEndOfArea HeaderExtensionType = 0x00000000
	// HdrExtBackingFileFormat Backing file format name.
	HdrExtBackingFileFormat HeaderExtensionType = 0xE2792ACA
	// HdrExtFeatureNameTable Feature name table.
	HdrExtFeatureNameTable HeaderExtensionType = 0x6803f857
	// HdrExtBitmapsExtension Bitmaps extension.
	HdrExtBitmapsExtension HeaderExtensionType = 0x23852875

	// Safely ignored other unknown header extension
)

// ExtensionHeader represents a optional header extension.
type ExtensionHeader struct {
	Type HeaderExtensionType // [:4] Header extension type
	Size int                 // [4:8] Length of the header extension data
	Data []byte              // [8:n] Header extension data
}

// FeatureNameTable represents a optional header extension that contains the name for features used by the image.
type FeatureNameTable struct {
	// Type type of feature [0:1]
	Type int
	// BitNumber bit number within the selected feature bitmap [1:2]
	BitNumber int
	// FeatureName feature name. padded with zeros [2:48]
	FeatureName int
}

// BitmapExtension represents a optional header extension.
type BitmapExtension struct {
	// NbBitmaps the number of bitmaps contained in the image. Must be greater than ro equal to 1. [1:4]
	NbBitmaps int
	// Reserved reserved, must be zero. [4:8]
	Reserved int
	// BitmapDirectorySize size of the bitmap directory in bytes. It is the cumulative size of all (nb_bitmaps) bitmap headers. [8:16]
	BitmapDirectorySize int
	// BitmapDirectoryOffset offste into the image file at which the bitmap directory starts. [16:24]
	BitmapDirectoryOffset int
}

// SnapshotHeader represents a header of snapshot.
type SnapshotHeader struct {
}

// SnapshotExtraData represents a extra data of snapshot.
type SnapshotExtraData struct {
}

// Snapshot represents a snapshot.
type Snapshot struct {
}

// Cache represents a cache.
type Cache struct {
}

// UnknownHeaderExtension represents a unknown of header extension.
type UnknownHeaderExtension struct {
	Magic int32
	Len   int32
	// Next QLIST_ENTRY(Qcow2UnknownHeaderExtension)
	Data []int8
}

// FeatureType represents a type of feature.
type FeatureType int

const (
	// IncompatibleFeature incompatible feature.
	IncompatibleFeature FeatureType = iota
	// CompatibleFeature compatible feature.
	CompatibleFeature
	// AutoclearFeature Autoclear feature.
	AutoclearFeature
)

// IncompatDirtyBitNr represents a incompatible dirty bit number.
type IncompatDirtyBitNr int

// IncompatCorruptBitNr represents a incompatible corrupt bit number.
type IncompatCorruptBitNr int

const (
	// IncompatDirty incompatible corrupt bit number.
	IncompatDirty IncompatDirtyBitNr = 1
	// IncompatCorrupt incompatible corrupt bit number.
	IncompatCorrupt IncompatCorruptBitNr = 1

	// IncompatMask mask of incompatible feature.
	IncompatMask = int(IncompatDirty) | int(IncompatCorrupt)
)

// CompatLazyRefcountsBitNr represents a compatible dirty bit number.
type CompatLazyRefcountsBitNr int

const (
	// CompatLazyRefcounts refcounts of lazy compatible.
	CompatLazyRefcounts CompatLazyRefcountsBitNr = 1

	// CompatFeatMask mask of compatible feature.
	CompatFeatMask = int(CompatLazyRefcounts)
)

// DiscardType represents a type of discard.
type DiscardType int

const (
	// DiscardNever discard never.
	DiscardNever DiscardType = iota
	// DiscardAlways discard always.
	DiscardAlways
	// DiscardRequest discard request.
	DiscardRequest
	// DiscardSnapshot discard snapshot.
	DiscardSnapshot
	// DiscardOther discard other.
	DiscardOther
	// DiscardMax discard max.
	DiscardMax
)
