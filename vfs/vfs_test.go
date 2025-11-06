package vfs

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"
	"time"

	"github.com/matryer/is"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

const (
	dots = "1....2....3....4"
	abc  = "abcdefghijklmnop"

	filename   = "testfile"
	renameFrom = "renamefrom"
	renameTo   = "renameto"
)

func TestVFS(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	vfs := NewFS()

	if err := vfs.Mkdir("memz", 0o777); err != nil {
		t.Fatalf("error creating dir: %v", err)
	}

	f, err := vfs.Create("memz/chungus")
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	if _, err := f.Write([]byte("The quick brown fox jumped over the lazy dog.\n")); err != nil {
		t.Fatalf("error writing to file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("error closing created file: %v", err)
	}

	if err := fstest.TestFS(vfs.FS(), "memz/chungus"); err != nil {
		t.Errorf("error testing vfs: %v", err)
	}
}

func TestFileReader(t *testing.T) {
	vfs := NewFS()

	contents := make([]byte, 1000)
	if _, err := rand.Read(contents); err != nil {
		t.Fatalf("error getting random contents: %v", err)
	}

	f, err := vfs.Create("file")
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	n, err := f.Write(contents)
	if n != len(contents) {
		t.Fatalf("didn't write all of contents; got %v want %v", n, len(contents))
	}
	if err != nil {
		t.Fatalf("error writing to file: %v", err)
	}

	o, err := f.Seek(0, io.SeekStart)
	if o != 0 {
		t.Fatalf("seek didn't seek to start of file; got %v want %v", o, 0)
	}
	if err != nil {
		t.Fatalf("error seeking in file: %v", err)
	}

	if err := iotest.TestReader(f, contents); err != nil {
		t.Error(err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("error closing file: %v", err)
	}
}

func TestMkdir(t *testing.T) {
	vfs := NewFS()

	if vfs.TempDir() != "/tmp" {
		t.Fatalf("wrong TempDir output: %q != %q", vfs.TempDir(), "/tmp")
	}

	testdir := path.Join(vfs.TempDir(), "mkdir_test")
	t.Logf("Test path: %q", testdir)

	err := vfs.MkdirAll(testdir, 0o777)
	if err != nil {
		t.Fatal(err)
	}

	var list []fs.DirEntry
	path := "/"
outer:
	for _, name := range strings.Split(testdir, "/")[1:] {
		if name == "" {
			continue
		}
		f, err := vfs.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		list, err = f.ReadDir(-1)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range list {
			if n.Name() == name {
				path = filepath.Join(path, name)
				continue outer
			}
		}
		t.Errorf("path error: %q + %q:  %s", path, name, list)
	}
}

func TestOpenWrite(t *testing.T) {
	vfs := NewFS()

	f, err := vfs.Create("/test_file.txt")
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("The quick brown fox jumped over the lazy dog.\n")
	n, err := f.Write(data)
	f.Close()
	if n != len(data) {
		t.Errorf("write error: wrong byte count %d, expected %d", n, len(data))
	}
	if err != nil {
		t.Fatal(err)
	}

	f, err = vfs.Open("/test_file.txt")
	if err != nil {
		t.Fatal(err)
	}
	buff := make([]byte, 512)
	n, err = f.Read(buff)
	f.Close()
	if n != len(data) {
		t.Errorf("write error: wrong byte count %d, expected %d", n, len(data))
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected EOF, got nil error")
	}
	buff = buff[:n]
	if !bytes.Equal(data, buff) {
		t.Log(string(data))
		t.Log(string(buff))

		t.Fatal("bytes written do not compare to bytes read")
	}
}

func TestCreate(t *testing.T) {
	vfs := NewFS()
	// Create file with absolute path
	{
		f, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %v", err)
		}
		if name := f.Name(); name != "/testfile" {
			t.Errorf("Wrong name: %s", name)
		}
	}

	// Create same file again
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE, 0o666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %v", err)
		}

	}

	// Create same file again, but truncate it
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %v", err)
		}
	}

	// Create same file again with O_CREATE|O_EXCL, which is an error
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
		if err == nil {
			t.Fatalf("Expected error creating file: %v", err)
		}
	}

	// Create file with unknown parent
	{
		_, err := vfs.OpenFile("/testfile/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
		if err == nil {
			t.Errorf("Expected error creating file")
		}
	}

	// Create file with relative path (workingDir == root)
	{
		f, err := vfs.OpenFile("relFile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %v", err)
		}
		if name := f.Name(); name != "relFile" {
			t.Errorf("Wrong name: %s", name)
		}
	}
}

