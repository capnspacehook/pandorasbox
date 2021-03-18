package vfs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/fstesting"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

const (
	dots = "1....2....3....4"
	abc  = "abcdefghijklmnop"
)

func TestWalk(t *testing.T) {
	fs := NewFS()
	testpath := ".."
	abs, err := filepath.Abs(testpath)
	if err != nil {
		t.Fatal(err)
	}

	testpath = abs

	err = filepath.Walk(testpath, func(path string, info os.FileInfo, err error) error {
		p := strings.TrimPrefix(path, testpath)
		if p == "" {
			return nil
		}
		if info.IsDir() {
			fs.MkdirAll(p, info.Mode())
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		fout, err := fs.Create(p)
		if err != nil {
			return err
		}
		defer fout.Close()
		fin, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fin.Close()
		io.Copy(fout, fin)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Walk", func(t *testing.T) {
		list := make(map[string]bool)
		count := 0
		err = filepath.Walk(testpath, func(path string, info os.FileInfo, err error) error {
			p := strings.TrimPrefix(path, testpath)
			if p == "" {
				p = "/"
			}
			if info.Mode().IsDir() {
				count++
				list[p] = true
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			list[p] = true
			count++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		count2 := 0
		err = fs.Walk("/", func(path string, info os.FileInfo, err error) error {
			if !list[path] {
				return fmt.Errorf("file not found %q", path)
			}
			delete(list, path)
			count2++
			if count2 > count {
				return fmt.Errorf("file count overflow")
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		if count < 10 || count != count2 {
			t.Errorf("incorrect file count: %d, %d", count, count2)
		}
		if len(list) > 0 {
			i := 0

			for k := range list {
				i++
				if i > 10 {
					break
				}
				t.Errorf("path not removed %q", k)
			}
		}
	})
}

func TestVFS(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	fs := NewFS()

	if fs.TempDir() != "/tmp" {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), "/tmp")
	}
	fs.Tempdir = os.TempDir()
	if fs.TempDir() != os.TempDir() {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), os.TempDir())
	}

	testdir := fs.TempDir()
	timestr := time.Now().Format(time.RFC3339)
	testdir = filepath.Join(testdir, fmt.Sprintf("fstesting%s", timestr))

	err := fs.MkdirAll(testdir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveAll(fs.TempDir())

	cwd, err := fs.Getwd()
	if cwd != "/" {
		t.Fatalf("incorrect cwd %q", cwd)
	}
	err = fs.Chdir(testdir)
	if err != nil {
		t.Fatal(err)
	}

	maxerrors := 10
	fstesting.AutoTest(0, func(testcase *fstesting.Testcase) error {
		result, err := fstesting.FsTest(fs, filepath.Dir(testcase.Path), testcase)
		if err != nil {
			t.Fatal(err)
		}
		Errors := result.Errors

		for op, report := range testcase.Errors {
			if Errors[op] == nil {
				t.Fatalf("%d: On %q got nil but expected to get an err of type (%T)\n", testcase.TestNo, op, testcase.Errors[op].Type())
				continue
			}
			if report.Err == nil {
				if Errors[op].Err == nil {
					continue
				}

				t.Fatalf("%d: On %q expected `err == nil` but got err: (%T) %q\n%s", testcase.TestNo, op, Errors[op].Type(), Errors[op].String(), Errors[op].Stack())
				maxerrors--
				continue
			}

			if Errors[op].Err == nil {
				t.Errorf("%d: On %q got `err == nil` but expected err: (%T) %q\n%s", testcase.TestNo, op, testcase.Errors[op].Type(), testcase.Errors[op].String(), Errors[op].Stack())
				maxerrors--
			}
			if !report.TypesEqual(Errors[op]) {
				t.Errorf("%d: On %q got different error types, expected (%T) but got (%T)\n", testcase.TestNo, op, report.Type(), Errors[op].Type())
				maxerrors--
			}
			if report.Error() != Errors[op].Error() {
				t.Errorf("%d: On %q got different error values,\nexpecte, got:\n%q\n%q\n%s", testcase.TestNo, op, report.Error(), Errors[op].Error(), Errors[op].Stack())
				maxerrors--
			}

			if maxerrors < 1 {
				t.Fatal("too many errors")
			}
			fmt.Printf("  %10d Tests\r", testcase.TestNo)
		}
		return nil
	})
	if err != nil && err.Error() != "stop" {
		t.Fatal(err)
	}
}

func TestMkdir(t *testing.T) {
	fs := NewFS()

	if fs.TempDir() != "/tmp" {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), "/tmp")
	}

	fs.Tempdir = os.TempDir()
	if fs.TempDir() != os.TempDir() {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), os.TempDir())
	}

	testdir := fs.TempDir()

	t.Logf("Test path: %q", testdir)
	err := fs.MkdirAll(testdir, 0777)
	if err != nil {
		t.Fatal(err)
	}

	var list []string
	path := "/"
outer:
	for _, name := range strings.Split(testdir, "/")[1:] {
		if name == "" {
			continue
		}
		f, err := fs.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		list, err = f.Readdirnames(-1)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range list {
			if n == name {
				path = filepath.Join(path, name)
				continue outer
			}
		}
		t.Errorf("path error: %q + %q:  %s", path, name, list)
	}
}

