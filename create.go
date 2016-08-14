// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"
)

// "compat"
//  "compat=0.10": uses the traditional image format that can be read by any QEMU since 0.10.
//  "compat=1.1":  enables image format extensions that only QEMU 1.1 and newer understand (this is the default).
//
// "backing_file"
//  File name of a base image (see create subcommand)
//
// "backing_fmt"
//  Image format of the base image
//
// "encryption"
//  If this option is set to "on", the image is encrypted with 128-bit AES-CBC.
//
// "cluster_size"
//  Changes the qcow2 cluster size (must be between 512 and 2M).
//  Smaller cluster sizes can improve the image file size whereas larger cluster sizes generally provide better performance.
//
// "preallocation"
//  Preallocation mode (allowed values: "off", "metadata", "falloc", "full").
//  An image with preallocated metadata is initially larger but can improve performance when the image needs to grow.
//  "falloc" and "full" preallocations are like the same options of "raw" format, but sets up metadata also.
//
// "lazy_refcounts"
//  If this option is set to "on", reference count updates are postponed with the goal of avoiding metadata I/O and improving performance.
//  This is particularly interesting with cache=writethrough which doesn't batch metadata updates.
//  The tradeoff is that after a host crash, the reference count tables must be rebuilt,
//  i.e. on the next open an (automatic) "qemu-img check -r all" is required, which may take some time.
//  This option can only be enabled if "compat=1.1" is specified.
//
// "nocow"
//  If this option is set to "on", it will turn off COW of the file. It's only valid on btrfs, no effect on other file systems.
//  Btrfs has low performance when hosting a VM image file, even more when the guest on the VM also using btrfs as file system.
//  Turning off COW is a way to mitigate this bad performance. Generally there are two ways to turn off COW on btrfs: a)
//  Disable it by mounting with nodatacow, then all newly created files will be NOCOW. b)
//  For an empty file, add the NOCOW file attribute. That's what this option does.
//  Note: this option is only valid to new or empty files.
//  If there is an existing file which is COW and has data blocks already, it couldn't be changed to NOCOW by setting "nocow=on".
//  One can issue "lsattr filename" to check if the NOCOW flag is set or not (Capital 'C' is NOCOW flag).

// Config configs the create new qcow2 format image.
type Config struct {
	FileName      string
	TotalSize     int
	Flags         int
	Version       Version
	RefcountOrder int

	// Command line options
	// Compat QCow2 image format compatible. values are 0.10 or 1.1(defaut).
	Compat float64
	// BackingFile File name of a base image.
	BackingFile string
	// BackingFormat Image format of the base image.
	BackingFormat string
	// Encryption Use 128-bit AES-CBC image encryption.
	Encryption bool
	// ClusterSize Must be between 512 and 2M.
	ClusterSize int
	// Preallocation Metadata preallocation mode.
	Preallocation PreallocMode
	// LazyRefcounts Avoiding metadata I/O and improving performance with the postponed updates reference count.
	LazyRefcounts bool
	// NoCow whether turn off COW of the file. only valid on btrfs.
	NoCow bool
}