func TestMkdirAbsRel(t *testing.T) {
	vfs := NewFS()

	// Create dir with absolute path
	{
		err := vfs.Mkdir("/usr", 0)
		if err != nil {
			t.Fatalf("Unexpected error creating directory: %v", err)
		}
	}

	// Create dir with relative path
	{
		err := vfs.Mkdir("home", 0)
		if err != nil {
			t.Fatalf("Unexpected error creating directory: %v", err)
		}
	}

	// Create dir twice
	{
		err := vfs.Mkdir("/home", 0)
		if err == nil {
			t.Fatalf("Expecting error creating directory: %s", "/home")
		}
	}
}

func TestMkdirTree(t *testing.T) {
	vfs := NewFS()

	err := vfs.Mkdir("/home", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home: %v", err)
	}

	err = vfs.Mkdir("/home/blang", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang: %v", err)
	}

	err = vfs.Mkdir("/home/blang/goprojects", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang/goprojects: %v", err)
	}

	err = vfs.Mkdir("/home/johndoe/goprojects", 0)
	if err == nil {
		t.Errorf("Expected error creating directory with non-existing parent")
	}

	// TODO: Subdir of file
}

func TestRemove(t *testing.T) {
	vfs := NewFS()
	err := vfs.Mkdir("/tmp", 0o777)
	if err != nil {
		t.Fatalf("Mkdir error: %v", err)
	}
	f, err := vfs.OpenFile("/tmp/README.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if _, err := f.Write([]byte("test")); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	f.Close()

	// remove non existing file
	if err := vfs.Remove("/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove non existing file from an non existing directory
	if err := vfs.Remove("/nonexisting/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove created file
	err = vfs.Remove(f.Name())
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}

	if _, err = vfs.OpenFile("/tmp/README.txt", os.O_RDWR, 0o666); err == nil {
		t.Errorf("Could open removed file!")
	}

	err = vfs.Remove("/tmp")
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}
	fis, err := vfs.ReadDir("/")
	if err != nil {
		t.Errorf("Readdir error: %v", err)
	} else if len(fis) != 0 {
		t.Errorf("Found files: %s", fis)
	}
}

// Read with length 0 should not return EOF.
func TestRead0(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("open failed:", err)
	}
	if _, err := f.WriteString(abc); err != nil {
		t.Fatal("writing failed:", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal("seeking to beginning failed:", err)
	}
	defer f.Close()

	b := make([]byte, 0)
	n, err := f.Read(b)
	if n != 0 || err != nil {
		t.Errorf("Read(0) = %d, %v, want 0, nil", n, err)
	}
	b = make([]byte, 10)
	n, err = f.Read(b)
	if n <= 0 || err != nil {
		t.Errorf("Read(10) = %d, %v, want >0, nil", n, err)
	}
}

// Reading a closed file should return ErrClosed error
func TestReadClosed(t *testing.T) {
	vfs := NewFS()
	file, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("open failed:", err)
	}
	file.Close() // close immediately

	b := make([]byte, 100)
	_, err = file.Read(b)

	var pErr *os.PathError
	if !errors.As(err, &pErr) {
		t.Fatalf("Read: %T(%v), want PathError", pErr, pErr)
	}

	if !errors.Is(pErr.Err, os.ErrClosed) {
		t.Errorf("Read: %v, want PathError(ErrClosed)", pErr)
	}
}

