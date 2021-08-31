package vfs

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

const (
	dots = "1....2....3....4"
	abc  = "abcdefghijklmnop"
)

func TestVFS(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	vfs := NewFS()

	if err := vfs.Mkdir("memz", 0777); err != nil {
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

	err := vfs.MkdirAll(testdir, 0777)
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
	if err != io.EOF {
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
		f, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}
		if name := f.Name(); name != "/testfile" {
			t.Errorf("Wrong name: %s", name)
		}
	}

	// Create same file again
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}

	}

	// Create same file again, but truncate it
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}
	}

	// Create same file again with O_CREATE|O_EXCL, which is an error
	{
		_, err := vfs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			t.Fatalf("Expected error creating file: %s", err)
		}
	}

	// Create file with unknown parent
	{
		_, err := vfs.OpenFile("/testfile/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err == nil {
			t.Errorf("Expected error creating file")
		}
	}

	// Create file with relative path (workingDir == root)
	{
		f, err := vfs.OpenFile("relFile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
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
			t.Fatalf("Unexpected error creating directory: %s", err)
		}
	}

	// Create dir with relative path
	{
		err := vfs.Mkdir("home", 0)
		if err != nil {
			t.Fatalf("Unexpected error creating directory: %s", err)
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
		t.Fatalf("Unexpected error creating directory /home: %s", err)
	}

	err = vfs.Mkdir("/home/blang", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang: %s", err)
	}

	err = vfs.Mkdir("/home/blang/goprojects", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang/goprojects: %s", err)
	}

	err = vfs.Mkdir("/home/johndoe/goprojects", 0)
	if err == nil {
		t.Errorf("Expected error creating directory with non-existing parent")
	}

	// TODO: Subdir of file
}

func TestRemove(t *testing.T) {
	vfs := NewFS()
	err := vfs.Mkdir("/tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir error: %s", err)
	}
	f, err := vfs.OpenFile("/tmp/README.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatalf("Create error: %s", err)
	}
	if _, err := f.Write([]byte("test")); err != nil {
		t.Fatalf("Write error: %s", err)
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
		t.Errorf("Remove failed: %s", err)
	}

	if _, err = vfs.OpenFile("/tmp/README.txt", os.O_RDWR, 0666); err == nil {
		t.Errorf("Could open removed file!")
	}

	err = vfs.Remove("/tmp")
	if err != nil {
		t.Errorf("Remove failed: %s", err)
	}
	/*if fis, err := vfs.ReadDir("/"); err != nil {
		t.Errorf("Readdir error: %s", err)
	} else if len(fis) != 0 {
		t.Errorf("Found files: %s", fis)
	}*/
}

// Read with length 0 should not return EOF.
func TestRead0(t *testing.T) {
	vfs := NewFS()
	filename := "testfile"
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
	filename := "testfile"
	file, err := vfs.Create(filename)
	if err != nil {
		t.Fatal("open failed:", err)
	}
	file.Close() // close immediately

	b := make([]byte, 100)
	_, err = file.Read(b)

	e, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("Read: %T(%v), want PathError", e, e)
	}

	if e.Err != os.ErrClosed {
		t.Errorf("Read: %v, want PathError(ErrClosed)", e)
	}
}

func TestReadWrite(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Write abc
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Seek to end of file
	if n, err := f.Seek(0, io.SeekEnd); err != nil || n != 32 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Write dots at end of file
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	p := make([]byte, len(dots)+len(abc)+len(dots))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc)+len(dots) {
		t.Errorf("Read error: %d %s", n, err)
	} else if s := string(p); s != dots+abc+dots {
		t.Errorf("Invalid read: %s", s)
	}
}

func TestOpenRO(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// Write first dots
	if _, err := f.Write([]byte(dots)); err == nil {
		t.Fatalf("Expected write error")
	}
	f.Close()
}

