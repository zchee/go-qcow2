// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

// image: /Users/zchee/Library/Containers/com.docker.docker/Data/com.docker.driver.amd64-linux/Docker.qcow2
// file format: qcow2
// virtual size: 64G (68719476736 bytes)
// disk size: 4.0G
// cluster_size: 65536
// Format specific information:
//     compat: 1.1
//     lazy refcounts: true
//     refcount bits: 16
//     corrupt: false
// qcow2.Header{Version:3, BackingFileOffset:0, BackingFileSize:0, ClusterBits:16, Size:68719476736, CryptMethod:0, L1Size:128, L1TableOffset:131072, RefcountTableOffset:65536, RefcountTableClusters:1, NbSnapshots:0, SnapshotsOffset:0, IncompatibleFeatures:0, CompatibleFeatures:0, AutoclearFeatures:0, RefcountOrder:4, HeaderLength:104, ExtHeaders:[]qcow2.ExtHeader(nil)}
// IncompatibleFeatures: 0
// CompatibleFeatures: 0
// []byte{
//  81, 70, 73, 251, // magic
//  0, 0, 0, 3,      // version
//  0, 0, 0, 0, 0, 0, 0, 0, // BacknigFileOffset
//  0, 0, 0, 0, // BackingFileSize
//  0, 0, 0, 16, // ClusterBits
//  0, 0, 0, 16, 0, 0, 0, 0, // Size( 4 / 160000 = 640000 MB)
//  0, 0, 0, 0, // CryptMethod
//  0, 0, 0, 128, // L1Size
//  0, 0, 0, 0, 0, 2, 0, 0, // L1TableOffset
//  0, 0, 0, 0, 0, 1, 0, 0, // RefcountTableOffset
//  0, 0, 0, 1, // RefcountTableClusters
//  0, 0, 0, 0, // NbSnapshots
//  0, 0, 0, 0, 0, 0, 0, 0, // SnapshotsOffset
//  0, 0, 0, 0, 0, 0, 0, 1, // IncompatibleFeatures
//  0, 0, 0, 0, 0, 0, 0, 1, // CompatibleFeatures
//  0, 0, 0, 0, 0, 0, 0, 0, // AutoclearFeatures
//  0, 0, 0, 4, // RefcountOrder
//  0, 0, 0, 104, // HeaderLength
//  0, 0, 0, 0, 0 // ExtensionHeader
//  }

// qcow2.Header{
// 	Version:3,
//	BackingFileOffset:0,
//	BackingFileSize:0,
//	ClusterBits:16,
//	Size:68719476736,
//	CryptMethod:0,
//	L1Size:128,
//	L1TableOffset:131072,
//	RefcountTableOffset:65536,
//	RefcountTableClusters:1,
//	NbSnapshots:0,
//	SnapshotsOffset:0,
//	IncompatibleFeatures:0,
//	CompatibleFeatures:0,
//	AutoclearFeatures:0,
//	RefcountOrder:4,
//	HeaderLength:104,
//	ExtHeaders:[]qcow2.ExtHeader(nil)
// }
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

// New return the new Qcow.
func New(config *Config) *QCow2 {
	return &QCow2{
		Header: &QCowHeader{
			Version: 3,
		},
	}
}