func TestReadWrite(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Write abc
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %v", n, err)
	}

	// Seek to end of file
	if n, err := f.Seek(0, io.SeekEnd); err != nil || n != 32 {
		t.Errorf("Seek error: %d %v", n, err)
	}

	// Write dots at end of file
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %v", n, err)
	}

	p := make([]byte, len(dots)+len(abc)+len(dots))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc)+len(dots) {
		t.Errorf("Read error: %d %v", n, err)
	} else if s := string(p); s != dots+abc+dots {
		t.Errorf("Invalid read: %s", s)
	}
}

func TestOpenRO(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDONLY, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// Write first dots
	if _, err := f.Write([]byte(dots)); err == nil {
		t.Fatalf("Expected write error")
	}
	f.Close()
}

func TestOpenWO(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %v", n, err)
	}

	// Try reading
	p := make([]byte, len(dots))
	if n, err := f.Read(p); err == nil || n > 0 {
		t.Errorf("Expected invalid read: %d %v", n, err)
	}

	f.Close()
}

func TestOpenAppend(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}
	f.Close()

	// Reopen file in append mode
	f, err = vfs.OpenFile("/readme.txt", os.O_APPEND|os.O_RDWR, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// append dots
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %v", n, err)
	}

	p := make([]byte, len(dots)+len(abc))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc) {
		t.Errorf("Read error: %d %v", n, err)
	} else if s := string(p); s != dots+abc {
		t.Errorf("Invalid read: %s", s)
	}
	f.Close()
}

func TestTruncateToLength(t *testing.T) {
	params := []struct {
		size int64
		err  bool
	}{
		{-1, true},
		{0, false},
		{int64(len(dots) - 1), false},
		{int64(len(dots)), false},
		{int64(len(dots) + 1), false},
	}

	for _, param := range params {
		vfs := NewFS()
		f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0o666)
		if err != nil {
			t.Fatalf("Could not open file: %v", err)
		}
		if n, err := f.Write([]byte(dots)); err != nil {
			t.Errorf("Unexpected error: %v", err)
		} else if n != len(dots) {
			t.Errorf("Invalid write count: %d", n)
		}
		f.Close()

		newSize := param.size
		err = vfs.Truncate("/readme.txt", newSize)
		if param.err {
			if err == nil {
				t.Errorf("Error expected truncating file to length %d", newSize)
			}
			return
		} else if err != nil {
			t.Errorf("Error truncating file: %v", err)
		}

		b, err := vfs.ReadFile("/readme.txt")
		if err != nil {
			t.Errorf("Error reading truncated file: %v", err)
		}
		if int64(len(b)) != newSize {
			t.Errorf("File should be empty after truncation: %d", len(b))
		}
		if fi, err := vfs.Stat("/readme.txt"); err != nil {
			t.Errorf("Error stat file: %v", err)
		} else if fi.Size() != newSize {
			t.Errorf("Filesize should be %d after truncation", newSize)
		}
	}
}

func TestTruncateToZero(t *testing.T) {
	const content = "read me"
	vfs := NewFS()
	if err := vfs.WriteFile("/readme.txt", []byte(content), 0o666); err != nil {
		t.Errorf("Unexpected error writing file: %v", err)
	}

	f, err := vfs.OpenFile("/readme.txt", os.O_RDWR|os.O_TRUNC, 0o666)
	if err != nil {
		t.Errorf("Error opening file truncated: %v", err)
	}
	f.Close()

	b, err := vfs.ReadFile("/readme.txt")
	if err != nil {
		t.Errorf("Error reading truncated file: %v", err)
	}
	if len(b) != 0 {
		t.Errorf("File should be empty after truncation")
	}
	if fi, err := vfs.Stat("/readme.txt"); err != nil {
		t.Errorf("Error stat file: %v", err)
	} else if fi.Size() != 0 {
		t.Errorf("Filesize should be 0 after truncation")
	}
}

