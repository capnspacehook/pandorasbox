package memfs

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
)

func TestWalk(t *testing.T) {
	fs, err := NewFS()
	if err != nil {
		t.Fatal(err)
	}
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

func TestMemFS(t *testing.T) {
	fs, err := NewFS()
	if err != nil {
		t.Fatal(err)
	}

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

	// t.Logf("Test path: %q", testdir)
	err = fs.MkdirAll(testdir, 0777)
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
				// t.Logf("expected: \n%s\n", testcase.Report())
				// t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q got nil but expected to get an err of type (%T)\n", testcase.TestNo, op, testcase.Errors[op].Type())
				continue
			}
			if report.Err == nil {
				if Errors[op].Err == nil {
					continue
				}

				// t.Logf("expected: \n%s\n", testcase.Report())
				// t.Logf("  result: \n%s\n", result.Report())
				// t.Logf("  flags: (%d)%s, (%d)%s", result.Flags, absfs.Flags(result.Flags), testcase.Flags, absfs.Flags(testcase.Flags))
				// t.Logf("  perm: %s, %s", result.Mode, testcase.Mode)
				t.Fatalf("%d: On %q expected `err == nil` but got err: (%T) %q\n%s", testcase.TestNo, op, Errors[op].Type(), Errors[op].String(), Errors[op].Stack())
				maxerrors--
				continue
			}

			if Errors[op].Err == nil {
				// t.Logf("expected: \n%s\n", testcase.Report())
				// t.Logf("  result: \n%s\n", result.Report())
				// t.Logf("  flags: (%d)%s, (%d)%s", result.Flags, absfs.Flags(result.Flags), testcase.Flags, absfs.Flags(testcase.Flags))
				// t.Logf("  perm: %s, %s", result.Mode, testcase.Mode)
				t.Errorf("%d: On %q got `err == nil` but expected err: (%T) %q\n%s", testcase.TestNo, op, testcase.Errors[op].Type(), testcase.Errors[op].String(), Errors[op].Stack())
				maxerrors--
			}
			if !report.TypesEqual(Errors[op]) {
				// t.Logf("expected: \n%s\n", testcase.Report())
				// t.Logf("  result: \n%s\n", result.Report())
				// t.Logf("%q %q", report.Error(), Errors[op].Error())
				// t.Logf("  flags: (%d)%s, (%d)%s", result.Flags, absfs.Flags(result.Flags), testcase.Flags, absfs.Flags(testcase.Flags))
				// t.Logf("  perm: %s, %s", result.Mode, testcase.Mode)
				t.Errorf("%d: On %q got different error types, expected (%T) but got (%T)\n", testcase.TestNo, op, report.Type(), Errors[op].Type())
				maxerrors--
			}
			if report.Error() != Errors[op].Error() { //report.Equal(Errors[op]) {
				// t.Logf("expected: \n%s\n", testcase.Report())
				// t.Logf("  result: \n%s\n", result.Report())

				// t.Logf("  flags: (%d)%s, (%d)%s", result.Flags, absfs.Flags(result.Flags), testcase.Flags, absfs.Flags(testcase.Flags))
				// t.Logf("  perm: %s, %s", result.Mode, testcase.Mode)
				t.Errorf("%d: On %q got different error values,\nexpecte, got:\n%q\n%q\n%s", testcase.TestNo, op, report.Error(), Errors[op].Error(), Errors[op].Stack())
				// t.Fatalf("report.Error() != Errors[op].Error()\n%s\n%s\n", report.Error(), Errors[op].Error())
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
	fs, err := NewFS()
	if err != nil {
		t.Fatal(err)
	}

	if fs.TempDir() != "/tmp" {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), "/tmp")
	}

	fs.Tempdir = os.TempDir()
	if fs.TempDir() != os.TempDir() {
		t.Fatalf("wrong TempDir output: %q != %q", fs.TempDir(), os.TempDir())
	}

	testdir := fs.TempDir()

	t.Logf("Test path: %q", testdir)
	err = fs.MkdirAll(testdir, 0777)
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
	fs, err := NewFS()
	if err != nil {
		t.Fatal(err)
	}

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