// Create create the new QCow2 virtual disk image.
//  Docker.qcow2[0:104]:
//  81 70 73 251 0 0 0 3 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 16 0 0 0 16 0 0 0 0 0 0 0 0 0 0 0 128 0 0 0 0 0 2 0 0 0 0 0 0 0 1 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0 0 0 4 0 0 0 104
func Create(cfg *Config) QCowHeader {
	// clusterBits := ToBigEndian32(cfg.ClusterSize)
	// if clusterBits < MinClusterBits || clusterBits > MaxClusterBits || (1<<uintptr(clusterBits)) != cfg.ClusterSize {
	// 	_ = errors.Errorf("Cluster size must be a power of two between %d and %dk", 1<<MinClusterBits, 1<<(MaxClusterBits-10))
	// 	return nil
	// }

	// TODO(zchee): Support (prealloc == PREALLOC_MODE_FULL || prealloc == PREALLOC_MODE_FALLOC)
	// if (prealloc == PREALLOC_MODE_FULL || prealloc == PREALLOC_MODE_FALLOC) {}

	blk := new(BlockBackend)

	blk.CreateFile(cfg.FileName)
	defer blk.disk.Close()
	defer os.Remove(blk.disk.Name())

	blk.allowBeyondEOF = true

	header := QCowHeader{
		Magic:                 Qcow2Magic,
		Version:               Version3,
		BackingFileOffset:     int64(0),
		BackingFileSize:       int32(0),
		ClusterBits:           int32(16),
		Size:                  int64(cfg.TotalSize), // TODO(zchee): Sets to when initializing of the header? qemu is after initialization.
		L1TableOffset:         int64(131072),        // TODO(zchee): ditto
		L1Size:                int32(128),           // TODO(zchee): ditto
		RefcountTableOffset:   int64(cfg.ClusterSize),
		RefcountTableClusters: int32(1), // TODO(zchee): contant of 1?
		IncompatibleFeatures:  int64(1),
		RefcountOrder:         int32(4),
		HeaderLength:          int32(Version3HeaderSize), // Sets 104 length by default.
	}

	// Check the optional config
	if cfg.BackingFile != "" {
		f, err := os.Open(cfg.BackingFile) // read-only open.
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		header.BackingFileOffset = int64(len(data))

		fstat, _ := f.Stat()
		header.BackingFileSize = int32(fstat.Size())
	}

	if cfg.Encryption {
		header.CryptMethod = CryptAES
	}

	if cfg.LazyRefcounts {
		header.CompatibleFeatures = int64(QCow2CompatLazyRefcounts)
	}

	blk.header = &header

	if err := blk.WriteHeader(); err != nil {
		log.Fatal(err)
	}

	data, _ := ioutil.ReadAll(blk.disk)
	log.Printf("data: %d\n%+v\n", len(data), data)

	return header
}

// BlockBackend represents a backend of the QCow2 image format block driver.
type BlockBackend struct {
	disk           *os.File
	header         *QCowHeader
	allowBeyondEOF bool

	Error error
}

// NewBlock return the new bulk structure.
func NewBlockBackend(header *QCowHeader, disk *os.File) *BlockBackend {
	return &BlockBackend{
		header: header,
		disk:   disk,
	}
}

// WriteMagic writes the QCow2 magic string.
func (blk *BlockBackend) WriteMagic() {
	// 0 - 3: QCow2 magic string
	_, err := blk.disk.WriteAt(blk.header.Magic, 0)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 magic string")
	}
}

// WriteVersion writes the version of QCow2 image format.
func (blk *BlockBackend) WriteVersion() {
	// 4 -7: version
	_, err := blk.disk.WriteAt(ToBigEndian32(int32(blk.header.Version)), 4)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 format version")
	}
}

// WriteBackingFile writes the backing file information.
func (blk *BlockBackend) WriteBackingFile() {
	//  8 - 15: backing_file_offset
	// 16 - 19: backing_file_size
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.BackingFileOffset), 8)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 backing file offset")
	}

	_, err = blk.disk.WriteAt(ToBigEndian32(blk.header.BackingFileSize), 16)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 backing file size")
	}
}

// WriteClusterBits writes the number of cluster bits.
func (blk *BlockBackend) WriteClusterBits() {
	// 20 - 23: cluster_bits
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.ClusterBits), 20)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 cluster bits")
	}
}

// WriteSize writes the virtual size of QCow2 image.
func (blk *BlockBackend) WriteSize() {
	// 24 - 31: size
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.Size), 24)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 image size")
	}
}

// WriteCryptMethod writes the encrypt method.
func (blk *BlockBackend) WriteCryptMethod() {
	// 32 - 35: crypt_method
	_, err := blk.disk.WriteAt(ToBigEndian32(int32(blk.header.CryptMethod)), 32)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 crypt method")
	}
}

// WriteL1Size writes the number of entries in the active L1 table.
func (blk *BlockBackend) WriteL1Size() {
	// 36 - 39: l1_size
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.L1Size), 36)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 L1 table size")
	}
}

// WriteL1TableOffset writes the number of entries in the active L1 table.
func (blk *BlockBackend) WriteL1TableOffset() {
	// 40 - 47: l1_table_offset
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.L1TableOffset), 40)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 L1 table offset")
	}
}