func TestStat(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	} else if n != len(dots) {
		t.Fatalf("Invalid write count: %d", n)
	}
	f.Close()

	if err := vfs.Mkdir("/tmp", 0o777); err != nil {
		t.Fatalf("Mkdir error: %v", err)
	}

	fi, err := vfs.Stat(f.Name())
	if err != nil {
		t.Errorf("Stat error: %v", err)
	}

	// Fileinfo name is base name
	if name := fi.Name(); name != "readme.txt" {
		t.Errorf("Invalid fileinfo name: %s", name)
	}

	// File name is abs name
	if name := f.Name(); name != "/readme.txt" {
		t.Errorf("Invalid file name: %s", name)
	}

	if s := fi.Size(); s != int64(len(dots)) {
		t.Errorf("Invalid size: %d", s)
	}
	if fi.IsDir() {
		t.Errorf("Invalid IsDir")
	}
}

func TestStatError(t *testing.T) {
	vfs := NewFS()
	path := "no-such-file"

	fi, err := vfs.Stat(path)
	if err == nil {
		t.Fatal("got nil, want error")
	}
	if fi != nil {
		t.Errorf("got %v, want nil", fi)
	}
	var perr *os.PathError
	if !errors.As(err, &perr) {
		t.Errorf("got %T, want %T", err, perr)
	}
}

func TestFstat(t *testing.T) {
	vfs := NewFS()
	file, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("open failed:", err)
	}
	if _, err = file.WriteString(abc); err != nil {
		t.Fatal("writing failed:", err)
	}
	if err = file.Sync(); err != nil {
		t.Fatal("syncing failed:", err)
	}
	defer file.Close()

	dir, err := file.Stat()
	if err != nil {
		t.Fatal("fstat failed:", err)
	}
	if filename != dir.Name() {
		t.Error("name should be ", filename, "; is", dir.Name())
	}
	filesize := int64(len(abc))
	if dir.Size() != filesize {
		t.Error("size should be", filesize, "; is", dir.Size())
	}
}

func TestRename(t *testing.T) {
	const content = "read me"
	vfs := NewFS()
	if err := vfs.WriteFile("/readme.txt", []byte(content), 0o666); err != nil {
		t.Errorf("Unexpected error writing file: %v", err)
	}

	if err := vfs.Rename("/readme.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file: %v", err)
	}

	if _, err := vfs.Stat("/readme.txt"); err == nil {
		t.Errorf("Old file still exists")
	}

	if _, err := vfs.Stat("/README.txt"); err != nil {
		t.Errorf("Error stat newfile: %v", err)
	}
	if b, err := vfs.ReadFile("/README.txt"); err != nil {
		t.Errorf("Error reading file: %v", err)
	} else if s := string(b); s != content {
		t.Errorf("Invalid content: %s", s)
	}

	// Rename unknown file
	if err := vfs.Rename("/nonexisting.txt", "/goodtarget.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	// Rename unknown file in nonexisting directory
	if err := vfs.Rename("/nonexisting/nonexisting.txt", "/goodtarget.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	// Rename existing file to nonexisting directory
	if err := vfs.Rename("/README.txt", "/nonexisting/nonexisting.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	if err := vfs.Mkdir("/newdirectory", 0o777); err != nil {
		t.Errorf("Error creating directory: %v", err)
	}

	if err := vfs.Rename("/README.txt", "/newdirectory/README.txt"); err != nil {
		t.Errorf("Error renaming file: %v", err)
	}

	// Create the same file again at root
	if err := vfs.WriteFile("/README.txt", []byte(content), 0o666); err != nil {
		t.Errorf("Unexpected error writing file: %v", err)
	}

	// Overwrite existing file
	if err := vfs.Rename("/newdirectory/README.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file")
	}
}

