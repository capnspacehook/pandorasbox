package pandorasbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSWalk(t *testing.T) {
	fs, err := NewOSFS()
	if err != nil {
		t.Fatal(err)
	}
	testpath := ".."
	abs, err := filepath.Abs(testpath)
	if err != nil {
		t.Fatal(err)
	}

	testpath = abs

	t.Run("Walk", func(t *testing.T) {
		list := make(map[string]bool)
		count := 0
		err = filepath.Walk(testpath, func(path string, info os.FileInfo, err error) error {
			p := strings.TrimPrefix(path, testpath)
			if p == "" {
				p = "/"
			}
			list[p] = true
			count++
			return nil
		})

		count2 := 0
		err = fs.Walk(testpath, func(path string, info os.FileInfo, err error) error {
			p := strings.TrimPrefix(path, testpath)
			if p == "" {
				p = "/"
			}
			if !list[p] {
				return fmt.Errorf("file not found %q", p)
			}
			delete(list, p)
			count2++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		if count < 10 || count != count2 {
			t.Errorf("incorrect file count: %d, %d", count, count2)
		}
		if len(list) > 0 {
			for k := range list {
				t.Errorf("path not removed %q", k)
			}
		}
	})
}

func TestOSFS(t *testing.T) {

	var ofs FileSystem

	t.Run("NewFs", func(t *testing.T) {
		fs, err := NewOSFS()
		if err != nil {
			t.Fatal(err)
		}

		ofs = fs
	})

	t.Run("Separators", func(t *testing.T) {
		if ofs.Separator() != filepath.Separator {
			t.Errorf("incorrect separator %q", ofs.Separator())
		}
		if ofs.ListSeparator() != filepath.ListSeparator {
			t.Errorf("incorrect list separator %q", ofs.ListSeparator())
		}
	})

	t.Run("Navigation", func(t *testing.T) {

		oswd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		fswd, err := ofs.Getwd()
		if fswd != oswd {
			t.Fatalf("incorrect working directory %q, %q", fswd, oswd)
		}

		cwd := "/"
		err = ofs.Chdir(cwd)
		if err != nil {
			t.Fatal(err)
		}

		fswd, err = ofs.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if fswd != cwd {
			t.Fatalf("incorrect working directory %q, %q", fswd, cwd)
		}

	})

	t.Run("TempDir", func(t *testing.T) {

		ostmp, fstmp := os.TempDir(), ofs.TempDir()
		if fstmp != ostmp {
			t.Fatalf("wrong TempDir output: %q != %q", fstmp, ostmp)
		}

	})

	// Move to the test directory for all further tests
	testdir, cleanup, err := FsTestDir(ofs, ofs.TempDir())
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	maxerrors := 10

	AutoTest(0, func(testcase *Testcase) error {
		result, err := FsTest(ofs, testdir, testcase)
		if err != nil {
			t.Fatal(err)
		}
		Errors := result.Errors

		for op, report := range testcase.Errors {
			if Errors[op] == nil {
				t.Logf("expected: \n%s\n", testcase.Report())
				t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q got nil but expected to get an err of type (%T)\n", testcase.TestNo, op, testcase.Errors[op].Type())
				continue
			}
			if report.Err == nil {
				if Errors[op].Err == nil {
					continue
				}

				t.Logf("expected: \n%s\n", testcase.Report())
				t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q expected `err == nil` but got err: (%T) %q\n%s", testcase.TestNo, op, Errors[op].Type(), Errors[op].String(), Errors[op].Stack())
				maxerrors--
				continue
			}

			if Errors[op].Err == nil {
				t.Logf("expected: \n%s\n", testcase.Report())
				t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q got `err == nil` but expected err: (%T) %q\n%s", testcase.TestNo, op, testcase.Errors[op].Type(), testcase.Errors[op].String(), Errors[op].Stack())
				maxerrors--
			}
			if !report.TypesEqual(Errors[op]) {
				t.Logf("expected: \n%s\n", testcase.Report())
				t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q got different error types, expected (%T) but got (%T)\n%s", testcase.TestNo, op, testcase.Errors[op].Type(), Errors[op].Type(), Errors[op].Stack())
				maxerrors--
			}
			if !report.Equal(Errors[op]) {
				t.Logf("expected: \n%s\n", testcase.Report())
				t.Logf("  result: \n%s\n", result.Report())
				t.Fatalf("%d: On %q got different error values, expected %q but got %q\n%s", testcase.TestNo, op, testcase.Errors[op], Errors[op].String(), Errors[op].Stack())
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
