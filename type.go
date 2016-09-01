// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"log"
	"os"
)

// ---------------------------------------------------------------------------
// block/qcow2.c

// Extension represents a optional header extension.
type Extension struct {
	Magic HeaderExtensionType // [:4] Header extension type
	Len   uint32              // [4:8] Length of the header extension data
}

// HeaderExtensionType represents a indicators the the entries in the optional header area
type HeaderExtensionType uint32

const (
	// HeaderExtensionEndOfArea End of the header extension area.
	HeaderExtensionEndOfArea HeaderExtensionType = 0x00000000

	// HeaderExtensionBackingFileFormat Backing file format name.
	HeaderExtensionBackingFileFormat HeaderExtensionType = 0xE2792ACA

	// HeaderExtensionFeatureNameTable Feature name table.
	HeaderExtensionFeatureNameTable HeaderExtensionType = 0x6803f857

	// HeaderExtensionBitmapsExtension Bitmaps extension.
	// TODO(zchee): qemu does not implements?
	HeaderExtensionBitmapsExtension HeaderExtensionType = 0x23852875

	// Safely ignored other unknown header extension
)

// ---------------------------------------------------------------------------
// block/qcow2.h

// MAGIC qemu QCow(2) magic ("QFI\xfb").
// Original source code:
//  #define QCOW_MAGIC (('Q' << 24) | ('F' << 16) | ('I' << 8) | 0xfb)
var MAGIC = []byte{0x51, 0x46, 0x49, 0xFB}

// CryptMethod represents a whether encrypted qcow2 image.
// 0 for no enccyption
// 1 for AES encryption
type CryptMethod uint32

const (
	// CRYPT_NONE no encryption.
	CRYPT_NONE CryptMethod = iota
	// CRYPT_AES AES encryption.
	CRYPT_AES

	MAX_CRYPT_CLUSTERS = 32
	MAX_SNAPSHOTS      = 65536
)

// String implementations of fmt.Stringer.
func (cm CryptMethod) String() string {
	if cm == 1 {
		return "AES"
	}
	return "none"
}

// MAX_REFTABLE_SIZE 8 MB refcount table is enough for 2 PB images at 64k cluster size
// (128 GB for 512 byte clusters, 2 EB for 2 MB clusters)
const MAX_REFTABLE_SIZE = 0x800000

// MAX_L1_SIZE 32 MB L1 table is enough for 2 PB images at 64k cluster size
// (128 GB for 512 byte clusters, 2 EB for 2 MB clusters)
const MAX_L1_SIZE = 0x2000000

/* Allow for an average of 1k per snapshot table entry, should be plenty of
 * space for snapshot names and IDs */
const MAX_SNAPSHOTS_SIZE = 1024 * MAX_SNAPSHOTS

const (
	// indicate that the refcount of the referenced cluster is exactly one.
	OFLAG_COPIED = 1 << 63
	// indicate that the cluster is compressed (they never have the copied flag)
	OFLAG_COMPRESSED = 1 << 62
	// The cluster reads as all zeros
	OFLAG_ZERO = 1 << 0
)

const (
	// MIN_CLUSTER_BITS minimum of cluster bits size.
	MIN_CLUSTER_BITS = 9
	// MAX_CLUSTER_BITS maximum of cluster bits size.
	MAX_CLUSTER_BITS = 21
)

// MIN_L2_CACHE_SIZE must be at least 2 to cover COW.
const MIN_L2_CACHE_SIZE = 2 // clusters

// MIN_REFCOUNT_CACHE_SIZE must be at least 4 to cover all cases of refcount table growth.
const MIN_REFCOUNT_CACHE_SIZE = 4 // clusters

/* Whichever is more */
const DEFAULT_L2_CACHE_CLUSTERS = 8        // clusters
const DEFAULT_L2_CACHE_BYTE_SIZE = 1048576 // bytes

// DEFAULT_L2_REFCOUNT_SIZE_RATIO the refblock cache needs only a fourth of the L2 cache size to cover as many
// clusters.
const DEFAULT_L2_REFCOUNT_SIZE_RATIO = 4

const DEFAULT_CLUSTER_SIZE = 65536