func TestRenameOverwriteDest(t *testing.T) {
	vfs := NewFS()
	from, to := renameFrom, renameTo

	toData := []byte("to")
	fromData := []byte("from")

	err := vfs.WriteFile(to, toData, 0o777)
	if err != nil {
		t.Fatalf("write file %q failed: %v", to, err)
	}

	err = vfs.WriteFile(from, fromData, 0o777)
	if err != nil {
		t.Fatalf("write file %q failed: %v", from, err)
	}
	err = vfs.Rename(from, to)
	if err != nil {
		t.Fatalf("rename %q, %q failed: %v", to, from, err)
	}

	_, err = vfs.Stat(from)
	if err == nil {
		t.Errorf("from file %q still exists", from)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stat from: %v", err)
	}
	toFi, err := vfs.Stat(to)
	if err != nil {
		t.Fatalf("stat %q failed: %v", to, err)
	}
	if toFi.Size() != int64(len(fromData)) {
		t.Errorf(`"to" size = %d; want %d (old "from" size)`, toFi.Size(), len(fromData))
	}
}

func TestRenameFailed(t *testing.T) {
	vfs := NewFS()
	from, to := renameFrom, renameTo

	err := vfs.Rename(from, to)
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if linkErr.Op != "rename" {
			t.Errorf("rename %q, %q: err.Op: want %q, got %q", from, to, "rename", linkErr.Op)
		}
		if linkErr.Old != from {
			t.Errorf("rename %q, %q: err.Old: want %q, got %q", from, to, from, linkErr.Old)
		}
		if linkErr.New != to {
			t.Errorf("rename %q, %q: err.New: want %q, got %q", from, to, to, linkErr.New)
		}
	} else if err == nil {
		t.Errorf("rename %q, %q: expected error, got nil", from, to)
	} else {
		t.Errorf("rename %q, %q: expected %T, got %T %v", from, to, new(os.LinkError), err, err)
	}
}

func TestRenameToDirFailed(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	from, to := renameFrom, renameTo

	is.NoErr(vfs.Mkdir(from, 0o777))
	is.NoErr(vfs.Mkdir(to, 0o777))

	err := vfs.Rename(from, to)
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if linkErr.Op != "rename" {
			t.Errorf("rename %q, %q: err.Op: want %q, got %q", from, to, "rename", linkErr.Op)
		}
		if linkErr.Old != from {
			t.Errorf("rename %q, %q: err.Old: want %q, got %q", from, to, from, linkErr.Old)
		}
		if linkErr.New != to {
			t.Errorf("rename %q, %q: err.New: want %q, got %q", from, to, to, linkErr.New)
		}
	} else if err == nil {
		t.Errorf("rename %q, %q: expected error, got nil", from, to)
	} else {
		t.Errorf("rename %q, %q: expected %T, got %T %v", from, to, new(os.LinkError), err, err)
	}
}

func checkSize(t *testing.T, f absfs.File, size int64) {
	t.Helper()

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", f.Name(), size, err)
	}
	if fi.Size() != size {
		t.Errorf("Stat %q: size %d want %d", f.Name(), fi.Size(), size)
	}
}

func TestFTruncate(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("create failed:", err)
	}
	defer f.Close()

	checkSize(t, f, 0)
	_, err = f.Write([]byte("hello, world\n"))
	is.NoErr(err)
	checkSize(t, f, 13)
	is.NoErr(f.Truncate(10))
	checkSize(t, f, 10)
	is.NoErr(f.Truncate(1024))
	checkSize(t, f, 1024)
	is.NoErr(f.Truncate(0))
	checkSize(t, f, 0)
	_, err = f.Write([]byte("surprise!"))
	if err == nil {
		checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
	}
}

func TestTruncate(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("create failed:", err)
	}
	defer f.Close()

	checkSize(t, f, 0)
	_, err = f.Write([]byte("hello, world\n"))
	is.NoErr(err)
	checkSize(t, f, 13)
	is.NoErr(vfs.Truncate(f.Name(), 10))
	checkSize(t, f, 10)
	is.NoErr(vfs.Truncate(f.Name(), 1024))
	checkSize(t, f, 1024)
	is.NoErr(vfs.Truncate(f.Name(), 0))
	checkSize(t, f, 0)
	_, err = f.Write([]byte("surprise!"))
	if err == nil {
		checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
	}
}

