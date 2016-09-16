// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"bytes"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

// New return the new Qcow.
func New(config *Opts) *Image {
	return &Image{}
}

// Open open the QCow2 block-backend image file.
// callgraph:
// qemu-img.c:img_create -> bdrv_img_create -> bdrv_open -> bdrv_open_inherit -> bdrv_open_common -> drv->bdrv_open -> .bdrv_open = qcow2_open
func Open(bs *BlockDriverState, options *QDict, flag int) error {
	s := bs.Opaque
	var header Header

	err := bdrvPread(bs.file, 0, &header, unsafe.Sizeof(header))
	if err != nil {
		err = errors.Wrap(err, "Could not read qcow2 header")
		return err
	}

	if !bytes.Equal(BEUvarint32(header.Magic), MAGIC) {
		err := errors.Wrap(syscall.EINVAL, "Image is not in qcow2 format")
		return err
	}
	if header.Version < Version2 || header.Version > Version3 {
		err := errors.Wrapf(syscall.ENOTSUP, "Unsupported qcow2 version %d", header.Version)
		return err
	}

	s.Version = header.Version

	// Initialise cluster size
	if header.ClusterBits < MIN_CLUSTER_BITS || header.ClusterBits > MAX_CLUSTER_BITS {
		err := errors.Wrapf(syscall.EINVAL, "Unsupported cluster size: 2^%d", header.ClusterBits)
		return err
	}

	s.ClusterBits = int(header.ClusterBits)
	s.ClusterSize = 1 << uint(s.ClusterBits)
	s.ClusterSectors = 1 << uint(s.ClusterBits-9)

	// Initialise version 3 header fields
	if header.Version == Version2 {
		header.IncompatibleFeatures = 0
		header.CompatibleFeatures = 0
		header.AutoclearFeatures = 0
		header.RefcountOrder = 4
		header.HeaderLength = 72
	} else {
		if header.HeaderLength < 104 {
			err := errors.Wrap(syscall.EINVAL, "qcow2 header too short")
			return err
		}
	}

	if header.HeaderLength > uint32(s.ClusterSize) {
		err := errors.Wrap(syscall.EINVAL, "qcow2 header exceeds cluster size")
		return err
	}

	hdrSizeof := uint32(unsafe.Sizeof(header))
	if header.HeaderLength > hdrSizeof {
		s.UnknownheaderFieldsSize = int(header.HeaderLength - hdrSizeof)
		s.UnknownHeaderFields = make([]byte, s.UnknownheaderFieldsSize)
		err := bdrvPread(bs.file, int64(hdrSizeof), &s.UnknownHeaderFields, uintptr(s.UnknownheaderFieldsSize))
		if err != nil {
			err = errors.Wrap(err, "Could not read unknown qcow2 header fields")
			return err
		}
	}

	if header.BackingFileOffset > uint64(s.ClusterSize) {
		err := errors.Wrap(syscall.EINVAL, "Invalid backing file offset")
		return err
	}

	var extEnd uint64
	if header.BackingFileOffset != 0 {
		extEnd = header.BackingFileOffset
	} else {
		extEnd = 1 << header.ClusterBits
	}
	log.Printf("extEnd: %+v\n", extEnd)

	// Handle feature bits
	s.IncompatibleFeatures = header.IncompatibleFeatures
	s.CompatibleFeatures = header.CompatibleFeatures
	s.AutoclearFeatures = header.AutoclearFeatures

	if int(s.IncompatibleFeatures) & ^INCOMPAT_MASK != 0 {
		// TODO(zchee): implements read extensions
		// featureTable := nil
		// qcow2_read_extensions(bs, header.header_length, ext_end, &feature_table, NULL);
		// report_unsupported_feature(errp, feature_table, s->incompatible_features & ~QCOW2_INCOMPAT_MASK);
		// ret = -ENOTSUP;
		// g_free(feature_table);
		// goto fail;
	}

	if s.IncompatibleFeatures&INCOMPAT_CORRUPT != 0 {
		// TODO(zchee): implements
		// Corrupt images may not be written to unless they are being repaired
		// if ((flags & BDRV_O_RDWR) && !(flags & BDRV_O_CHECK)) {
		// 	error_setg(errp, "qcow2: Image is corrupt; cannot be opened read/write");
		// 	ret = -EACCES;
		// 	goto fail;
		// }
	}

	// Check support for various header values
	if header.RefcountOrder > 6 {
		err := errors.Wrap(syscall.EINVAL, "Reference count entry width too large; may not exceed 64 bits")
		return err
	}
	s.RefcountOrder = int(header.RefcountOrder)
	s.RefcountBits = 1 << uint(s.RefcountOrder)
	s.RefcountMax = uint64(1) << uint64(s.RefcountBits-1)
	s.RefcountMax += s.RefcountMax - 1

	if header.CryptMethod > CRYPT_AES {
		err := errors.Wrapf(syscall.EINVAL, "Unsupported encryption method: %d", header.CryptMethod)
		return err
	}
	// TODO(zchee): implements
	// if (!qcrypto_cipher_supports(QCRYPTO_CIPHER_ALG_AES_128)) {
	// 	error_setg(errp, "AES cipher not available");
	// 	ret = -EINVAL;
	// 	goto fail;
	// }
	s.CryptMethodHeader = uint32(header.CryptMethod)
	if s.CryptMethodHeader != 0 {
		// TODO(zchee): implements
		// s->crypt_method_header == QCOW_CRYPT_AES) {
		// 	error_setg(errp, "Use of AES-CBC encrypted qcow2 images is no longer supported in system emulators")
		// 	error_append_hint(errp, "You can use 'qemu-img convert' to convert your image to an alternative supported format, such as unencrypted qcow2, or raw with the LUKS format instead.\n")
		// 	ret = -ENOSYS;
		// 	goto fail;
	}

	s.L2Bits = s.ClusterBits - 3
	s.L2Size = 1 << uint(s.L2Bits)
	// 2^(s->refcount_order - 3) is the refcount width in bytes
	s.RefcountBlockBits = s.ClusterBits - (s.RefcountOrder - 3)
	s.RefcountBlockSize = 1 << uint(s.RefcountBlockBits)
	bs.TotalSectors = int64(header.Size / 512)
	s.Csize_shift = (62 - (s.ClusterBits - 8))
	s.Csize_mask = (1 - (s.ClusterBits - 8)) - 1
	s.ClusterOffsetMask = (1 << uint(s.Csize_shift)) - 1

	s.RefcountTableOffset = header.RefcountTableOffset
	s.RefcountTableSize = header.RefcountTableClusters << uint(s.ClusterBits-3)

	if uint64(header.RefcountTableClusters) > maxRefcountClusters(s) {
		err := errors.Wrap(syscall.EINVAL, "Reference count table too large")
		return err
	}

	// ret = validate_table_offset(bs, header.l1_table_offset, header.l1_size, sizeof(uint64_t));

	return nil
}

// Open open the QCow2 block-backend image file.
func (blk *BlockBackend) Open(filename, reference string, options *BlockOption, flag int) error {
	file, err := os.OpenFile(filename, flag, os.FileMode(0))
	if err != nil {
		return err
	}

	blk.BlockDriverState.File = file

	return nil
}

func sizeToClusters(s *BDRVState, size uint64) uint64 {
	return (size + uint64(s.ClusterSize-1)) >> uint(s.ClusterBits)
}

func offsetIntoCluster(s *BDRVState, offset int64) uint64 {
	return uint64(offset & (int64(s.ClusterSize) - 1))
}

func maxRefcountClusters(s *BDRVState) uint64 {
	return MAX_REFTABLE_SIZE >> uint(s.ClusterBits)
}