// Header represents a header of qcow2 image format.
type Header struct {
	Magic                 uint32      //     [0:3] magic: QCOW magic string ("QFI\xfb")
	Version               Version     //     [4:7] Version number
	BackingFileOffset     uint64      //    [8:15] Offset into the image file at which the backing file name is stored.
	BackingFileSize       uint32      //   [16:19] Length of the backing file name in bytes.
	ClusterBits           uint32      //   [20:23] Number of bits that are used for addressing an offset whithin a cluster.
	Size                  uint64      //   [24:31] Virtual disk size in bytes
	CryptMethod           CryptMethod //   [32:35] Crypt method
	L1Size                uint32      //   [36:39] Number of entries in the active L1 table
	L1TableOffset         uint64      //   [40:47] Offset into the image file at which the active L1 table starts
	RefcountTableOffset   uint64      //   [48:55] Offset into the image file at which the refcount table starts
	RefcountTableClusters uint32      //   [56:59] Number of clusters that the refcount table occupies
	NbSnapshots           uint32      //   [60:63] Number of snapshots contained in the image
	SnapshotsOffset       uint64      //   [64:71] Offset into the image file at which the snapshot table starts
	IncompatibleFeatures  uint64      //   [72:79] for version >= 3: Bitmask of incomptible feature
	CompatibleFeatures    uint64      //   [80:87] for version >= 3: Bitmask of compatible feature
	AutoclearFeatures     uint64      //   [88:95] for version >= 3: Bitmask of auto-clear feature
	RefcountOrder         uint32      //   [96:99] for version >= 3: Describes the width of a reference count block entry
	HeaderLength          uint32      // [100:103] for version >= 3: Length of the header structure in bytes
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
	Magic uint32
	Len   uint32
	// Next QLIST_ENTRY(Qcow2UnknownHeaderExtension)
	Data []int8
}

// FeatureType represents a type of feature.
type FeatureType uint8

const (
	// FEAT_TYPE_INCOMPATIBLE incompatible feature.
	FEAT_TYPE_INCOMPATIBLE FeatureType = iota
	// FEAT_TYPE_COMPATIBLE compatible feature.
	FEAT_TYPE_COMPATIBLE
	// FEAT_TYPE_AUTOCLEAR Autoclear feature.
	FEAT_TYPE_AUTOCLEAR
)

const (
	// INCOMPAT_DIRTY_BITNR represents a incompatible dirty bit number.
	INCOMPAT_DIRTY_BITNR = iota

	// INCOMPAT_CORRUPT_BITNR represents a incompatible corrupt bit number.
	INCOMPAT_CORRUPT_BITNR

	// INCOMPAT_DIRTY incompatible corrupt bit number.
	INCOMPAT_DIRTY = 1 << INCOMPAT_DIRTY_BITNR
	// INCOMPAT_CORRUPT incompatible corrupt bit number.
	INCOMPAT_CORRUPT = 1 << INCOMPAT_CORRUPT_BITNR

	// INCOMPAT_MASK mask of incompatible feature.
	INCOMPAT_MASK = INCOMPAT_DIRTY | INCOMPAT_CORRUPT
)

const (
	// COMPAT_LAZY_REFCOUNTS_BITNR represents a compatible dirty bit number.
	COMPAT_LAZY_REFCOUNTS_BITNR = iota
	// COMPAT_LAZY_REFCOUNTS refcounts of lazy compatible.
	COMPAT_LAZY_REFCOUNTS = 1 << COMPAT_LAZY_REFCOUNTS_BITNR

	// COMPAT_FEAT_MASK mask of compatible feature.
	COMPAT_FEAT_MASK = COMPAT_LAZY_REFCOUNTS
)

// DiscardType represents a type of discard.
type DiscardType int

const (
	// DISCARD_NEVER discard never.
	DISCARD_NEVER DiscardType = iota
	// DISCARD_ALWAYS discard always.
	DISCARD_ALWAYS
	// DISCARD_REQUEST discard request.
	DISCARD_REQUEST
	// DISCARD_SNAPSHOT discard snapshot.
	DISCARD_SNAPSHOT
	// DISCARD_OTHER discard other.
	DISCARD_OTHER
	// DISCARD_MAX discard max.
	DISCARD_MAX
)

type Feature struct {
	Type uint8  // uint8_t
	Bit  uint8  // uint8_t
	Name string // char    name[46];
	byt  []byte
}

type DiscardRegion struct {
	Bs     *BlockDriverState
	Offset uint64 // uint64_t
	byt    uint64 // uint64_t
	// next QTAILQ_ENTRY(Qcow2DiscardRegion)
}

