// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

// Opts options of the create qcow2 image format.
type Opts struct {
	// Filename filename of create image.
	Filename string
	// Fmt format of create image.
	Fmt DriverFmt
	// BaseFliename base filename of create image.
	BaseFilename string
	// BaseFmt base format of create image.
	BaseFmt string

	// BLOCK_OPT
	// Size size of create image virtual size.
	Size int64

	//  Encryption option is if this option is set to "on", the image is encrypted with 128-bit AES-CBC.
	Encryption bool

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
func Create(opts *Opts) (*Image, error) {
	if opts.Filename == "" {
		err := errors.New("Expecting image file name")
		return nil, err
	}

	// TODO(zchee): implements file size eror handling
	// sval = qemu_strtosz_suffix(argv[optind++], &end,
	// QEMU_STRTOSZ_DEFSUFFIX_B);
	// if (sval < 0 || *end) {
	// 	if (sval == -ERANGE) {
	// 		error_report("Image size must be less than 8 EiB!");
	// 	} else {
	// 		error_report("Invalid image size specified! You may use k, M, "
	// 		"G, T, P or E suffixes for ");
	// 		error_report("kilobytes, megabytes, gigabytes, terabytes, "
	// 		"petabytes and exabytes.");
	// 	}
	// 	goto fail;
	// }

	img := new(Image)
	blk, err := create(opts.Filename, opts)
	if err != nil {
		return nil, err
	}
	img.BlockBackend = blk
	return img, nil
}

func create(filename string, opts *Opts) (*BlockBackend, error) {

	// ------------------------------------------------------------------------
	// static int qcow2_create(const char *filename, QemuOpts *opts,
	//                         Error **errp)

	var (
		flags int
		// default is version3
		version = Version3
	)

	size := roundUp(int(opts.Size), BDRV_SECTOR_SIZE)
	backingFile := opts.BackingFile
	// backingFormat := opts.BackingFormat

	if opts.Encryption {
		flags |= BLOCK_FLAG_ENCRYPT
	}

	clusterSize := int64(opts.ClusterSize)
	if clusterSize == 0 {
		clusterSize = DEFAULT_CLUSTER_SIZE
	}

	// TODO(zchee): error handle
	prealloc := opts.Preallocation

	compat := opts.Compat
	switch compat {
	case "":
		compat = "1.1" // automatically set to latest compatible version
	case "0.10":
		version = Version2
	case "1.1":
		// nothing to do
	default:
		err := errors.Errorf("Invalid compatibility level: '%s'", compat)
		return nil, err
	}

	if opts.LazyRefcounts {
		flags |= BLOCK_FLAG_LAZY_REFCOUNTS
	}

	if backingFile != "" && prealloc != PREALLOC_MODE_OFF {
		err := errors.New("Backing file and preallocation cannot be used at the same time")
		return nil, err
	}

	if version < 3 && (flags&BLOCK_FLAG_LAZY_REFCOUNTS) == 0 {
		err := errors.New("Lazy refcounts only supported with compatibility level 1.1 and above (use compat=1.1 or greater)")
		return nil, err
	}

	refcountBits := opts.RefcountBits
	if refcountBits == 0 {
		refcountBits = 16 // defaults
	}
	if refcountBits > 64 {
		err := errors.New("Refcount width must be a power of two and may not exceed 64 bits")
		return nil, err
	}

	refcountOrder := ctz32(uint32(refcountBits))

	// ------------------------------------------------------------------------
	// static int qcow2_create2(const char *filename, int64_t total_size,
	//                          const char *backing_file,
	//                          const char *backing_format,
	//                          int flags, size_t cluster_size,
	//                          PreallocMode prealloc,
	//                          QemuOpts *opts, int version,
	//                          int refcount_order,
	//                          Error **errp)

	// Calculate cluster_bits
	clusterBits := ctz32(uint32(clusterSize))
	if clusterBits < MIN_CLUSTER_BITS || clusterBits > MAX_CLUSTER_BITS || (1<<uint(clusterBits)) != opts.ClusterSize {
		err := errors.Errorf("Cluster size must be a power of two between %d and %dk", 1<<MIN_CLUSTER_BITS, 1<<(MAX_CLUSTER_BITS-10))
		return nil, err
	}

	// Open the image file and write a minimal qcow2 header.
	//
	// We keep things simple and start with a zero-sized image. We also
	// do without refcount blocks or a L1 table for now. We'll fix the
	// inconsistency later.
	//
	// We do need a refcount table because growing the refcount table means
	// allocating two new refcount blocks - the seconds of which would be at
	// 2 GB for 64k clusters, and we don't want to have a 2 GB initial file
	// size for any qcow2 image.

	if prealloc == PREALLOC_MODE_FULL || prealloc == PREALLOC_MODE_FALLOC {
		// Note: The following calculation does not need to be exact; if it is a
		// bit off, either some bytes will be "leaked" (which is fine) or we
		// will need to increase the file size by some bytes (which is fine,
		// too, as long as the bulk is allocated here). Therefore, using
		// floating point arithmetic is fine.
		var metaSize int64
		alignedTotalZize := alignOffset(size, int(clusterSize))
		rces := int64(1<<uint(refcountOrder)) / 8.

		refblockBits := clusterBits - (refcountOrder - 3)
		refblockSize := 1 << uint(refblockBits)

		metaSize += int64(clusterSize)

		nl2e := alignedTotalZize / clusterSize
		nl2e = alignOffset(nl2e, int(clusterSize/int64(UINT64_SIZE)))
		metaSize += nl2e * UINT64_SIZE

		nl1e := nl2e * UINT64_SIZE / clusterSize
		nl1e = alignOffset(nl1e, int(clusterSize/int64(UINT64_SIZE)))
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
		nreftablee = alignOffset(nreftablee, int(clusterSize/int64(UINT64_SIZE)))
		metaSize += nreftablee * UINT64_SIZE

		size = alignedTotalZize + metaSize
	}

	blkOption := new(BlockOption)
	diskImage, err := CreateFile(filename, blkOption)
	if err != nil {
		return nil, err
	}
	defer diskImage.Close()

	blk := new(BlockBackend)
	blk.BlockDriverState = &BlockDriverState{
		file: &BdrvChild{
			Name: diskImage.Name(),
		},
	}

	// TODO(zchee): should use func Open(bs BlockDriverState, options *QDict, flag int) error
	// if err := Open(blk.bs(), nil, flags); err != nil {
	if err := blk.Open(diskImage.Name(), "", nil, os.O_RDWR|os.O_CREATE); err != nil {
		return nil, err
	}

	blk.BlockDriverState.Opaque = new(BDRVState)

	blk.allowBeyondEOF = true

	blk.Header = Header{
		Magic:                 BEUint32(MAGIC),
		Version:               version,
		BackingFileOffset:     uint64(0),
		BackingFileSize:       uint32(0),
		ClusterBits:           uint32(clusterBits),
		Size:                  uint64(size), // TODO(zchee): Sets to when initializing of the header? qemu is after initialization.
		CryptMethod:           CRYPT_NONE,
		L1Size:                uint32(128),    // TODO(zchee): hardcoded
		L1TableOffset:         uint64(458752), // TODO(zchee): hardcoded
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

	// Write a header data to blk.buf
	binary.Write(&blk.buf, binary.BigEndian, blk.Header)

	if blk.Header.Version >= Version3 {
		binary.Write(&blk.buf, binary.BigEndian, uint32(HeaderExtensionFeatureNameTable))

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

		binary.Write(&blk.buf, binary.BigEndian, uint32(unsafe.Sizeof(Feature{}))*uint32(len(features)))

		for _, f := range features {
			binary.Write(&blk.buf, binary.BigEndian, f.Type)
			binary.Write(&blk.buf, binary.BigEndian, f.Bit)
			binary.Write(&blk.buf, binary.BigEndian, []byte(f.Name))
			zeroFill(&blk.buf, int64(46-uint8(len([]byte(f.Name)))))
		}
	}

	// Write a header data to image file
	Write(blk.bs(), 0, blk.buf.Bytes(), blk.buf.Len())

	// Write a refcount table with one refcount block
	refcountTable := make([][]byte, 2*clusterSize)
	refcountTable[0] = BEUvarint64(uint64(2 * clusterSize))

	// TODO(zchee): int(2*clusterSize))?
	Write(blk.bs(), clusterSize, bytes.Join(refcountTable, []byte{}), int(clusterSize))

	blk.BlockDriverState.Drv = new(BlockDriver)
	blk.BlockDriverState.Drv.bdrvGetlength = getlength
	// bs.Drv.bdrvTruncate = bdrvTruncate

	blk.BlockDriverState.Opaque = &BDRVState{
		ClusterSize:   int(clusterSize),
		ClusterBits:   clusterBits,
		RefcountOrder: refcountOrder,
	}

	if _, err := AllocClusters(blk.bs(), uint64(3*clusterSize)); err != nil {
		if err != syscall.Errno(0) {
			err = errors.Wrap(err, "Huh, first cluster in empty image is already in use?")
			return nil, err
		}

		err = errors.Wrap(err, "Could not allocate clusters for qcow2 header and refcount table")
		return nil, err
	}

	// Create a full header (including things like feature table)
	// ret = qcow2_update_header(blk_bs(blk));
	// if (ret < 0) {
	// 	error_setg_errno(errp, -ret, "Could not update qcow2 header");
	// 	goto out;
	// }

	// TODO(zchee): carried from bdrv_open_common, should move to the Open function
	blk.bs().Opaque.L2Bits = blk.bs().Opaque.ClusterBits - 3
	blk.bs().Opaque.L2Size = 1 << uint(blk.bs().Opaque.L2Bits)
	blk.bs().Opaque.RefcountTableOffset = blk.Header.RefcountTableOffset
	// blk.bs().Opaque.RefcountTableSize = blk.Header.RefcountTableClusters << uint(blk.bs().Opaque.ClusterBits-3)

	// Okay, now that we have a valid image, let's give it the right size
	if err := Truncate(blk.bs(), size); err != nil {
		err = errors.Wrap(err, "Could not resize image")
		return nil, err
	}

	// Want a backing file? There you go
	if backingFile != "" {
		// TODO(zchee): implements bdrv_change_backing_file
	}

	// And if we're supposed to preallocate metadata, do that now
	if prealloc != PREALLOC_MODE_OFF {
		// TODO(zchee): implements preallocate()
	}

	return blk, nil
}

// refreshTotalSectors sets the current 'total_sectors' value
func refreshTotalSectors(bs *BlockDriverState, hint int64) error {
	drv := bs.Drv

	// Do not attempt drv->bdrv_getlength() on scsi-generic devices
	if bs.SG {
		return nil
	}

	// query actual device if possible, otherwise just trust the hint
	if drv.bdrvGetlength != nil {
		length, err := drv.bdrvGetlength(bs)
		if err != nil {
			return err
		}
		if length < 0 {
			return nil
		}
		hint = divRoundUp(int(length), BDRV_SECTOR_SIZE)
	}

	bs.TotalSectors = hint
	return nil
}

func Truncate(bs *BlockDriverState, offset int64) error {
	s := bs.Opaque

	if offset&511 != 0 {
		err := errors.Wrap(syscall.EINVAL, "The new size must be a multiple of 512")
		return err
	}

	// cannot proceed if image has snapshots
	if s.NbSnapshots != 0 {
		err := errors.Wrap(syscall.ENOTSUP, "Can't resize an image which has snapshots")
		return err
	}

	// shrinking is currently not supported
	if offset < bs.TotalSectors*512 {
		err := errors.Wrap(syscall.ENOTSUP, "qcow2 doesn't support shrinking images yet")
		return err
	}

	log.Printf("offset: %+v\n", offset)
	newL1Size := sizeToL1(s, offset)
	log.Printf("newL1Size: %+v\n", newL1Size)
	if err := growL1Table(bs, uint64(newL1Size), true); err != nil {
		return err
	}

	// write updated header.size
	// off := BEUvarint64(uint64(offset))
	// if err := bdrvPwriteSync(bs.File, unsafe.Offsetof(Header.Size), &offset, UINT64_SIZE); err != nil {
	// 	return err
	// }

	s.L1VmStateIndex = int(newL1Size)
	return nil
}

func startOfCluster(clusterSize int64, offset int64) int64 {
	return offset &^ (clusterSize - 1)
}

// Write writes the data in image file.
func Write(bs *BlockDriverState, offset int64, data []byte, length int) error {
	if bs.File == nil {
		err := errors.New("Not found BlockBackend file")
		return err
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, data)

	bs.File.Seek(offset, 0)
	off, err := bs.File.Write(buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "Could not write a data")
	}
	// stat, _ := bs.File.Stat()
	// if offset > stat.Size() {
	// 	bs.File.WriteAt(buf.Bytes(), offset)
	// } else {
	// 	bs.File.WriteAt(buf.Bytes(), stat.Size())
	// }

	if length > off {
		if err := zeroFill(bs.File, int64(length-off)); err != nil {
			return err
		}
	}

	return nil
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
			if err != nil {
				return err
			}

		} else {
			k, err = w.Write(zeros[:n])
			if err != nil {
				return err
			}

		}
		if err != nil {
			return err
		}
		n -= int64(k)
	}
	return nil
}

// CreateFile creates the new file based by block driver backend.
func CreateFile(filename string, opts *BlockOption) (*os.File, error) {
	image, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return image, nil
}
