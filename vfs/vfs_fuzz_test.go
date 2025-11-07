package vfs

import (
	"errors"
	"io"
	stdfs "io/fs"
	"math/rand/v2"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/matryer/is"
)

type fsOp uint8

const (
	openFileFS fsOp = iota
	createFS
	readFileFS
	readDirFS
	writeFileFS
	mkDirFS
	mkdirAllFS
	statFS
	lstatFS
	renameFS
	removeFS
	removeAllFS
	truncateFS
	walkDirFS
	chdirFS

	read
	readAt
	readDir
	write
	writeAt
	stat
	seek
	truncate
	closeFile

	endOp
)

const (
	initialDirname  = "dir"
	initialFilename = "file.txt"

	maxOps       = 50
	totalMaxOps  = 75
	maxRandBytes = 8096
)

// FuzzVFSRace preforms random operations on a virtual filesystem in
// two different goroutines and checks that operations don't fail
// unexpectedly. It also tests that operations preform as expected
// when multiple operations are preformed concurrently. It should be
// run with the race detector enabled for best results.
func FuzzVFSRace(f *testing.F) {
	f.Fuzz(func(t *testing.T, ops1 []uint8, ops2 []uint8, seed1, seed2 uint64) {
		// Skip if the number of operations is too large, it can
		// exceed the 1 second timeout.
		if len(ops1) == 0 || len(ops2) == 0 {
			t.Skip()
		} else if len(ops1) > maxOps || len(ops2) > maxOps || len(ops1)+len(ops2) > totalMaxOps {
			t.Skip()
		}

		is := is.New(t)

		fs := NewFS().(*virtualFS)
		rand.Int()

		is.NoErr(fs.Mkdir(initialDirname, 0o777))
		initialFile, err := fs.OpenFile(initialFilename, os.O_RDWR|os.O_CREATE, 0o777)
		is.NoErr(err)

		performOps := func(ops []uint8, done chan struct{}, num uint64) {
			defer func() { done <- struct{}{} }()

			f := initialFile.(*vfsFile)
			dirname := initialDirname
			filename := initialFilename
			r := rand.New(rand.NewPCG(seed1+num, seed2+num))

			checkErr := checkErrFunc(t, num)

			for _, o := range ops {
				op := fsOp(o) % endOp

				switch op {
				// filesystem operations
				case openFileFS:
					t.Logf("%d: openFileFS(%s)", num, filename)

					openedFile, err := fs.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o777)
					checkErr("openFileFS", err)
					f = openedFile.(*vfsFile)

					t.Logf("%d: openFileFS finished", num)
				case createFS:
					filename := strconv.FormatUint(r.Uint64(), 10)
					t.Logf("%d: createFS(%s)", num, filename)

					createdFile, err := fs.Create(filename)
					checkErr("createFS", err)
					f = createdFile.(*vfsFile)

					t.Logf("%d: createFS finished", num)
				case readFileFS:
					t.Logf("%d: readFileFS(%s)", num, filename)

					_, err := fs.ReadFile(filename)
					checkErr("readFileFS", err, io.EOF, stdfs.ErrNotExist)

					t.Logf("%d: readFileFS finished", num)
				case readDirFS:
					t.Logf("%d: readDirFS(%s)", num, dirname)

					_, err := fs.ReadDir(dirname)
					checkErr("readDirFS", err, stdfs.ErrNotExist)

					t.Logf("%d: readDirFS finished", num)
				case writeFileFS:
					data := randBytes(t, r)
					t.Logf("%d: writeFileFS(%s [%d bytes])", num, filename, len(data))

					err := fs.WriteFile(filename, data, 0o666)
					checkErr("writeFileFS", err)

					t.Logf("%d: writeFileFS finished", num)
				case mkDirFS:
					dirname = strconv.FormatUint(r.Uint64(), 10)
					t.Logf("%d: mkDirFS(%s)", num, dirname)

					err := fs.Mkdir(dirname, 0o777)
					checkErr("mkDirFS", err)

					t.Logf("%d: mkDirFS finished", num)
				case mkdirAllFS:
					dirname = path.Join(dirname, strconv.FormatUint(r.Uint64(), 10))
					t.Logf("%d: mkdirAllFS(%s)", num, dirname)

					err := fs.MkdirAll(dirname, 0o777)
					checkErr("mkdirAllFS", err)

					t.Logf("%d: mkdirAllFS finished", num)
				case statFS:
					t.Logf("%d: statFS(%s)", num, filename)

					_, err := fs.Stat(filename)
					checkErr("statFS", err, stdfs.ErrNotExist)

					t.Logf("%d: statFS finished", num)
				case lstatFS:
					t.Logf("%d: lstatFS(%s)", num, filename)

					_, err := fs.Lstat(filename)
					checkErr("lstatFS", err, stdfs.ErrNotExist)

					t.Logf("%d: lstatFS finished", num)
				case renameFS:
					newFilename := strconv.FormatUint(r.Uint64(), 10)
					t.Logf("%d: renameFS(%s, %s)", num, filename, newFilename)

					err := fs.Rename(filename, newFilename)
					if checkErr("renameFS", err, stdfs.ErrNotExist) {
						filename = newFilename
					}

					t.Logf("%d: renameFS finished", num)
				case removeFS:
					t.Logf("%d: removeFS(%s)", num, filename)

					err := fs.Remove(filename)
					if checkErr("removeFS", err, stdfs.ErrNotExist) {
						f = initialFile.(*vfsFile)
						filename = initialFilename
					}

					t.Logf("%d: removeFS finished", num)
				case removeAllFS:
					t.Logf("%d: removeAllFS(%s)", num, dirname)

					err := fs.RemoveAll(dirname)
					if checkErr("removeAllFS", err, stdfs.ErrNotExist) {
						dirname = initialDirname
					}

					t.Logf("%d: removeAllFS finished", num)
				case truncateFS:
					newSize := r.Int64N(maxRandBytes)
					t.Logf("%d: truncateFS(%s, %d)", num, filename, newSize)

					err := fs.Truncate(filename, newSize)
					checkErr("truncateFS", err, stdfs.ErrNotExist)

					t.Logf("%d: truncateFS finished", num)
				case chdirFS:
					t.Logf("%d: chdirFS(%s)", num, dirname)

					err := fs.Chdir(dirname)
					checkErr("chdirFS", err, stdfs.ErrNotExist)

					t.Logf("%d: chdirFS finished", num)

					// file operations
				case read:
					buf := make([]byte, r.IntN(maxRandBytes))
					t.Logf("%d: (%s) read([%d bytes])", num, f.name, len(buf))

					_, err := f.Read(buf)
					checkErr("read", err, io.EOF, stdfs.ErrClosed)

					t.Logf("%d: (%s) read finished", num, f.name)
				case readAt:
					fi, err := f.Stat()
					if checkErr("stat", err, stdfs.ErrClosed) {
						off := r.Int64N(fi.Size() + 1)
						buf := make([]byte, r.IntN(maxRandBytes))
						t.Logf("%d: (%s) readAt([%d bytes], %d)", num, f.name, len(buf), off)

						_, err = f.ReadAt(buf, off)
						checkErr("readAt", err, io.EOF, stdfs.ErrClosed)
					}

					t.Logf("%d: (%s) readAt finished", num, f.name)
				case write:
					data := randBytes(t, r)
					t.Logf("%d: (%s) write([%d bytes])", num, f.name, len(data))

					_, err := f.Write(data)
					checkErr("write", err, stdfs.ErrClosed)

					t.Logf("%d: (%s) write finished", num, f.name)
				case writeAt:
					fi, err := f.Stat()
					if checkErr("stat", err, stdfs.ErrClosed) {
						off := r.Int64N(fi.Size() + 1)
						data := randBytes(t, r)
						t.Logf("%d: (%s) writeAt([%d bytes], %d)", num, f.name, len(data), off)

						_, err = f.WriteAt(data, off)
						checkErr("writeAt", err, stdfs.ErrClosed)
					}

					t.Logf("%d: (%s) writeAt finished", num, f.name)
				case stat:
					t.Logf("%d: (%s) stat()", num, f.name)
					_, err := f.Stat()
					checkErr("stat", err, stdfs.ErrClosed)

					t.Logf("%d: (%s) stat finished", num, f.name)
				case seek:
					fi, err := f.Stat()
					if checkErr("stat", err, stdfs.ErrClosed) {
						off := r.Int64N(fi.Size() + 1)
						whence := r.IntN(3)
						t.Logf("%d: (%s) seek(%d, %d)", num, f.name, off, whence)

						_, err = f.Seek(off, whence)
						checkErr("seek", err, stdfs.ErrClosed)
					}

					t.Logf("%d: (%s) seek finished", num, f.name)
				case truncate:
					newSize := r.Int64N(maxRandBytes)
					t.Logf("%d: (%s) truncate(%d)", num, f.name, newSize)

					err := f.Truncate(newSize)
					checkErr("truncate", err, stdfs.ErrClosed)

					t.Logf("%d: (%s) truncate finished", num, f.name)
				case closeFile:
					t.Logf("%d: (%s) close()", num, f.name)

					err := f.Close()
					checkErr("close", err, stdfs.ErrClosed)
					f = initialFile.(*vfsFile)

					t.Logf("%d: (%s) close finished", num, f.name)
				}
			}
		}

		done1 := make(chan struct{})
		done2 := make(chan struct{})
		go performOps(ops1, done1, 1)
		go performOps(ops2, done2, 2)

		for range 2 {
			select {
			case <-done1:
			case <-done2:
			case <-time.After(time.Second):
				t.Fatal("timed out")
			}
		}
	})
}

// checkErrFunc returns a function that checks if an error is nil or
// of an expected type. Otherwise it will log the error and fail the
// test. Because we are testing random operations concurrently, some
// errors are expected to occur.
func checkErrFunc(t *testing.T, num uint64) func(string, error, ...error) bool {
	t.Helper()

	return func(op string, err error, allowedErrs ...error) bool {
		t.Helper()

		if err == nil {
			return true
		}
		if len(allowedErrs) == 0 {
			t.Fatalf("%d: %s: expected no errors, got: %v", num, op, err)
			return false
		}

		for _, allowedErr := range allowedErrs {
			if errors.Is(err, allowedErr) {
				return false
			}
		}

		t.Fatalf("%d: %s: expected one of these errors: %v got: %v", num, op, allowedErrs, err)
		return false
	}
}

func randBytes(t *testing.T, r *rand.Rand) []byte {
	t.Helper()

	data := make([]byte, r.IntN(maxRandBytes))
	//nolint:intrange
	for i := range len(data) {
		data[i] = byte(r.UintN(256))
	}

	return data
}