// GetRefcountFunc typedef uint64_t Qcow2GetRefcountFunc(const void *refcount_array, uint64_t index);
func GetRefcountFunc(refcountArray map[uint64]uintptr, index uint64) uint64 {
	ro0 := (refcountArray[index/8] >> (index % 8)) & 0x1
	ro1 := (refcountArray)[index/4] >> (2 * (index % 4))
	ro2 := (refcountArray)[index/2] >> (4 * (index % 2))
	ro3 := (refcountArray)[index]
	ro4 := BEUvarint16(uint16(refcountArray[index]))
	ro5 := BEUvarint32(uint32(refcountArray[index]))
	ro6 := BEUvarint64(uint64(refcountArray[index]))
	log.Println(ro0, ro1, ro2, ro3, ro4, ro5, ro6)

	// TODO(zchee): WIP
	return 0
}

// SetRefcountFunc typedef void Qcow2SetRefcountFunc(void *refcount_array, uint64_t index, uint64_t value);
func SetRefcountFunc(refcountArray map[uint64]uintptr, index uint64) {
	// TODO(zchee): WIP
	return
}

type BDRVState struct {
	ClusterBits       int    // int
	ClusterSize       int    // int
	ClusterSectors    int    // int
	L2Bits            int    // int
	L2Size            int    // int
	L1Size            int    // int
	L1VmStateIndex    int    // int
	RefcountBlockBits int    // int
	RefcountBlockSize int    // int
	Csize_shift       int    // int
	Csize_mask        int    // int
	ClusterOffsetMask uint64 // uint64_t
	L1TableOffset     uint64 // uint64_t
	L1Table           uint64 // uint64_t

	L2TableCache       *Cache // *Qcow2Cache
	RefcountBlockCache *Cache // *Qcow2Cache
	// cache_clean_timer    *QEMUTimer
	CacheCleanInterval uintptr // unsigned

	ClusterCache       uint8  // uint8_t
	ClusterData        uint8  // uint8_t
	ClusterCacheOffset uint64 // uint64_t
	// cluster_allocs QLIST_HEAD(QCowClusterAlloc, QCowL2Meta)

	RefcountTable       map[uint64]int64 // uint64_t
	RefcountTableOffset uint64           // uint64_t
	RefcountTableSize   uint64           // uint64_t
	FreeClusterIndex    uint64           // uint64_t
	FreeByteOffset      uint64           // uint64_t

	// lock CoMutex // CoMutex

	// cipher              *QCryptoCipher // current cipher, NULL if no key yet
	CryptMethodHeader uint32  // uint32_t
	SnapshotsOffset   uint64  // uint64_t
	SnapshotsSize     int     // int
	NbSnapshots       uintptr // unsigend int
	// snapshots           *QCowSnapshot

	Flags            int     // int
	Version          Version // int
	UseLazyRefcounts bool    // bool
	RefcountOrder    int     // int
	RefcountBits     int     // int
	RefcountMax      uint64  // uint64_t

	GetRefcount func(refcountArray interface{}, index uint64) uint64        // *Qcow2GetRefcountFunc
	SetRefcount func(refcountArray interface{}, index uint64, value uint64) // *Qcow2SetRefcountFunc

	DiscardPassthrough bool // bool discard_passthrough[QCOW2_DISCARD_MAX]

	OverlapCheck       int  // int: bitmask of Qcow2MetadataOverlap values
	SignaledCorruption bool // bool

	IncompatibleFeatures uint64 // uint64_t
	CompatibleFeatures   uint64 // uint64_t
	AutoclearFeatures    uint64 // uint64_t

	UnknownheaderFieldsSize int    // size_t
	UnknownHeaderFields     []byte // void*
	// unknown_header_ext QLIST_HEAD(, Qcow2UnknownHeaderExtension)
	// discards QTAILQ_HEAD (, Qcow2DiscardRegion)
	CacheDiscards bool // bool

	// Backing file path and format as stored in the image (this is not the
	// effective path/format, which may be the result of a runtime option
	// override)
	ImageBackingFile   string // char *
	ImageBackingFormat []byte // char *
}

// ---------------------------------------------------------------------------
// include/block/block_int.h

const BLOCK_FLAG_ENCRYPT = 1

const BLOCK_FLAG_LAZY_REFCOUNTS = 8

const BLOCK_PROBE_BUF_SIZE = 512

// Driver represents a name of driver.
type BlockDriver string

const (
	// DriverQCow2 qcow2 driver.
	DriverQCow2 BlockDriver = "qcow2"
)

