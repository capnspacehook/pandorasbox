// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestTempFile(t *testing.T) {
	fs := setup(t) //fs, _ := memfs.NewFS()

	dir := fs.TempDir()
	if _, err := fs.Stat(dir); os.IsNotExist(err) {
		fs.Mkdir(dir, 0700)
	}
	dir, err := TempDir(fs, "", "TestTempFile_BadDir")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveAll(dir)

	nonexistentDir := filepath.Join(dir, "_not_exists_")
	f, err := TempFile(fs, nonexistentDir, "foo")
	if f != nil || err == nil {
		t.Errorf("TempFile(%q, `foo`) = %v, %v", nonexistentDir, f, err)
	}

	dir = fs.TempDir()
	f, err = TempFile(fs, dir, "ioutil_test")
	if f == nil || err != nil {
		t.Errorf("TempFile(dir, `ioutil_test`) = %v, %v", f, err)
	}
	if f != nil {
		f.Close()
		fs.Remove(f.Name())
		re := regexp.MustCompile("^" + regexp.QuoteMeta(filepath.Join(dir, "ioutil_test")) + "[0-9]+$")
		if !re.MatchString(f.Name()) {
			t.Errorf("TempFile(`"+dir+"`, `ioutil_test`) created bad name %s", f.Name())
		}
	}
}

func TestTempDir(t *testing.T) {
	fs := setup(t)
	name, err := TempDir(fs, "/_not_exists_", "foo")
	if name != "" || err == nil {
		t.Errorf("TempDir(`/_not_exists_`, `foo`) = %v, %v", name, err)
	}

	dir := fs.TempDir()
	name, err = TempDir(fs, dir, "ioutil_test")
	if name == "" || err != nil {
		t.Errorf("TempDir(dir, `ioutil_test`) = %v, %v", name, err)
	}
	if name != "" {
		fs.Remove(name)
		re := regexp.MustCompile("^" + regexp.QuoteMeta(filepath.Join(dir, "ioutil_test")) + "[0-9]+$")
		if !re.MatchString(name) {
			t.Errorf("TempDir(`"+dir+"`, `ioutil_test`) created bad name %s", name)
		}
	}
}

// test that we return a nice error message if the dir argument to TempDir doesn't
// exist (or that it's empty and os.TempDir doesn't exist)
func TestTempDir_BadDir(t *testing.T) {
	fs := setup(t)
	dir, err := TempDir(fs, "", "TestTempDir_BadDir")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveAll(dir)

	badDir := filepath.Join(dir, "not-exist")
	_, err = TempDir(fs, badDir, "foo")
	if pe, ok := err.(*os.PathError); !ok || !os.IsNotExist(err) || pe.Path != badDir {
		t.Errorf("TempDir error = %#v; want PathError for path %q satisifying os.IsNotExist", err, badDir)
	}
}
