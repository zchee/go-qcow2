// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"os"
	"unsafe"

	"github.com/pkg/errors"
)

// QemuOpts configs the create new qcow2 format image.
type Opts struct {
	// BLOCK_OPT
	Size int64

	//  Encryption option is if this option is set to "on", the image is encrypted with 128-bit AES-CBC.
	Encryption bool

	Compat6 string

	//  BackingFile file name of a base image (see create subcommand).
	BackingFile string

	//  BackingFormat image format of the base image.
	BackingFormat string

	//  ClusterSize option is changes the qcow2 cluster size (must be between 512 and 2M).
	//  Smaller cluster sizes can improve the image file size whereas larger cluster sizes generally provide better performance.
	ClusterSize int

	TableSize int

	//  Preallocation mode of pre-allocation metadata (allowed values: "off", "metadata", "falloc", "full").
	//  An image with preallocated metadata is initially larger but can improve performance when the image needs to grow.
	//  "falloc" and "full" preallocations are like the same options of "raw" format, but sets up metadata also.
	Preallocation PreallocMode

	SubFormat string

	//  Compat QCow2 image format compatible. "compat=0.10": uses the traditional image format that can be read by any QEMU since 0.10.
	//  "compat=1.1":  enables image format extensions that only QEMU 1.1 and newer understand (this is the default).
	Compat string

	//  LazyRefcounts option is if this option is set to "on", reference count updates are postponed with the goal of avoiding metadata I/O and improving performance.
	//  This is particularly interesting with cache=writethrough which doesn't batch metadata updates.
	//  The tradeoff is that after a host crash, the reference count tables must be rebuilt,
	//  i.e. on the next open an (automatic) "qemu-img check -r all" is required, which may take some time.
	//  This option can only be enabled if "compat=1.1" is specified.
	LazyRefcounts bool // LazyRefcounts Avoiding metadata I/O and improving performance with the postponed updates reference count.

	AdapterType string

	Redundancy bool

	//  NoCow option is if this option is set to "on", it will turn off COW of the file. It's only valid on btrfs, no effect on other file systems.
	//  Btrfs has low performance when hosting a VM image file, even more when the guest on the VM also using btrfs as file system.
	//  Turning off COW is a way to mitigate this bad performance. Generally there are two ways to turn off COW on btrfs: a)
	//  Disable it by mounting with nodatacow, then all newly created files will be NOCOW. b)
	//  For an empty file, add the NOCOW file attribute. That's what this option does.
	//  Note: this option is only valid to new or empty files.
	//  If there is an existing file which is COW and has data blocks already, it couldn't be changed to NOCOW by setting "nocow=on".
	//  One can issue "lsattr filename" to check if the NOCOW flag is set or not (Capital 'C' is NOCOW flag).
	NoCow bool

	ObjectSize int

	RefcountBits int
}