func TestOpenWrite(t *testing.T) {
	fs := NewFS()

	f, err := fs.Create("/test_file.txt")
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

	f, err = fs.Open("/test_file.txt")
	if err != nil {
		t.Fatal(err)
	}
	buff := make([]byte, 512)
	n, err = f.Read(buff)
	f.Close()
	if n != len(data) {
		t.Errorf("write error: wrong byte count %d, expected %d", n, len(data))
	}
	if err != nil {
		t.Fatal(err)
	}
	buff = buff[:n]
	if bytes.Compare(data, buff) != 0 {
		t.Log(string(data))
		t.Log(string(buff))

		t.Fatal("bytes written do not compare to bytes read")
	}
}

func TestCreate(t *testing.T) {
	fs := NewFS()
	// Create file with absolute path
	{
		f, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}
		if name := f.Name(); name != "/testfile" {
			t.Errorf("Wrong name: %s", name)
		}
	}

	// Create same file again
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}

	}

	// Create same file again, but truncate it
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}
	}

	// Create same file again with O_CREATE|O_EXCL, which is an error
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			t.Fatalf("Expected error creating file: %s", err)
		}
	}

	// Create file with unknown parent
	{
		_, err := fs.OpenFile("/testfile/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err == nil {
			t.Errorf("Expected error creating file")
		}
	}

	// Create file with relative path (workingDir == root)
	{
		f, err := fs.OpenFile("relFile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			t.Fatalf("Unexpected error creating file: %s", err)
		}
		if name := f.Name(); name != "relFile" {
			t.Errorf("Wrong name: %s", name)
		}
	}
}

func TestMkdirAbsRel(t *testing.T) {
	fs := NewFS()

	// Create dir with absolute path
	{
		err := fs.Mkdir("/usr", 0)
		if err != nil {
			t.Fatalf("Unexpected error creating directory: %s", err)
		}
	}

	// Create dir with relative path
	{
		err := fs.Mkdir("home", 0)
		if err != nil {
			t.Fatalf("Unexpected error creating directory: %s", err)
		}
	}

	// Create dir twice
	{
		err := fs.Mkdir("/home", 0)
		if err == nil {
			t.Fatalf("Expecting error creating directory: %s", "/home")
		}
	}
}

func TestMkdirTree(t *testing.T) {
	fs := NewFS()

	err := fs.Mkdir("/home", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home: %s", err)
	}

	err = fs.Mkdir("/home/blang", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang: %s", err)
	}

	err = fs.Mkdir("/home/blang/goprojects", 0)
	if err != nil {
		t.Fatalf("Unexpected error creating directory /home/blang/goprojects: %s", err)
	}

	err = fs.Mkdir("/home/johndoe/goprojects", 0)
	if err == nil {
		t.Errorf("Expected error creating directory with non-existing parent")
	}

	// TODO: Subdir of file
}

