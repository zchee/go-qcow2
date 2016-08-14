// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qcow2

import "os"

type Driver string

const (
	DriverQCow2 Driver = "qcow2"
)

type BlockOption struct {
	Driver Driver
}

func NewBlockOption(driver Driver) *BlockOption {
	return &BlockOption{
		Driver: driver,
	}
}

// BlockBackend represents a backend of the QCow2 image format block driver.
type BlockBackend struct {
	img            *os.File
	header         *QCowHeader
	allowBeyondEOF bool

	Error error
}

// NewBlock return the new bulk structure.
func NewBlockBackend(header *QCowHeader, disk *os.File) *BlockBackend {
	return &BlockBackend{
		header: header,
		img:    disk,
	}
}

// Open open the QCow2 block-backend image file.
func (blk *BlockBackend) Open(filename, reference string, options *BlockOption, flag int) error {
	image, err := os.OpenFile(filename, flag, os.FileMode(0))
	if err != nil {
		return err
	}

	blk.img = image

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
