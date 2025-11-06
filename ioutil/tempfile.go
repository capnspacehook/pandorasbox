// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import (
	"errors"
	stdfs "io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
)

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
var (
	rand   uint32
	randmu sync.Mutex
)

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// TempFile creates a new temporary file in the directory dir of the
// absfs.FileSystem fs with a name beginning with prefix, opens the file for
// reading and writing, and returns the resulting absfs.File.
// If dir is the empty string, TempFile uses the default directory
// for temporary files for the given FileSystem (see absfs.TempDir).
// Multiple programs calling TempFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file. It is the caller's responsibility
// to remove the file when no longer needed.
func TempFile(fs absfs.FileSystem, dir, prefix string) (f absfs.File, err error) {
	if dir == "" || dir == fs.TempDir() {
		dir = fs.TempDir()
		if _, err := fs.Stat(dir); errors.Is(err, stdfs.ErrNotExist) {
			err = fs.Mkdir(dir, 0o755)
			if err != nil {
				return nil, err
			}
		}
	}

	nconflict := 0
	for range 10000 {
		name := filepath.Join(dir, prefix+nextSuffix())
		f, err = fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, stdfs.ErrExist) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

// TempDir creates a new temporary directory in the directory dir of the
// absfs.FileSystem fs with a name beginning with prefix and returns the
// path of the new directory. If dir is the empty string, TempDir uses the
// default directory for temporary files (see os.TempDir).
// Multiple programs calling TempDir simultaneously
// will not choose the same directory. It is the caller's responsibility
// to remove the directory when no longer needed.
func TempDir(fs absfs.FileSystem, dir, prefix string) (name string, err error) {
	if dir == "" || dir == fs.TempDir() {
		dir = fs.TempDir()
		if _, err := fs.Stat(dir); errors.Is(err, stdfs.ErrNotExist) {
			err = fs.Mkdir(dir, 0o700)
			if err != nil {
				return "", err
			}
		}
	}

	nconflict := 0
	for range 10000 {
		try := filepath.Join(dir, prefix+nextSuffix())
		err = fs.Mkdir(try, 0o700)
		if errors.Is(err, stdfs.ErrExist) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if errors.Is(err, stdfs.ErrNotExist) {
			if _, err := fs.Stat(dir); errors.Is(err, stdfs.ErrNotExist) {
				return "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}