// WriteRefcountTableOffset writes the offset into the image file at which the refcount table starts.
func (blk *BlockBackend) WriteRefcountTableOffset() {
	// 48 - 55: refcount_table_offset
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.RefcountTableOffset), 48)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 refcount table offset")
	}
}

// WriteRefcountTableClusters writes the number of refcount table occupies clusters.
func (blk *BlockBackend) WriteRefcountTableClusters() {
	// 56 - 59: refcount_table_clusters
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.RefcountTableClusters), 56)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 refcount table clusters")
	}
}

// WriteNbSnapshots writes the number of snapshots contained in the image.
func (blk *BlockBackend) WriteNbSnapshots() {
	// 60 - 63: nb_snapshots
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.NbSnapshots), 60)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 number of snapshots")
	}
}

// WriteSnapshotsOffset writes the offset into the image file at which the snapshot table starts.
func (blk *BlockBackend) WriteSnapshotsOffset() {
	// 64 - 71: snapshots_offset
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.SnapshotsOffset), 64)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 snapshots offset")
	}
}

// WriteIncompatibleFeatures writes the incompatible features bitmask.
func (blk *BlockBackend) WriteIncompatibleFeatures() {
	// 72 - 79: incompatible_features
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.IncompatibleFeatures), 72)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 incompatible features")
	}
}

// WriteCompatibleFeatures writes the compatible features bitmask.
func (blk *BlockBackend) WriteCompatibleFeatures() {
	// 80 - 87: compatible_features
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.CompatibleFeatures), 80)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 compatible features")
	}
}

// WriteAutoClearFeatures writes the auto-clear features bitmask.
func (blk *BlockBackend) WriteAutoClearFeatures() {
	// 88 - 95: autoclear_fuatures
	_, err := blk.disk.WriteAt(ToBigEndian64(blk.header.AutoclearFeatures), 88)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 auto-clear features")
	}
}

// WriteRefcountOrder writes the width of a reference count block entry(width in bits).
func (blk *BlockBackend) WriteRefcountOrder() {
	// 96 - 99: refcount_order
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.RefcountOrder), 96)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write refcount order")
	}
}

// WriteHeaderLength writes the length of the header structure in bytes.
func (blk *BlockBackend) WriteHeaderLength() {
	// V3: 100 - 103: header_length
	_, err := blk.disk.WriteAt(ToBigEndian32(blk.header.HeaderLength), 100)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 header length")
	}
}

// WriteHeader writes the binary of the QCow2 image format header data.
// The return to error is always first error of internal function and additional cause message.
func (blk *BlockBackend) WriteHeader() error {
	blk.WriteMagic()
	blk.WriteVersion()
	blk.WriteBackingFile()
	blk.WriteClusterBits()
	blk.WriteSize()
	blk.WriteCryptMethod()
	blk.WriteL1Size()
	blk.WriteL1TableOffset()
	blk.WriteRefcountTableOffset()
	blk.WriteRefcountTableClusters()
	blk.WriteNbSnapshots()
	blk.WriteSnapshotsOffset()

	if blk.header.Version == Version3 {
		blk.WriteIncompatibleFeatures()
		blk.WriteCompatibleFeatures()
		blk.WriteAutoClearFeatures()
		blk.WriteRefcountOrder()
		blk.WriteHeaderLength()
	}

	// Check the first of internal functions error
	if blk.Error != nil {
		blk.Error = errors.Wrap(blk.Error, "Could not write qcow2 header")
	}

	return blk.Error
}

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

// CreateFile creates the new file based by block driver backend.
func (bdrv *BlockBackend) CreateFile(filename string) error {
	disk, err := ioutil.TempFile(os.TempDir(), "qcow2")
	// disk, err := os.Create(cfg.FileName)
	if err != nil {
		return err
	}

	bdrv.disk = disk

	// drv := "protocol?"
	// if drv == "" {
	// 	err := errors.New("unknown protocol")
	// 	return err
	// }

	// f, err := os.Open(filename)
	// if err != nil {
	// 	return errors.Wrap(err, "invalid filename")
	// }
	// AllowWriteBeyondEOF = true

	// header := &QCowHeader{}

	return nil
}
