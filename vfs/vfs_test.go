package vfs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// Create file with unkown parent
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
		if name := f.Name(); name != "/relFile" {
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

	//TODO: Subdir of file
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
	var params = []struct {
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
		err = f.Truncate(newSize)
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
	if err := fs.Rename("/newdirectory/README.txt", "/README.txt"); err == nil {
		t.Errorf("Expected error renaming file")
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