type BlockLimits struct {
	// RequestAlignment alignment requirement, in bytes, for offset/length of I/O
	// requests. Must be a power of 2 less than INT_MAX; defaults to
	// 1 for drivers with modern byte interfaces, and to 512
	// otherwise.
	RequestAlignment uint32 // uint32_t

	// MaxPdiscard maximum number of bytes that can be discarded at once (since it
	// is signed, it must be < 2G, if set). Must be multiple of
	// pdiscard_alignment, but need not be power of 2. May be 0 if no
	// inherent 32-bit limit
	MaxPdiscard int32 // int32_t

	// PdiscardAlignment optimal alignment for discard requests in bytes. A power of 2
	// is best but not mandatory.  Must be a multiple of
	// bl.request_alignment, and must be less than max_pdiscard if
	// that is set. May be 0 if bl.request_alignment is good enough
	PdiscardAlignment uint32 // uint32_t

	// MaxPwriteZeroes maximum number of bytes that can zeroized at once (since it is
	// signed, it must be < 2G, if set). Must be multiple of
	// pwrite_zeroes_alignment. May be 0 if no inherent 32-bit limit
	MaxPwriteZeroes int32 // int32_t

	// PwriteZeroesAlignment optimal alignment for write zeroes requests in bytes. A power
	// of 2 is best but not mandatory.  Must be a multiple of
	// bl.request_alignment, and must be less than max_pwrite_zeroes
	// if that is set. May be 0 if bl.request_alignment is good
	// enough
	PwriteZeroesAlignment uint32 // uint32_t

	// OptTransfer optimal transfer length in bytes.  A power of 2 is best but not
	// mandatory.  Must be a multiple of bl.request_alignment, or 0 if
	// no preferred size
	OptTransfer uint32 // uint32_t

	// MaxTransfer maximal transfer length in bytes.  Need not be power of 2, but
	// must be multiple of opt_transfer and bl.request_alignment, or 0
	// for no 32-bit limit.  For now, anything larger than INT_MAX is
	// clamped down.
	MaxTransfer uint32 // uint32_t

	// MinMemAlignment memory alignment, in bytes so that no bounce buffer is needed
	MinMemAlignment uint32 // size_t

	// OptMemAlignment memory alignment, in bytes, for bounce buffer
	OptMemAlignment uint32 // size_t

	// MaxIov maximum number of iovec elements
	MaxIov int // int
}

// BlockDriverState represents a state of block driver.
//
// Note: the function bdrv_append() copies and swaps contents of
// BlockDriverStates, so if you add new fields to this struct, please
// inspect bdrv_append() to determine if the new fields need to be
// copied as well.
type BlockDriverState struct {
	TotalSectors int64 // int64_t: if we are reading a disk image, give its size in sectors
	OpenFlags    int   // int:     flags used to open the file, re-used for re-open
	ReadOnly     bool  // bool:    if true, the media is read only
	Encrypted    bool  // bool:    if true, the media is encrypted
	ValidKey     bool  // bool:    if true, a valid encryption key has been set
	SG           bool  // bool:    if true, the device is a /dev/sg*
	Probed       bool  // bool:    if true, format was probed rather than specified

	CopyOnRead int // int: if nonzero, copy read backing sectors into image. note this is a reference count.

	// flush_queue // CoQueue: Serializing flush queue // TODO
	// active_flush_req // *BdrvTrackedRequest: Flush request in flight // TODO
	WriteGen   uint // unsigned int: Current data generation
	FlushedGen uint // unsigned int: Flushed write generation

	Drv    *BlockDriver // BlockDriver *: NULL means no media
	Opaque *BDRVState   // void *

	// AioContext *AioContext // event loop used for fd handlers, timers, etc // TODO

	// long-running tasks intended to always use the same AioContext as this
	// BDS may register themselves in this list to be notified of changes
	// regarding this BDS's context
	// AioNotifiers QLIST_HEAD(, BdrvAioNotifier) // TODO
	WalkingAioNotifiers bool // bool: to make removal during iteration safe

	Filename      string // char: filename[PATH_MAX]
	BackingFile   string // char: if non zero, the image is a diff of this file image
	BackingFormat string // char: if non-zero and backing_file exists

	// FullOpenOptions *QDict // *QDict * TODO
	ExactFilename string // char: exact_filename[PATH_MAX]

	// Backing *BdrvChild // TODO
	File os.File // BdrvChild

	// BeforeWriteNotifiers Callback before write request is processed
	// BeforeWriteNotifiers NotifierWithReturnList // TODO

	// SerialisingInFlight number of in-flight serialising requests
	SerialisingInFlight uint // unsigned int

	// Offset after the highest byte written to
	WrHighestOffset uint64 // uint64_t

	// I/O Limits
	BL BlockLimits

	// unsigned int: Flags honored during pwrite (so far: BDRV_REQ_FUA)
	SupportedWriteFlags uint
	// unsigned int: Flags honored during pwrite_zeroes (so far: BDRV_REQ_FUA, *BDRV_REQ_MAY_UNMAP)
	SupportedZeroFlags uint

	// NodeName the following member gives a name to every node on the bs graph.
	NodeName string // char node_name[32]
	// NodeList element of the list of named nodes building the graph
	// NodeList QTAILQ_ENTRY(BlockDriverState) // TODO
	// BsList element of the list of all BlockDriverStates (all_bdrv_states)
	// BsList QTAILQ_ENTRY(BlockDriverState) // TODO
	// MonitorList element of the list of monitor-owned BDS
	// MonitorList QTAILQ_ENTRY(BlockDriverState) // TODO
	// DirtyBitmaps QLIST_HEAD(, BdrvDirtyBitmap) // TODO
	Refcnt int // int

	// TrackedRequests QLIST_HEAD(, BdrvTrackedRequest) // TODO

	// operation blockers
	// OpBlockers [BLOCK_OP_TYPE_MAX]QLIST_HEAD(, BdrvOpBlocker) // operation blockers TODO

	// long-running background operation
	// Job *BlockJob // TODO

	// The node that this node inherited default options from (and a reopen on
	// which can affect this node by changing these defaults). This is always a
	// parent node of this node.
	// InheritsFrom *BlockDriverState // BlockDriverState *: TODO
	// Children QLIST_HEAD(, BdrvChild) // TODO
	// Parents QLIST_HEAD(, BdrvChild) // TODO

	// Options         *QDict                      // TODO
	// ExplicitOptions *QDict                      // TODO
	// DetectZeroes    BlockdevDetectZeroesOptions // TODO

	// The error object in use for blocking operations on backing_hd
	BackingBlocker error

	// threshold limit for writes, in bytes. "High water mark"
	WriteThresholdOffset uint64
	// WriteThresholdNotifier NotifierWithReturn // TODO

	// Counters for nested bdrv_io_plug and bdrv_io_unplugged_begin
	IOPlugged      uintptr // unsigned: TODO
	IOPlugDisabled uintptr // unsigned: TODO

	QuiesceCounter int // int
}