// Create creates the new QCow2 virtual disk image by the qemu style.
func Create(filename string, opts *Opts) (Header, error) {
	var (
		flags   int
		version = Version3
		hdrbuf  bytes.Buffer
	)

	size := roundUp(int(opts.Size), BDRV_SECTOR_SIZE)
	backingFile := opts.BackingFile
	backingFormat := opts.BackingFormat
	log.Printf("backingFormat: %+v\n", backingFormat)
	if opts.Encryption {
		flags |= BLOCK_FLAG_ENCRYPT
	}
	clusterSize := opts.ClusterSize
	if clusterSize == 0 {
		clusterSize = DEFAULT_CLUSTER_SIZE
	}
	// TODO(zchee): error handle
	prealloc := opts.Preallocation

	compat := opts.Compat
	switch compat {
	case "0.10":
		version = Version2
	case "1.1":
		// nothing to do
	case "":
		compat = "1.1"
	default:
		err := errors.Errorf("Invalid compatibility level: '%s'", compat)
		return Header{}, err
	}

	if opts.LazyRefcounts {
		flags |= BLOCK_FLAG_LAZY_REFCOUNTS
	}

	if backingFile != "" && prealloc != PREALLOC_MODE_OFF {
		err := errors.New("Backing file and preallocation cannot be used at the same time")
		return Header{}, err
	}

	if version < 3 && (flags&BLOCK_FLAG_LAZY_REFCOUNTS) == 0 {
		err := errors.New("Lazy refcounts only supported with compatibility level 1.1 and above (use compat=1.1 or greater)")
		return Header{}, err
	}

	refcountBits := opts.RefcountBits
	if refcountBits == 0 {
		refcountBits = 16 // defaults
	}
	if refcountBits > 64 {
		err := errors.New("Refcount width must be a power of two and may not exceed 64 bits")
		return Header{}, err
	}

	refcountOrder := ctz32(uint32(refcountBits))

	clusterBits := ctz32(uint32(clusterSize))
	if clusterBits < MIN_CLUSTER_BITS || clusterBits > MAX_CLUSTER_BITS || (1<<uint(clusterBits)) != opts.ClusterSize {
		err := errors.Errorf("Cluster size must be a power of two between %d and %dk", 1<<MIN_CLUSTER_BITS, 1<<(MAX_CLUSTER_BITS-10))
		return Header{}, err
	}

	if prealloc == PREALLOC_MODE_FULL || prealloc == PREALLOC_MODE_FALLOC {
		// Note: The following calculation does not need to be exact; if it is a
		// bit off, either some bytes will be "leaked" (which is fine) or we
		// will need to increase the file size by some bytes (which is fine,
		// too, as long as the bulk is allocated here). Therefore, using
		// floating point arithmetic is fine.
		var metaSize int64
		alignedTotalZize := alignOffset(size, clusterSize)
		rces := int64(1<<uint(refcountOrder)) / 8.

		refblockBits := clusterBits - (refcountOrder - 3)
		refblockSize := 1 << uint(refblockBits)

		metaSize += int64(clusterSize)

		nl2e := alignedTotalZize / int64(clusterSize)
		nl2e = alignOffset(nl2e, clusterSize/UINT64_SIZE)
		metaSize += nl2e * UINT64_SIZE

		nl1e := nl2e * UINT64_SIZE / int64(clusterSize)
		nl1e = alignOffset(nl1e, clusterSize/UINT64_SIZE)
		metaSize += nl1e * UINT64_SIZE

		// total size of refcount blocks
		//
		// note: every host cluster is reference-counted, including metadata
		// (even refcount blocks are recursively included).
		// Let:
		//   a = total_size (this is the guest disk size)
		//   m = meta size not including refcount blocks and refcount tables
		//   c = cluster size
		//   y1 = number of refcount blocks entries
		//   y2 = meta size including everything
		//   rces = refcount entry size in bytes
		// then,
		//   y1 = (y2 + a)/c
		//   y2 = y1 * rces + y1 * rces * sizeof(u64) / c + m
		// we can get y1:
		//   y1 = (a + m) / (c - rces - rces * sizeof(u64) / c)
		nrefblocke := (alignedTotalZize + metaSize + int64(clusterSize)) / (int64(clusterSize) - rces - rces*UINT64_SIZE) / int64(clusterSize)
		metaSize += divRoundUp(int(nrefblocke), refblockSize) * int64(clusterSize)

		// total size of refcount tables
		nreftablee := nrefblocke / int64(refblockSize)
		nreftablee = alignOffset(nreftablee, clusterSize/UINT64_SIZE)
		metaSize += nreftablee * UINT64_SIZE

		size = alignedTotalZize + metaSize
	}

	blkOption := new(BlockOption)
	image, err := CreateFile(filename, blkOption)
	if err != nil {
		return Header{}, err
	}
	defer image.Close()
	defer os.Remove(image.Name())

	blk := new(BlockBackend)
	if err := blk.Open(image.Name(), "", nil, os.O_RDWR); err != nil {
		return Header{}, err
	}

	blk.allowBeyondEOF = true

	blk.Header = Header{
		Magic:                 BEUint32(MAGIC),
		Version:               version,
		BackingFileOffset:     uint64(0),
		BackingFileSize:       uint32(0),
		ClusterBits:           uint32(clusterBits),
		Size:                  uint64(size), // TODO(zchee): Sets to when initializing of the header? qemu is after initialization.
		CryptMethod:           CRYPT_NONE,
		L1Size:                uint32(0),
		L1TableOffset:         uint64(0),
		RefcountTableOffset:   uint64(clusterSize),
		RefcountTableClusters: uint32(1),
		NbSnapshots:           uint32(0),
		SnapshotsOffset:       uint64(0),
		IncompatibleFeatures:  uint64(0),
		CompatibleFeatures:    uint64(0),
		AutoclearFeatures:     uint64(0),
		RefcountOrder:         uint32(refcountOrder), // NOTE: qemu now supported only refcount_order = 4
		HeaderLength:          uint32(unsafe.Sizeof(Header{})),
	}

	if opts.Encryption {
		blk.Header.CryptMethod = CRYPT_AES
	}

	if opts.LazyRefcounts {
		blk.Header.CompatibleFeatures |= uint64(COMPAT_LAZY_REFCOUNTS)
	}

	binary.Write(&hdrbuf, binary.BigEndian, blk.Header)

	if blk.Header.Version >= Version3 {
		binary.Write(&hdrbuf, binary.BigEndian, uint32(HeaderExtensionFeatureNameTable))

		features := []Feature{
			Feature{
				Type: uint8(FEAT_TYPE_INCOMPATIBLE),
				Bit:  uint8(INCOMPAT_DIRTY_BITNR),
				Name: "dirty bit",
			},
			Feature{
				Type: uint8(FEAT_TYPE_INCOMPATIBLE),
				Bit:  uint8(INCOMPAT_CORRUPT_BITNR),
				Name: "corrupt bit",
			},
			Feature{
				Type: uint8(FEAT_TYPE_COMPATIBLE),
				Bit:  uint8(COMPAT_LAZY_REFCOUNTS_BITNR),
				Name: "lazy refcounts",
			},
		}
		binary.Write(&hdrbuf, binary.BigEndian, uint32(unsafe.Sizeof(Feature{}))*uint32(len(features)))

		for _, f := range features {
			binary.Write(&hdrbuf, binary.BigEndian, f.Type)
			binary.Write(&hdrbuf, binary.BigEndian, f.Bit)
			binary.Write(&hdrbuf, binary.BigEndian, []byte(f.Name))
			zeroFill(&hdrbuf, int64(46-len([]byte(f.Name))))
		}
	}

	blk.buf = hdrbuf

	PrintByte(hdrbuf.Bytes())
	return blk.Header, nil
}

func roundUp(n, d int) int64 {
	return int64((n + d - 1) & -d)
}

func divRoundUp(n, d int) int64 {
	return int64((n + d - 1) / d)
}

func alignOffset(offset int64, n int) int64 {
	offset = int64((int(offset) + n - 1) & (n - 1))
	return int64(offset)
}

// zeroFill writes n zero bytes into w.
func zeroFill(w io.Writer, n int64) error {
	const blocksize = 32 << 10
	zeros := make([]byte, blocksize)
	var k int
	var err error
	for n > 0 {
		if n > blocksize {
			k, err = w.Write(zeros)
		} else {
			k, err = w.Write(zeros[:n])
		}
		if err != nil {
			return err
		}
		n -= int64(k)
	}
	return nil
}

// CreateFile creates the new file based by block driver backend.
func CreateFile(filename string, options *BlockOption) (*os.File, error) {
	image, err := ioutil.TempFile(os.TempDir(), "qcow2")
	// disk, err := os.Create(cfg.FileName)
	if err != nil {
		return nil, err
	}

	return image, nil
}
