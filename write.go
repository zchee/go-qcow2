// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import "github.com/pkg/errors"

// WriteMagic writes the QCow2 magic string.
func (blk *BlockBackend) WriteMagic() {
	// 0 - 3: QCow2 magic string
	_, err := blk.img.WriteAt(blk.header.Magic, 0)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 magic string")
	}
}

// WriteVersion writes the version of QCow2 image format.
func (blk *BlockBackend) WriteVersion() {
	// 4 -7: version
	_, err := blk.img.WriteAt(ToBigEndian32(int32(blk.header.Version)), 4)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 format version")
	}
}

// WriteBackingFile writes the backing file information.
func (blk *BlockBackend) WriteBackingFile() {
	//  8 - 15: backing_file_offset
	// 16 - 19: backing_file_size
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.BackingFileOffset), 8)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 backing file offset")
	}

	_, err = blk.img.WriteAt(ToBigEndian32(blk.header.BackingFileSize), 16)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 backing file size")
	}
}

// WriteClusterBits writes the number of cluster bits.
func (blk *BlockBackend) WriteClusterBits() {
	// 20 - 23: cluster_bits
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.ClusterBits), 20)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 cluster bits")
	}
}

// WriteSize writes the virtual size of QCow2 image.
func (blk *BlockBackend) WriteSize() {
	// 24 - 31: size
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.Size), 24)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 image size")
	}
}

// WriteCryptMethod writes the encrypt method.
func (blk *BlockBackend) WriteCryptMethod() {
	// 32 - 35: crypt_method
	_, err := blk.img.WriteAt(ToBigEndian32(int32(blk.header.CryptMethod)), 32)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 crypt method")
	}
}

// WriteL1Size writes the number of entries in the active L1 table.
func (blk *BlockBackend) WriteL1Size() {
	// 36 - 39: l1_size
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.L1Size), 36)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 L1 table size")
	}
}

// WriteL1TableOffset writes the number of entries in the active L1 table.
func (blk *BlockBackend) WriteL1TableOffset() {
	// 40 - 47: l1_table_offset
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.L1TableOffset), 40)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 L1 table offset")
	}
}

// WriteRefcountTableOffset writes the offset into the image file at which the refcount table starts.
func (blk *BlockBackend) WriteRefcountTableOffset() {
	// 48 - 55: refcount_table_offset
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.RefcountTableOffset), 48)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 refcount table offset")
	}
}

// WriteRefcountTableClusters writes the number of refcount table occupies clusters.
func (blk *BlockBackend) WriteRefcountTableClusters() {
	// 56 - 59: refcount_table_clusters
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.RefcountTableClusters), 56)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 refcount table clusters")
	}
}

// WriteNbSnapshots writes the number of snapshots contained in the image.
func (blk *BlockBackend) WriteNbSnapshots() {
	// 60 - 63: nb_snapshots
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.NbSnapshots), 60)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 number of snapshots")
	}
}

// WriteSnapshotsOffset writes the offset into the image file at which the snapshot table starts.
func (blk *BlockBackend) WriteSnapshotsOffset() {
	// 64 - 71: snapshots_offset
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.SnapshotsOffset), 64)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 snapshots offset")
	}
}

// WriteIncompatibleFeatures writes the incompatible features bitmask.
func (blk *BlockBackend) WriteIncompatibleFeatures() {
	// 72 - 79: incompatible_features
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.IncompatibleFeatures), 72)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 incompatible features")
	}
}

// WriteCompatibleFeatures writes the compatible features bitmask.
func (blk *BlockBackend) WriteCompatibleFeatures() {
	// 80 - 87: compatible_features
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.CompatibleFeatures), 80)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 compatible features")
	}
}

// WriteAutoClearFeatures writes the auto-clear features bitmask.
func (blk *BlockBackend) WriteAutoClearFeatures() {
	// 88 - 95: autoclear_fuatures
	_, err := blk.img.WriteAt(ToBigEndian64(blk.header.AutoclearFeatures), 88)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write qcow2 auto-clear features")
	}
}

// WriteRefcountOrder writes the width of a reference count block entry(width in bits).
func (blk *BlockBackend) WriteRefcountOrder() {
	// 96 - 99: refcount_order
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.RefcountOrder), 96)
	if err != nil && blk.Error != nil {
		blk.Error = errors.Wrap(err, "Could not write refcount order")
	}
}

// WriteHeaderLength writes the length of the header structure in bytes.
func (blk *BlockBackend) WriteHeaderLength() {
	// V3: 100 - 103: header_length
	_, err := blk.img.WriteAt(ToBigEndian32(blk.header.HeaderLength), 100)
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