func TestOpenWO(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Try reading
	p := make([]byte, len(dots))
	if n, err := f.Read(p); err == nil || n > 0 {
		t.Errorf("Expected invalid read: %d %s", n, err)
	}

	f.Close()
}

func TestOpenAppend(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}
	f.Close()

	// Reopen file in append mode
	f, err = vfs.OpenFile("/readme.txt", os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// append dots
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, io.SeekStart); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	p := make([]byte, len(dots)+len(abc))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc) {
		t.Errorf("Read error: %d %s", n, err)
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
		f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			t.Fatalf("Could not open file: %s", err)
		}
		if n, err := f.Write([]byte(dots)); err != nil {
			t.Errorf("Unexpected error: %s", err)
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
			t.Errorf("Error truncating file: %s", err)
		}

		b, err := ioutil.ReadFile(vfs, "/readme.txt")
		if err != nil {
			t.Errorf("Error reading truncated file: %s", err)
		}
		if int64(len(b)) != newSize {
			t.Errorf("File should be empty after truncation: %d", len(b))
		}
		if fi, err := vfs.Stat("/readme.txt"); err != nil {
			t.Errorf("Error stat file: %s", err)
		} else if fi.Size() != newSize {
			t.Errorf("Filesize should be %d after truncation", newSize)
		}
	}
}

func TestTruncateToZero(t *testing.T) {
	const content = "read me"
	vfs := NewFS()
	if err := ioutil.WriteFile(vfs, "/readme.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	f, err := vfs.OpenFile("/readme.txt", os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		t.Errorf("Error opening file truncated: %s", err)
	}
	f.Close()

	b, err := ioutil.ReadFile(vfs, "/readme.txt")
	if err != nil {
		t.Errorf("Error reading truncated file: %s", err)
	}
	if len(b) != 0 {
		t.Errorf("File should be empty after truncation")
	}
	if fi, err := vfs.Stat("/readme.txt"); err != nil {
		t.Errorf("Error stat file: %s", err)
	} else if fi.Size() != 0 {
		t.Errorf("Filesize should be 0 after truncation")
	}
}

func TestStat(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Fatalf("Invalid write count: %d", n)
	}
	f.Close()

	if err := vfs.Mkdir("/tmp", 0777); err != nil {
		t.Fatalf("Mkdir error: %s", err)
	}

	fi, err := vfs.Stat(f.Name())
	if err != nil {
		t.Errorf("Stat error: %s", err)
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
	if perr, ok := err.(*os.PathError); !ok {
		t.Errorf("got %T, want %T", err, perr)
	}
}