func TestRemove(t *testing.T) {
	fs := NewFS()
	err := fs.Mkdir("/tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir error: %s", err)
	}
	f, err := fs.OpenFile("/tmp/README.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatalf("Create error: %s", err)
	}
	if _, err := f.Write([]byte("test")); err != nil {
		t.Fatalf("Write error: %s", err)
	}
	f.Close()

	// remove non existing file
	if err := fs.Remove("/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove non existing file from an non existing directory
	if err := fs.Remove("/nonexisting/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove created file
	err = fs.Remove(f.Name())
	if err != nil {
		t.Errorf("Remove failed: %s", err)
	}

	if _, err = fs.OpenFile("/tmp/README.txt", os.O_RDWR, 0666); err == nil {
		t.Errorf("Could open removed file!")
	}

	err = fs.Remove("/tmp")
	if err != nil {
		t.Errorf("Remove failed: %s", err)
	}
	/*if fis, err := fs.ReadDir("/"); err != nil {
		t.Errorf("Readdir error: %s", err)
	} else if len(fis) != 0 {
		t.Errorf("Found files: %s", fis)
	}*/
}

// Read with length 0 should not return EOF.
func TestRead0(t *testing.T) {
	fs := NewFS()
	filename := "testfile"
	f, err := fs.Create(filename)
	if err != nil {
		t.Fatal("open failed:", err)
	}
	if _, err = f.WriteString(abc); err != nil {
		t.Fatal("writing failed:", err)
	}
	defer f.Close()

	b := make([]byte, 0)
	n, err := f.Read(b)
	if n != 0 || err != nil {
		t.Errorf("Read(0) = %d, %v, want 0, nil", n, err)
	}
	b = make([]byte, 100)
	n, err = f.ReadAt(b, 0)
	if n <= 0 || err != nil {
		t.Errorf("Read(100) = %d, %v, want >0, nil", n, err)
	}
}

// Reading a closed file should return ErrClosed error
func TestReadClosed(t *testing.T) {
	fs := NewFS()
	filename := "testfile"
	file, err := fs.Create(filename)
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
	fs := NewFS()
	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
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
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Seek to end of file
	if n, err := f.Seek(0, os.SEEK_END); err != nil || n != 32 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Write dots at end of file
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
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
	fs := NewFS()
	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDONLY, 0666)
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
	fs := NewFS()
	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_WRONLY, 0666)
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
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
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
	fs := NewFS()
	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
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
	f, err = fs.OpenFile("/readme.txt", os.O_APPEND|os.O_RDWR, 0666)
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
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
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
		fs := NewFS()
		f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
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
		err = fs.Truncate("/readme.txt", newSize)
		if param.err {
			if err == nil {
				t.Errorf("Error expected truncating file to length %d", newSize)
			}
			return
		} else if err != nil {
			t.Errorf("Error truncating file: %s", err)
		}

		b, err := ioutil.ReadFile(fs, "/readme.txt")
		if err != nil {
			t.Errorf("Error reading truncated file: %s", err)
		}
		if int64(len(b)) != newSize {
			t.Errorf("File should be empty after truncation: %d", len(b))
		}
		if fi, err := fs.Stat("/readme.txt"); err != nil {
			t.Errorf("Error stat file: %s", err)
		} else if fi.Size() != newSize {
			t.Errorf("Filesize should be %d after truncation", newSize)
		}
	}
}

func TestTruncateToZero(t *testing.T) {
	const content = "read me"
	fs := NewFS()
	if err := ioutil.WriteFile(fs, "/readme.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	f, err := fs.OpenFile("/readme.txt", os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		t.Errorf("Error opening file truncated: %s", err)
	}
	f.Close()

	b, err := ioutil.ReadFile(fs, "/readme.txt")
	if err != nil {
		t.Errorf("Error reading truncated file: %s", err)
	}
	if len(b) != 0 {
		t.Errorf("File should be empty after truncation")
	}
	if fi, err := fs.Stat("/readme.txt"); err != nil {
		t.Errorf("Error stat file: %s", err)
	} else if fi.Size() != 0 {
		t.Errorf("Filesize should be 0 after truncation")
	}
}

func TestStat(t *testing.T) {
	fs := NewFS()
	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
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

	if err := fs.Mkdir("/tmp", 0777); err != nil {
		t.Fatalf("Mkdir error: %s", err)
	}

	fi, err := fs.Stat(f.Name())
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
	if m := fi.Mode(); m != fs.Umask&0666 {
		t.Errorf("Invalid mode: %d", m)
	}
}

