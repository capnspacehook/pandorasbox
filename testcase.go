package pandorasbox

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
)

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var magenta = color.New(color.FgMagenta).SprintFunc()

type ErrorReport struct {
	Op       string
	Path     string
	Err      error
	StackStr string
	TypeStr  string

	ErrStr string
}

func (e *ErrorReport) Type() string {
	return e.TypeStr
}

func (e *ErrorReport) Error() string {
	return e.ErrStr
}

func (e *ErrorReport) Stack() string {
	return red(e.StackStr)
}

func (e *ErrorReport) String() string {

	str := e.Error()
	start := strings.Index(str, "/")
	if start < 0 {
		return str
	}
	end := strings.LastIndex(str, "/") + 1

	if end == len(str)-1 {
		return str[:start]
	}
	return str[:start] + str[end:]
}

func (e *ErrorReport) TypesEqual(r *ErrorReport) bool {
	reply := e.Type() == r.Type()
	if !reply {
		fmt.Printf("TypesEqual: %q, %q", e.Type(), r.Type())
	}
	return reply
}

func (e *ErrorReport) Equal(r *ErrorReport) bool {
	return e.String() == r.String()
}

func NewErrorReport(op, path string, err error, stackstr string) *ErrorReport {
	_, ok := err.(*ErrorString)
	if ok {
		panic("how?")
	}
	typestr := fmt.Sprintf("%T", err)

	err = errorStringConvert(err)
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	return &ErrorReport{
		Op:       op,
		Path:     path,
		Err:      err,
		StackStr: stackstr,
		TypeStr:  typestr,
		ErrStr:   errstr,
	}
}

type ErrorString struct {
	Err string
}

func (e *ErrorString) Error() string {
	return e.Err
}

func errorStringConvert(err error) error {
	if err == nil {
		return nil
	}
	buf := new(bytes.Buffer)
	if gob.NewEncoder(buf).Encode(err) == nil {
		return err
	}
	return &ErrorString{err.Error()}
}

type Testcase struct {
	TestNo       int         `json:"test_no"`
	PreCondition string      `json:"pre_condition"`
	Op           string      `json:"op"`
	Path         string      `json:"path"`
	Flags        int         `json:"flags"`
	Mode         os.FileMode `json:"mode"`

	Errors map[string]*ErrorReport
}

func (t *Testcase) Report() string {
	buff := new(bytes.Buffer)
	data, _ := json.Marshal(t)
	json.Indent(buff, data, "\t", "  ")
	return fmt.Sprintf("%s\n", string(buff.Bytes()))
}

func init() {
	var errno syscall.Errno
	patherr := new(os.PathError)
	errorerrorstring := errors.New("error")
	errorstring := new(ErrorString)
	gob.Register(patherr)
	gob.Register(errno)
	gob.Register(errorerrorstring) // apparently this won't work
	gob.Register(errorstring)

}

func testDir() (testdir string, cleanup func(), err error) {

	// assign noop to cleanup until there is something to clean up.
	cleanup = func() {}
	timestamp := time.Now().Format(time.RFC3339)
	testdir = filepath.Join(os.TempDir(), fmt.Sprintf("fstesting%s", timestamp))

	err = os.Mkdir(testdir, 0777)
	if err != nil {
		panic(err.Error())
		return testdir, cleanup, err
	}

	// capture the current working directory
	var startingDir string
	startingDir, err = os.Getwd()
	if err != nil {
		return testdir, cleanup, err
	}

	cleanup = func() {
		os.Chdir(startingDir)
		err := os.RemoveAll(testdir)
		if err != nil {
			panic(err)
		}
	}

	err = os.Chdir(testdir)
	return testdir, cleanup, err
}

// FsTestDir Creates a timestamped folder for filesystem testing, and changes directory
// to it.
// Returns the path to the new directory, a cleanup function and an error.
// The `cleanup` method changes the directory back to the original location
// and removes testdir and all of it's contents.
func FsTestDir(fs FileSystem, path string) (testdir string, cleanup func(), err error) {

	timestamp := time.Now().Format(time.RFC3339)
	testdir = filepath.Join(path, fmt.Sprintf("FsTestDir%s", timestamp))
	var cwd string
	cwd, err = fs.Getwd()
	if err != nil {
		panic(err)
		return testdir, cleanup, err
	}
	cleanup = func() {
		fs.Chdir(cwd)
		err := fs.RemoveAll(testdir)
		if err != nil {
			panic(err)
		}
	}

	for _, path := range []string{path, testdir} {
		fmt.Printf("Mkdir(%q)\n", path)
		err := fs.Mkdir(path, 0777)
		if os.IsExist(err) {
			continue
		}
	}

	err = fs.Chdir(testdir)
	if err != nil {
		return testdir, cleanup, err
	}

	return testdir, cleanup, nil
}