func TestFstat(t *testing.T) {
	vfs := NewFS()
	filename := "testfile"
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
	if err := ioutil.WriteFile(vfs, "/readme.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	if err := vfs.Rename("/readme.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file: %s", err)
	}

	if _, err := vfs.Stat("/readme.txt"); err == nil {
		t.Errorf("Old file still exists")
	}

	if _, err := vfs.Stat("/README.txt"); err != nil {
		t.Errorf("Error stat newfile: %s", err)
	}
	if b, err := ioutil.ReadFile(vfs, "/README.txt"); err != nil {
		t.Errorf("Error reading file: %s", err)
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

	if err := vfs.Mkdir("/newdirectory", 0777); err != nil {
		t.Errorf("Error creating directory: %s", err)
	}

	if err := vfs.Rename("/README.txt", "/newdirectory/README.txt"); err != nil {
		t.Errorf("Error renaming file: %s", err)
	}

	// Create the same file again at root
	if err := ioutil.WriteFile(vfs, "/README.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	// Overwrite existing file
	if err := vfs.Rename("/newdirectory/README.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file")
	}
}

func TestRenameOverwriteDest(t *testing.T) {
	vfs := NewFS()
	from, to := "renamefrom", "renameto"

	toData := []byte("to")
	fromData := []byte("from")

	err := ioutil.WriteFile(vfs, to, toData, 0777)
	if err != nil {
		t.Fatalf("write file %q failed: %v", to, err)
	}

	err = ioutil.WriteFile(vfs, from, fromData, 0777)
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
	if err != nil && !os.IsNotExist(err) {
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
	from, to := "renamefrom", "renameto"

	err := vfs.Rename(from, to)
	switch err := err.(type) {
	case *os.LinkError:
		if err.Op != "rename" {
			t.Errorf("rename %q, %q: err.Op: want %q, got %q", from, to, "rename", err.Op)
		}
		if err.Old != from {
			t.Errorf("rename %q, %q: err.Old: want %q, got %q", from, to, from, err.Old)
		}
		if err.New != to {
			t.Errorf("rename %q, %q: err.New: want %q, got %q", from, to, to, err.New)
		}
	case nil:
		t.Errorf("rename %q, %q: expected error, got nil", from, to)
	default:
		t.Errorf("rename %q, %q: expected %T, got %T %v", from, to, new(os.LinkError), err, err)
	}
}

func TestRenameToDirFailed(t *testing.T) {
	vfs := NewFS()
	from, to := "renamefrom", "renameto"

	vfs.Mkdir(from, 0777)
	vfs.Mkdir(to, 0777)

	err := vfs.Rename(from, to)
	switch err := err.(type) {
	case *os.LinkError:
		if err.Op != "rename" {
			t.Errorf("rename %q, %q: err.Op: want %q, got %q", from, to, "rename", err.Op)
		}
		if err.Old != from {
			t.Errorf("rename %q, %q: err.Old: want %q, got %q", from, to, from, err.Old)
		}
		if err.New != to {
			t.Errorf("rename %q, %q: err.New: want %q, got %q", from, to, to, err.New)
		}
	case nil:
		t.Errorf("rename %q, %q: expected error, got nil", from, to)
	default:
		t.Errorf("rename %q, %q: expected %T, got %T %v", from, to, new(os.LinkError), err, err)
	}
}

func checkSize(t *testing.T, f absfs.File, size int64) {
	dir, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", f.Name(), size, err)
	}
	if dir.Size() != size {
		t.Errorf("Stat %q: size %d want %d", f.Name(), dir.Size(), size)
	}
}

func TestFTruncate(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.Create("testfile")
	if err != nil {
		t.Fatal("create failed:", err)
	}
	defer f.Close()

	checkSize(t, f, 0)
	f.Write([]byte("hello, world\n"))
	checkSize(t, f, 13)
	f.Truncate(10)
	checkSize(t, f, 10)
	f.Truncate(1024)
	checkSize(t, f, 1024)
	f.Truncate(0)
	checkSize(t, f, 0)
	_, err = f.Write([]byte("surprise!"))
	if err == nil {
		checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
	}
}

func TestTruncate(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.Create("testfile")
	if err != nil {
		t.Fatal("create failed:", err)
	}
	defer f.Close()

	checkSize(t, f, 0)
	f.Write([]byte("hello, world\n"))
	checkSize(t, f, 13)
	vfs.Truncate(f.Name(), 10)
	checkSize(t, f, 10)
	vfs.Truncate(f.Name(), 1024)
	checkSize(t, f, 1024)
	vfs.Truncate(f.Name(), 0)
	checkSize(t, f, 0)
	_, err = f.Write([]byte("surprise!"))
	if err == nil {
		checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
	}
}

func TestChdir(t *testing.T) {
	vfs := NewFS()
	const N = 10
	c := make(chan bool)
	cpwd := make(chan string)
	for i := 0; i < N; i++ {
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
	for i := 0; i < N; i++ {
		pwd := <-cpwd
		if pwd != d {
			t.Errorf("Getwd returned %q; want %q", pwd, d)
		}
	}
}

func newFile(testName string, fs absfs.FileSystem, t *testing.T) (f absfs.File) {
	f, err := ioutil.TempFile(fs, "/", "_Go_"+testName)
	if err != nil {
		t.Fatalf("TempFile %s: %s", testName, err)
	}
	return
}

func TestSeek(t *testing.T) {
	vfs := NewFS()
	f := newFile("TestSeek", vfs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

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
	vfs := NewFS()
	f := newFile("TestReadAt", vfs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

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
	vfs := NewFS()
	f := newFile("TestReadAtOffset", vfs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

	f.Seek(0, 0)
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
	vfs := NewFS()
	f := newFile("TestReadAtNegativeOffset", vfs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

	f.Seek(0, 0)
	b := make([]byte, 5)

	n, err := f.ReadAt(b, -10)

	const wantsub = "negative offset"
	if !strings.Contains(fmt.Sprint(err), wantsub) || n != 0 {
		t.Errorf("ReadAt(-10) = %v, %v; want 0, ...%q...", n, err, wantsub)
	}
}

func TestWriteAt(t *testing.T) {
	vfs := NewFS()
	f := newFile("TestWriteAt", vfs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

	n, err := f.WriteAt([]byte("WORLD"), 7)
	if err != nil || n != 5 {
		t.Fatalf("WriteAt 7: %d, %v", n, err)
	}

	b, err := ioutil.ReadFile(vfs, f.Name())
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
	f := newFile("TestWriteAtNegativeOffset", vfs, t)
	defer f.Close()

	n, err := f.WriteAt([]byte("WORLD"), -10)

	const wantsub = "negative offset"
	if !strings.Contains(fmt.Sprint(err), wantsub) || n != 0 {
		t.Errorf("WriteAt(-10) = %v, %v; want 0, ...%q...", n, err, wantsub)
	}
}

// Verify that WriteAt doesn't work in append mode.
func TestWriteAtInAppendMode(t *testing.T) {
	vfs := NewFS()
	f, err := vfs.OpenFile("write_at_in_append_mode.txt", os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer f.Close()

	_, err = f.WriteAt([]byte(""), 1)
	if err != os.ErrPermission {
		t.Fatalf("f.WriteAt returned %v, expected %v", err, os.ErrPermission)
	}
}

func writeFile(vfs absfs.FileSystem, t *testing.T, fname string, flag int, text string) string {
	f, err := vfs.OpenFile(fname, flag, 0666)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n, err := io.WriteString(f, text)
	if err != nil {
		t.Fatalf("WriteString: %d, %v", n, err)
	}
	f.Close()
	data, err := ioutil.ReadFile(vfs, fname)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

func TestAppend(t *testing.T) {
	vfs := NewFS()
	const f = "append.txt"
	s := writeFile(vfs, t, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: have %q want %q", s, "new")
	}
	s = writeFile(vfs, t, f, os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append")
	}
	s = writeFile(vfs, t, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append|append")
	}
	err := vfs.Remove(f)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	s = writeFile(vfs, t, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "new&append")
	if s != "new&append" {
		t.Fatalf("writeFile: after append have %q want %q", s, "new&append")
	}
	s = writeFile(vfs, t, f, os.O_CREATE|os.O_RDWR, "old")
	if s != "old&append" {
		t.Fatalf("writeFile: after create have %q want %q", s, "old&append")
	}
	s = writeFile(vfs, t, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: after truncate have %q want %q", s, "new")
	}
}

func TestModTime(t *testing.T) {
	vfs := NewFS()

	tBeforeWrite := time.Now()
	ioutil.WriteFile(vfs, "/readme.txt", []byte{0, 0, 0}, 0666)
	fi, _ := vfs.Stat("/readme.txt")
	mtimeAfterWrite := fi.ModTime()

	if !mtimeAfterWrite.After(tBeforeWrite) {
		t.Error("Open should modify mtime")
	}

	f, err := vfs.OpenFile("/readme.txt", os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	f.Close()
	tAfterRead := fi.ModTime()

	if tAfterRead != mtimeAfterWrite {
		t.Error("Open with O_RDONLY should not modify mtime")
	}
}