type BdrvChild struct {
	BlockDriverState *BlockDriverState
	Name             string
	// Role   *BdrvChildRole
	Opaque *BDRVState
	// next QLIST_ENTRY(BdrvChild)
	// next_parent QLIST_ENTRY(BdrvChild)
}

// ---------------------------------------------------------------------------
// qapi-types.h

// PreallocMode represents a mode of Pre-allocation feature.
type PreallocMode int

const (
	// PREALLOC_MODE_OFF turn off preallocation.
	PREALLOC_MODE_OFF PreallocMode = iota
	// PREALLOC_MODE_METADATA preallocation of metadata only mode.
	PREALLOC_MODE_METADATA
	// PREALLOC_MODE_FALLOC preallocation of falloc only mode.
	PREALLOC_MODE_FALLOC
	// PREALLOC_MODE_FULL full preallocation mode.
	PREALLOC_MODE_FULL
	// PREALLOC_MODE__MAX preallocation maximum preallocation mode.
	PREALLOC_MODE__MAX
)

// ---------------------------------------------------------------------------
// unknown

const (
	// UINT16_SIZE results of sizeof(uint16_t) in C.
	UINT16_SIZE = 2
	// UINT32_SIZE results of sizeof(uint32_t) in C.
	UINT32_SIZE = 4
	// UINT64_SIZE results of sizeof(uint64_t) in C.
	UINT64_SIZE = 8
)

// Image represents a qemu QCow2 image format.
type Image struct {
	BlockBackend
}

// Version represents a version number of qcow image format.
// The valid values are 2 and 3.
type Version uint32

const (
	// Version2 qcow2 image format version2.
	Version2 Version = 2
	// Version3 qcow2 image format version3.
	Version3 Version = 3
)

const (
	// Version2HeaderSize is the image header at the beginning of the file.
	Version2HeaderSize = 72
	// Version3HeaderSize is directly following the v2 header, up to 104.
	Version3HeaderSize = 104
)

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

const BDRV_SECTOR_BITS = 9

var (
	BDRV_SECTOR_SIZE = 1 << BDRV_SECTOR_BITS // (1ULL << BDRV_SECTOR_BITS)
	BDRV_SECTOR_MASK = BDRV_SECTOR_SIZE - 1  // ~(BDRV_SECTOR_SIZE - 1)
)

const ReftOffsetMask = 1844674407370955110