func TestChdir(t *testing.T) {
	vfs := NewFS()
	c := make(chan bool)
	cpwd := make(chan string)
	for i := range 10 {
		go func(i int) {
			// Lock half the goroutines in their own operating system
			// thread to exercise more scheduler possibilities.
			if i%2 == 1 {
				runtime.LockOSThread()
			}
			<-c
			pwd, err := vfs.Getwd()
			if err != nil {
				t.Errorf("Getwd on goroutine %d: %v", i, err)
				return
			}
			cpwd <- pwd
		}(i)
	}
	d, err := ioutil.TempDir(vfs, "", "test")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	if err := vfs.Chdir(d); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	d, err = vfs.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	close(c)
	for range 10 {
		pwd := <-cpwd
		if pwd != d {
			t.Errorf("Getwd returned %q; want %q", pwd, d)
		}
	}
}

func newFile(t *testing.T, fs absfs.FileSystem, testName string) (f absfs.File) {
	t.Helper()

	f, err := ioutil.TempFile(fs, "/", "_Go_"+testName)
	if err != nil {
		t.Fatalf("TempFile %s: %s", testName, err)
	}
	return
}

func TestSeek(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f := newFile(t, vfs, "TestSeek")
	defer f.Close()

	const data = "hello, world\n"
	_, err := io.WriteString(f, data)
	is.NoErr(err)

	type test struct {
		in     int64
		whence int
		out    int64
	}
	tests := []test{
		{0, io.SeekCurrent, int64(len(data))},
		{0, io.SeekStart, 0},
		{5, io.SeekStart, 5},
		{0, io.SeekEnd, int64(len(data))},
		{0, io.SeekStart, 0},
		{-1, io.SeekEnd, int64(len(data)) - 1},
		{1 << 33, io.SeekStart, 1 << 33},
		{1 << 33, io.SeekEnd, 1<<33 + int64(len(data))},

		// Issue 21681, Windows 4G-1, etc:
		{1<<32 - 1, io.SeekStart, 1<<32 - 1},
		{0, io.SeekCurrent, 1<<32 - 1},
		{2<<32 - 1, io.SeekStart, 2<<32 - 1},
		{0, io.SeekCurrent, 2<<32 - 1},
	}
	for i, tt := range tests {
		off, err := f.Seek(tt.in, tt.whence)
		if off != tt.out || err != nil {
			t.Errorf("#%d: Seek(%v, %v) = %v, %v want %v, nil", i, tt.in, tt.whence, off, err, tt.out)
		}
	}
}

func TestReadAt(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f := newFile(t, vfs, "TestReadAt")
	defer f.Close()

	const data = "hello, world\n"
	_, err := io.WriteString(f, data)
	is.NoErr(err)

	b := make([]byte, 5)
	n, err := f.ReadAt(b, 7)
	if err != nil || n != len(b) {
		t.Fatalf("ReadAt 7: %d, %v", n, err)
	}
	if string(b) != "world" {
		t.Fatalf("ReadAt 7: have %q want %q", string(b), "world")
	}
}

// Verify that ReadAt doesn't affect seek offset.
func TestReadAtOffset(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f := newFile(t, vfs, "TestReadAtOffset")
	defer f.Close()

	const data = "hello, world\n"
	_, err := io.WriteString(f, data)
	is.NoErr(err)

	_, err = f.Seek(0, 0)
	is.NoErr(err)
	b := make([]byte, 5)

	n, err := f.ReadAt(b, 7)
	if err != nil || n != len(b) {
		t.Fatalf("ReadAt 7: %d, %v", n, err)
	}
	if string(b) != "world" {
		t.Fatalf("ReadAt 7: have %q want %q", string(b), "world")
	}

	n, err = f.Read(b)
	if err != nil || n != len(b) {
		t.Fatalf("Read: %d, %v", n, err)
	}
	if string(b) != "hello" {
		t.Fatalf("Read: have %q want %q", string(b), "hello")
	}
}

