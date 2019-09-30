// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/vfs"
)

func checkSize(t *testing.T, fs absfs.FileSystem, path string, size int64) {
	dir, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", path, size, err)
	}
	if dir.Size() != size {
		t.Errorf("Stat %q: size %d want %d", path, dir.Size(), size)
	}
}

func setup(t *testing.T) absfs.FileSystem {
	fs, _ := vfs.NewFS()
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if path == "." {
			return nil
		}
		if info.IsDir() {
			return fs.Mkdir(path, info.Mode())
		}
		fin, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fin.Close()

		fout, err := fs.Create(path)
		if err != nil {
			return err
		}
		defer fout.Close()

		io.Copy(fout, fin)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return fs
}

func TestReadFile(t *testing.T) {
	fs := setup(t)

	filename := "rumpelstilzchen"
	contents, err := ReadFile(fs, filename)
	if err == nil {
		t.Fatalf("ReadFile %s: error expected, none found", filename)
	}

	filename = "ioutil_test.go"
	contents, err = ReadFile(fs, filename)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", filename, err)
	}

	checkSize(t, fs, filename, int64(len(contents)))
}

func TestWriteFile(t *testing.T) {
	fs := setup(t)

	f, err := TempFile(fs, "", "ioutil-test")
	if err != nil {
		t.Fatal(err)
	}
	filename := f.Name()
	data := "Programming today is a race between software engineers striving to " +
		"build bigger and better idiot-proof programs, and the Universe trying " +
		"to produce bigger and better idiots. So far, the Universe is winning."

	if err := WriteFile(fs, filename, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", filename, err)
	}

	contents, err := ReadFile(fs, filename)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", filename, err)
	}

	if string(contents) != data {
		t.Fatalf("contents = %q\nexpected = %q", string(contents), data)
	}

	// cleanup
	f.Close()
	fs.Remove(filename) // ignore error
}

func TestReadDir(t *testing.T) {
	fs := setup(t)

	dirname := "rumpelstilzchen"
	_, err := ReadDir(fs, dirname)
	if err == nil {
		t.Fatalf("ReadDir %s: error expected, none found", dirname)
	}

	dirname = "."
	list, err := ReadDir(fs, dirname)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dirname, err)
	}

	foundFile := false
	foundSubDir := false
	for _, dir := range list {
		switch {
		case !dir.IsDir() && dir.Name() == "ioutil_test.go":
			foundFile = true
		case dir.IsDir() && dir.Name() == "testdata":
			foundSubDir = true
		}
	}
	if !foundFile {
		t.Fatalf("ReadDir %s: io_test.go file not found", dirname)
	}
	if !foundSubDir {
		t.Fatalf("ReadDir %s: ioutil directory not found", dirname)
	}
}