func TestStatError(t *testing.T) {
	fs := NewFS()
	path := "no-such-file"

	fi, err := fs.Stat(path)
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
	fs := NewFS()
	filename := "testfile"
	file, err := fs.Create(filename)
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
	fs := NewFS()
	if err := ioutil.WriteFile(fs, "/readme.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	if err := fs.Rename("/readme.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file: %s", err)
	}

	if _, err := fs.Stat("/readme.txt"); err == nil {
		t.Errorf("Old file still exists")
	}

	if _, err := fs.Stat("/README.txt"); err != nil {
		t.Errorf("Error stat newfile: %s", err)
	}
	if b, err := ioutil.ReadFile(fs, "/README.txt"); err != nil {
		t.Errorf("Error reading file: %s", err)
	} else if s := string(b); s != content {
		t.Errorf("Invalid content: %s", s)
	}

	// Rename unknown file
	if err := fs.Rename("/nonexisting.txt", "/goodtarget.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	// Rename unknown file in nonexisting directory
	if err := fs.Rename("/nonexisting/nonexisting.txt", "/goodtarget.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	// Rename existing file to nonexisting directory
	if err := fs.Rename("/README.txt", "/nonexisting/nonexisting.txt"); err == nil {
		t.Errorf("Expected error renaming file")
	}

	if err := fs.Mkdir("/newdirectory", 0777); err != nil {
		t.Errorf("Error creating directory: %s", err)
	}

	if err := fs.Rename("/README.txt", "/newdirectory/README.txt"); err != nil {
		t.Errorf("Error renaming file: %s", err)
	}

	// Create the same file again at root
	if err := ioutil.WriteFile(fs, "/README.txt", []byte(content), 0666); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	// Overwrite existing file
	if err := fs.Rename("/newdirectory/README.txt", "/README.txt"); err != nil {
		t.Errorf("Unexpected error renaming file")
	}
}

func TestRenameOverwriteDest(t *testing.T) {
	fs := NewFS()
	from, to := "renamefrom", "renameto"

	toData := []byte("to")
	fromData := []byte("from")

	err := ioutil.WriteFile(fs, to, toData, 0777)
	if err != nil {
		t.Fatalf("write file %q failed: %v", to, err)
	}

	err = ioutil.WriteFile(fs, from, fromData, 0777)
	if err != nil {
		t.Fatalf("write file %q failed: %v", from, err)
	}
	err = fs.Rename(from, to)
	if err != nil {
		t.Fatalf("rename %q, %q failed: %v", to, from, err)
	}

	_, err = fs.Stat(from)
	if err == nil {
		t.Errorf("from file %q still exists", from)
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat from: %v", err)
	}
	toFi, err := fs.Stat(to)
	if err != nil {
		t.Fatalf("stat %q failed: %v", to, err)
	}
	if toFi.Size() != int64(len(fromData)) {
		t.Errorf(`"to" size = %d; want %d (old "from" size)`, toFi.Size(), len(fromData))
	}
}

func TestRenameFailed(t *testing.T) {
	fs := NewFS()
	from, to := "renamefrom", "renameto"

	err := fs.Rename(from, to)
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
	fs := NewFS()
	from, to := "renamefrom", "renameto"

	fs.Mkdir(from, 0777)
	fs.Mkdir(to, 0777)

	err := fs.Rename(from, to)
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
	fs := NewFS()
	f, err := fs.Create("testfile")
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
	fs := NewFS()
	f, err := fs.Create("testfile")
	if err != nil {
		t.Fatal("create failed:", err)
	}
	defer f.Close()

	checkSize(t, f, 0)
	f.Write([]byte("hello, world\n"))
	checkSize(t, f, 13)
	fs.Truncate(f.Name(), 10)
	checkSize(t, f, 10)
	fs.Truncate(f.Name(), 1024)
	checkSize(t, f, 1024)
	fs.Truncate(f.Name(), 0)
	checkSize(t, f, 0)
	_, err = f.Write([]byte("surprise!"))
	if err == nil {
		checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
	}
}

func TestChdir(t *testing.T) {
	fs := NewFS()
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
			pwd, err := fs.Getwd()
			if err != nil {
				t.Errorf("Getwd on goroutine %d: %v", i, err)
				return
			}
			cpwd <- pwd
		}(i)
	}
	d, err := ioutil.TempDir(fs, "", "test")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	if err := fs.Chdir(d); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	d, err = fs.Getwd()
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

func newFile(testName string, fs *FileSystem, t *testing.T) (f absfs.File) {
	f, err := ioutil.TempFile(fs, "/", "_Go_"+testName)
	if err != nil {
		t.Fatalf("TempFile %s: %s", testName, err)
	}
	return
}

func TestSeek(t *testing.T) {
	fs := NewFS()
	f := newFile("TestSeek", fs, t)
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
	fs := NewFS()
	f := newFile("TestReadAt", fs, t)
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
	fs := NewFS()
	f := newFile("TestReadAtOffset", fs, t)
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
	fs := NewFS()
	f := newFile("TestReadAtNegativeOffset", fs, t)
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
	fs := NewFS()
	f := newFile("TestWriteAt", fs, t)
	defer f.Close()

	const data = "hello, world\n"
	io.WriteString(f, data)

	n, err := f.WriteAt([]byte("WORLD"), 7)
	if err != nil || n != 5 {
		t.Fatalf("WriteAt 7: %d, %v", n, err)
	}

	b, err := ioutil.ReadFile(fs, f.Name())
	if err != nil {
		t.Fatalf("ReadFile %s: %v", f.Name(), err)
	}
	if string(b) != "hello, WORLD\n" {
		t.Fatalf("after write: have %q want %q", string(b), "hello, WORLD\n")
	}
}

// Verify that WriteAt doesn't allow negative offset.
func TestWriteAtNegativeOffset(t *testing.T) {
	fs := NewFS()
	f := newFile("TestWriteAtNegativeOffset", fs, t)
	defer f.Close()

	n, err := f.WriteAt([]byte("WORLD"), -10)

	const wantsub = "negative offset"
	if !strings.Contains(fmt.Sprint(err), wantsub) || n != 0 {
		t.Errorf("WriteAt(-10) = %v, %v; want 0, ...%q...", n, err, wantsub)
	}
}

// Verify that WriteAt doesn't work in append mode.
func TestWriteAtInAppendMode(t *testing.T) {
	fs := NewFS()
	f, err := fs.OpenFile("write_at_in_append_mode.txt", os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer f.Close()

	_, err = f.WriteAt([]byte(""), 1)
	if err != os.ErrPermission {
		t.Fatalf("f.WriteAt returned %v, expected %v", err, os.ErrPermission)
	}
}

func writeFile(fs *FileSystem, t *testing.T, fname string, flag int, text string) string {
	f, err := fs.OpenFile(fname, flag, 0666)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n, err := io.WriteString(f, text)
	if err != nil {
		t.Fatalf("WriteString: %d, %v", n, err)
	}
	f.Close()
	data, err := ioutil.ReadFile(fs, fname)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

func TestAppend(t *testing.T) {
	fs := NewFS()
	const f = "append.txt"
	s := writeFile(fs, t, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: have %q want %q", s, "new")
	}
	s = writeFile(fs, t, f, os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append")
	}
	s = writeFile(fs, t, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "|append")
	if s != "new|append|append" {
		t.Fatalf("writeFile: have %q want %q", s, "new|append|append")
	}
	err := fs.Remove(f)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	s = writeFile(fs, t, f, os.O_CREATE|os.O_APPEND|os.O_RDWR, "new&append")
	if s != "new&append" {
		t.Fatalf("writeFile: after append have %q want %q", s, "new&append")
	}
	s = writeFile(fs, t, f, os.O_CREATE|os.O_RDWR, "old")
	if s != "old&append" {
		t.Fatalf("writeFile: after create have %q want %q", s, "old&append")
	}
	s = writeFile(fs, t, f, os.O_CREATE|os.O_TRUNC|os.O_RDWR, "new")
	if s != "new" {
		t.Fatalf("writeFile: after truncate have %q want %q", s, "new")
	}
}

func TestModTime(t *testing.T) {
	fs := NewFS()

	tBeforeWrite := time.Now()
	ioutil.WriteFile(fs, "/readme.txt", []byte{0, 0, 0}, 0666)
	fi, _ := fs.Stat("/readme.txt")
	mtimeAfterWrite := fi.ModTime()

	if !mtimeAfterWrite.After(tBeforeWrite) {
		t.Error("Open should modify mtime")
	}

	f, err := fs.OpenFile("/readme.txt", os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}
	f.Close()
	tAfterRead := fi.ModTime()

	if tAfterRead != mtimeAfterWrite {
		t.Error("Open with O_RDONLY should not modify mtime")
	}
}