// Verify that ReadAt doesn't allow negative offset.
func TestReadAtNegativeOffset(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f := newFile(t, vfs, "TestReadAtNegativeOffset")
	defer f.Close()

	const data = "hello, world\n"
	_, err := io.WriteString(f, data)
	is.NoErr(err)

	_, err = f.Seek(0, 0)
	is.NoErr(err)
	b := make([]byte, 5)

	n, err := f.ReadAt(b, -10)
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("ReadAt(-10) = %v, %v; want 0, %v", n, err, fs.ErrInvalid)
	}
}

func TestWriteAt(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()
	f := newFile(t, vfs, "TestWriteAt")
	defer f.Close()

	const data = "hello, world\n"
	_, err := io.WriteString(f, data)
	is.NoErr(err)

	n, err := f.WriteAt([]byte("WORLD"), 7)
	if err != nil || n != 5 {
		t.Fatalf("WriteAt 7: %d, %v", n, err)
	}

	b, err := vfs.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("ReadFile %s: %v", f.Name(), err)
	}
	if string(b) != "hello, WORLD\n" {
		t.Fatalf("after write: have %q want %q", string(b), "hello, WORLD\n")
	}
}

// Verify that WriteAt doesn't allow negative offset.
func TestWriteAtNegativeOffset(t *testing.T) {
	vfs := NewFS()
	f := newFile(t, vfs, "TestWriteAtNegativeOffset")
	defer f.Close()

	n, err := f.WriteAt([]byte("WORLD"), -10)
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("WriteAt(-10) = %v, %v; want 0, %v", n, err, fs.ErrInvalid)
	}
}

// Verify that WriteAt doesn't work in append mode.
func TestWriteAtInAppendMode(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("write_at_in_append_mode.txt", os.O_APPEND|os.O_CREATE, 0o666)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer f.Close()

	_, err = f.WriteAt([]byte(""), 1)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("f.WriteAt returned %v, expected %v", err, os.ErrPermission)
	}
}

//nolint:unparam
func writeFile(t *testing.T, vfs absfs.FileSystem, fname string, flag int, text string) string {
	t.Helper()

	f, err := vfs.OpenFile(fname, flag, 0o666)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n, err := io.WriteString(f, text)
	if err != nil {
		t.Fatalf("WriteString: %d, %v", n, err)
	}
	f.Close()
	data, err := vfs.ReadFile(fname)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	return string(data)
}

func TestAppend(t *testing.T) {
	vfs := NewFS()
	const f = "append.txt"
	s := writeFile(t, vfs, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: have %q want %q", s, "new")
	}
	s = writeFile(t, vfs, f, os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append")
	}
	s = writeFile(t, vfs, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append|append")
	}
	err := vfs.Remove(f)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	s = writeFile(t, vfs, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "new&append")
	if s != "new&append" {
		t.Fatalf("writeFile: after append have %q want %q", s, "new&append")
	}
	s = writeFile(t, vfs, f, os.O_CREATE|os.O_RDWR, "old")
	if s != "old&append" {
		t.Fatalf("writeFile: after create have %q want %q", s, "old&append")
	}
	s = writeFile(t, vfs, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: after truncate have %q want %q", s, "new")
	}
}

func TestModTime(t *testing.T) {
	is := is.New(t)

	vfs := NewFS()

	tBeforeWrite := time.Now()
	is.NoErr(vfs.WriteFile("/readme.txt", []byte{0, 0, 0}, 0o666))
	fi, _ := vfs.Stat("/readme.txt")
	mtimeAfterWrite := fi.ModTime()

	if !mtimeAfterWrite.After(tBeforeWrite) {
		t.Error("Open should modify mtime")
	}

	f, err := vfs.OpenFile("/readme.txt", os.O_RDONLY, 0o666)
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}
	f.Close()
	tAfterRead := fi.ModTime()

	if tAfterRead != mtimeAfterWrite {
		t.Error("Open with O_RDONLY should not modify mtime")
	}
}
