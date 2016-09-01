// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import (
	"bytes"
	"log"
	"os"
)

// BlockOption represents a block options.
type BlockOption struct {
	Driver BlockDriver
}

// BlockBackend represents a backend of the QCow2 image format block driver.
type BlockBackend struct {
	File             *os.File
	Header           Header
	allowBeyondEOF   bool
	BlockDriverState *BlockDriverState

	buf bytes.Buffer

	Error error
}

// Open open the QCow2 block-backend image file.
func (blk *BlockBackend) Open(filename, reference string, options *BlockOption, flag int) error {
	file, err := os.OpenFile(filename, flag, os.FileMode(0))
	if err != nil {
		return err
	}

	blk.File = file

	return nil
}

func PrintByte(buf []byte) {
	log.Println(buf)
	log.Printf("    [0:3] Magic:                 %+v", buf[:4])
	log.Printf("    [4:7] Version:               %+v", buf[4:8])
	log.Printf("   [8:15] BackingFileOffset:     %+v", buf[8:16])
	log.Printf("  [16:19] BackingFileSize:       %+v", buf[16:20])
	log.Printf("  [20:23] ClusterBits:           %+v", buf[20:24])
	log.Printf("  [24:31] Size:                  %+v", buf[24:32])
	log.Printf("  [32:35] CryptMethod:           %+v", buf[32:36])
	log.Printf("  [36:39] L1Size:                %+v", buf[36:40])
	log.Printf("  [40:47] L1TableOffset:         %+v", buf[40:48])
	log.Printf("  [48:55] RefcountTableOffset:   %+v", buf[48:56])
	log.Printf("  [56:59] RefcountTableClusters: %+v", buf[56:60])
	log.Printf("  [60:63] NbSnapshots:           %+v", buf[60:64])
	log.Printf("  [64:71] SnapshotsOffset:       %+v", buf[64:72])
	log.Printf("  [72:79] IncompatibleFeatures:  %+v", buf[72:80])
	log.Printf("  [80:87] CompatibleFeatures:    %+v", buf[80:88])
	log.Printf("  [88:95] AutoclearFeatures:     %+v", buf[88:96])
	log.Printf("  [96:99] RefcountOrder:         %+v", buf[96:100])
	log.Printf("[101:105] HeaderLength:          %+v", buf[100:104])

	log.Printf("[106:109] HeaderExtensionType:   %+v", buf[104:108])
	log.Printf("[110:114] HeaderExtensionLength: %+v", buf[108:112])
	log.Printf("[115:162] HeaderExtensionData:   %+v", buf[112:158])

	log.Printf("[163:166] HeaderExtensionType:   %+v", buf[158:162])
	log.Printf("[167:171] HeaderExtensionLength: %+v", buf[162:166])
	log.Printf("[172:219] HeaderExtensionData:   %+v", buf[166:212])

	log.Printf("[220:223] HeaderExtensionType:   %+v", buf[212:216])
	log.Printf("[224:228] HeaderExtensionLength: %+v", buf[216:220])
	log.Printf("[229:276] HeaderExtensionData:   %+v", buf[220:266])

	if len(buf) > 266 {
		log.Printf("[277:]    Other:                 %+v", buf[277:len(buf)-1])
	}
}