// GenerateTestcases runs all tests on the `os` package to establish baseline
// results that can be used to test that `absfs` FileSystems are consistent with
// native file system support.
// If not `nil` GenerateTestcases will call `fn` with each generated testcase. If
// `fn` returns an error then testcase generation will stop and GenerateTestcases
// will return an the same error and the testcases crated so far.
// (TODO: many tests still to be added to exercise the entire FileSystem interface)
func AutoTest(startno int, fn func(*Testcase) error) error {
	testdir, cleanup, err := testDir()
	defer cleanup()
	if err != nil {
		return err
	}

	// Various OpenFile pre-conditions
	preconditions := []string{
		"notcreated",  // No file exists for filename.
		"created",     // A file with normal permissions exists for filename
		"dir",         // A directory with normal permissions exists for filename
		"permissions", // A file with no permissions exists for filename
	}
	testNo := 0
	if testdir == "" {
		return errors.New("testdir undefined")
	}

	// define noop function if needed
	if fn == nil {
		fn = func(*Testcase) error {
			return nil
		}
	}
	return ForEveryFlag(func(flag int) error {
		return ForEveryPermission(func(mode os.FileMode) error {
			for _, pathPrefix := range []string{testdir, ".", ""} {
				for _, condition := range preconditions {
					if testNo < startno {
						testNo++
						continue
					}

					name := filepath.Join(pathPrefix, fmt.Sprintf("fstestingFile%08d", testNo))
					switch condition {
					case "notcreated":
					case "created":
						info, err := os.Stat(name)
						if !os.IsNotExist(err) {
							return fmt.Errorf("file exists unexpectedly %s %q", info.Mode(), name)
						}
						f, err := os.Create(name)
						if err != nil {
							return fmt.Errorf("unable to create %q + %q, %s", testdir, name, err)
						} else {
							_, err = f.WriteString("Hello, world!\n")
							if err != nil {
								return err
							}
							f.Close()
						}

					case "dir":
						name = filepath.Join(pathPrefix, fmt.Sprintf("fstestingDir%08d", testNo))
						err = os.Mkdir(name, 0777)
						if err != nil {
							return err
						}

					case "permissions":
						f, err := os.Create(name)
						if err != nil {
							return err
						}
						_, err = f.WriteString("Hello, world!\n")
						if err != nil {
							return err
						}
						f.Close()
						err = os.Chmod(name, 0)
						if err != nil {
							return err
						}
					}
					Errors := make(map[string]*ErrorReport)

					// Tests

					// OpenFile test
					f, err := os.OpenFile(name, flag, os.FileMode(mode))
					Errors["OpenFile"] = NewErrorReport("OpenFile", name, err, fmt.Sprintf("%+v", err))
					if f != nil {
						// Name test
						fname := f.Name()
						Errors["Name"] = NewErrorReport("Name", fname, nil, "")

						// Write test
						writedata := []byte("The quick brown fox, jumped over the lazy dog!")
						n, err := f.Write(writedata)
						Errors["Write"] = NewErrorReport("Write", name, err, fmt.Sprintf("%+v", err))
						_ = n
						// TODO: check if n == len(writedata)

						// Read test
						f.Seek(0, io.SeekStart)
						readdata := make([]byte, 512)
						n, err = f.Read(readdata)
						Errors["Read"] = NewErrorReport("Read", name, err, fmt.Sprintf("%+v", err))
						readdata = readdata[:n]
						_ = readdata

						// Close test
						err = f.Close()
						Errors["Close"] = NewErrorReport("Close", name, err, fmt.Sprintf("%+v", err))
					}

					testcase := &Testcase{
						TestNo:       testNo,
						PreCondition: condition,
						Op:           "openfile",
						Path:         name,
						Flags:        flag,
						Mode:         os.FileMode(mode),
						Errors:       Errors,
					}

					err = fn(testcase)
					if err != nil {
						return err
					}

					testNo++
				}
			}
			return nil
		})
	})
}

func FsTest(fs FileSystem, path string, testcase *Testcase) (*Testcase, error) {
	// defer fmt.Fprintf(os.Stderr, "FsTest %s\n", blue(path))
	name, err := pretest(fs, path, testcase)
	if err != nil {
		return nil, err
	}

	newtestcase, err := test(fs, testcase.TestNo, name, testcase.Flags, testcase.Mode, testcase.PreCondition)
	posttest(fs, newtestcase)
	return newtestcase, err
}

func createFile(fs FileSystem, name string) error {
	info, err := fs.Stat(name)
	if !os.IsNotExist(err) {
		return fmt.Errorf("file exists unexpectedly %s %q", info.Mode(), name)
	}
	f, err := fs.Create(name)
	if err != nil {
		return fmt.Errorf("unable to create  %q, %s", name, err)
	}
	defer f.Close()

	_, err = f.WriteString("Hello, world!\n")
	return err
}

func pretest(fs FileSystem, path string, testcase *Testcase) (string, error) {
	name := filepath.Join(path, fmt.Sprintf("fstestingFile%08d", testcase.TestNo))
	switch testcase.PreCondition {
	case "":
		fallthrough
	case "notcreated":

	case "created":
		err := createFile(fs, name)
		if err != nil {
			return "", err
		}

	case "dir":
		name = filepath.Join(path, fmt.Sprintf("fstestingDir%08d", testcase.TestNo))
		err := fs.Mkdir(name, 0777)
		if err != nil {
			return name, err
		}

	case "permissions":
		err := createFile(fs, name)
		if err != nil {
			return name, err
		}
		err = fs.Chmod(name, 0)
		if err != nil {
			return name, err
		}
	}

	return name, nil
}

func posttest(fs FileSystem, testcase *Testcase) error {

	return nil
}

func test(fs FileSystem, testNo int, name string, flags int, mode os.FileMode, precondition string) (*Testcase, error) {
	Errors := make(map[string]*ErrorReport)

	// OpenFile test
	f, err := fs.OpenFile(name, flags, mode)
	Errors["OpenFile"] = NewErrorReport("OpenFile", name, err, fmt.Sprintf("%+v", err))
	var n int

	if f != nil {
		// Name test
		fname := f.Name()
		Errors["Name"] = NewErrorReport("Name", fname, nil, "")

		// Write test
		writedata := []byte("The quick brown fox, jumped over the lazy dog!")
		n, err = f.Write(writedata)
		Errors["Write"] = NewErrorReport("Write", name, err, fmt.Sprintf("%+v", err))
		_ = n
		// TODO: check if n == len(writedata)

		// Read test
		f.Seek(0, io.SeekStart)
		readdata := make([]byte, 512)
		n, err = f.Read(readdata)
		Errors["Read"] = NewErrorReport("Read", name, err, fmt.Sprintf("%+v", err))
		readdata = readdata[:n]
		_ = readdata

		// Close test
		err = f.Close()
		Errors["Close"] = NewErrorReport("Close", name, err, fmt.Sprintf("%+v", err))
	}

	// Create a new test case with above values.
	testcase := &Testcase{
		TestNo:       testNo,
		PreCondition: precondition,
		Op:           "openfile",
		Path:         name,
		Flags:        flags,
		Mode:         os.FileMode(mode),
		Errors:       Errors,
	}

	return testcase, nil
}

func CompareErrors(err1 error, err2 error) error {

	switch v1 := err1.(type) {

	case nil:
		if err2 == nil {
			return nil
		}
		return fmt.Errorf("err1 is <nil> err2 is %s", err2)

	case *os.PathError:
		v2, ok := err2.(*os.PathError)
		if !ok {
			return fmt.Errorf("errors differ in type %T != %T", err1, err2)
		}

		var list []string

		if path.Base(v1.Path) != path.Base(v2.Path) {
			list = append(list, fmt.Sprintf("paths not equal %q != %q", v1.Path, v2.Path))
		}

		if v1.Op != v2.Op {
			list = append(list, fmt.Sprintf("ops not equal %q != %q", v1.Op, v2.Op))
		}

		if v1.Err.Error() != v2.Err.Error() {
			list = append(list, fmt.Sprintf("errors not equal %q != %q", v1.Err.Error(), v2.Err.Error()))
		}

		if len(list) == 0 {
			return nil
		}

		return fmt.Errorf("os.PathErrors:  %s", strings.Join(list, "; "))

	case *ErrorString:
		v2, ok := err2.(*ErrorString)
		if !ok {
			return fmt.Errorf("errors differ in type %T != %T", err1, err2)
		}
		_ = v2

	default:
		panic(fmt.Sprintf("un-handled types %T, %T", err1, err2))
	}

	if err1.Error() != err2.Error() {
		return fmt.Errorf("unknown unequal errors %T & %T differ %q != %q", err1, err2, err1, err2)
	}

	return nil // fmt.Errorf("unknown matching errors %T, %q", err1, err2)
}
