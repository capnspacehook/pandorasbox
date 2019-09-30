package absfs

import (
	"syscall"

	"os"
)

type File interface {
	// Name returns the name of the file as presented to Open.
	Name() string

	// Read reads up to len(b) bytes from the File. It returns the number of bytes
	// read and any error encountered. At end of file, Read returns 0, io.EOF.
	Read(p []byte) (int, error)

	// Write writes len(b) bytes to the File. It returns the number of bytes
	// written and an error, if any. Write returns a non-nil error when
	// n != len(b).
	Write(p []byte) (int, error)

	// Close closes the File, rendering it unusable for I/O. It returns an error,
	// if any.
	Close() error

	// Sync commits the current contents of the file to stable storage. Typically,
	// this means flushing the file system's in-memory copy of recently written
	// data to disk.
	Sync() error

	// Stat returns the FileInfo structure describing file. If there is an error,
	// it will be of type *PathError.
	Stat() (os.FileInfo, error)

	// Readdir reads the contents of the directory associated with file and
	// returns a slice of up to n FileInfo values, as would be returned by Lstat,
	// in directory order. Subsequent calls on the same file will yield further
	// FileInfos.

	// If n > 0, Readdir returns at most n FileInfo structures. In this case, if
	// Readdir returns an empty slice, it will return a non-nil error explaining
	// why. At the end of a directory, the error is io.EOF.

	// If n <= 0, Readdir returns all the FileInfo from the directory in a single
	// slice. In this case, if Readdir succeeds (reads all the way to the end of
	// the directory), it returns the slice and a nil error. If it encounters an
	// error before the end of the directory, Readdir returns the FileInfo read
	// until that point and a non-nil error.
	Readdir(int) ([]os.FileInfo, error)

	// Seek sets the offset for the next Read or Write on file to offset,
	// interpreted according to whence: 0 means relative to the origin of the
	// file, 1 means relative to the current offset, and 2 means relative to the
	// end. It returns the new offset and an error, if any. The behavior of Seek
	// on a file opened with O_APPEND is not specified.
	Seek(offset int64, whence int) (ret int64, err error)

	// ReadAt reads len(b) bytes from the File starting at byte offset off. It
	// returns the number of bytes read and the error, if any. ReadAt always
	// returns a non-nil error when n < len(b). At end of file, that error is
	// io.EOF.
	ReadAt(b []byte, off int64) (n int, err error)

	// WriteAt writes len(b) bytes to the File starting at byte offset off. It
	// returns the number of bytes written and an error, if any. WriteAt returns
	// a non-nil error when n != len(b).
	WriteAt(b []byte, off int64) (n int, err error)

	// WriteString is like Write, but writes the contents of string s rather than
	// a slice of bytes.
	WriteString(s string) (n int, err error)

	// Truncate changes the size of the file. It does not change the I/O offset.
	// If there is an error, it will be of type *PathError.
	Truncate(size int64) error

	// If n > 0, Readdirnames returns at most n names. In this case, if
	// Readdirnames returns an empty slice, it will return a non-nil error
	// explaining why. At the end of a directory, the error is io.EOF.

	// If n <= 0, Readdirnames returns all the names from the directory in a single
	// slice. In this case, if Readdirnames succeeds (reads all the way to the end of
	// the directory), it returns the slice and a nil error. If it encounters an
	// error before the end of the directory, Readdirnames returns the names read
	// until that point and a non-nil error.
	Readdirnames(n int) (names []string, err error)
}

// InvalidFile is a no-op implementation of File that can be returned from any
// file open methods when an error occurs. InvalidFile mimics the behavior of
// file handles returnd by the `os` package when there is an error.
type InvalidFile struct {

	// Path should be the filepath provided to the `Open` method.
	Path string
}

func (f *InvalidFile) Name() string {
	return f.Path
}

func (f *InvalidFile) Read(p []byte) (int, error) {
	return 0, &os.PathError{Op: "read", Path: f.Name(), Err: syscall.EBADF}
	// return 0, io.EOF
}

func (f *InvalidFile) Write(p []byte) (int, error) {
	return 0, &os.PathError{Op: "write", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Close() error {
	return nil
}

func (f *InvalidFile) Sync() error {
	return &os.PathError{Op: "sync", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Stat() (os.FileInfo, error) {
	return nil, &os.PathError{Op: "stat", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, &os.PathError{Op: "readdir", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Seek(offset int64, whence int) (ret int64, err error) {
	return 0, &os.PathError{Op: "seek", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, &os.PathError{Op: "read", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, &os.PathError{Op: "write", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) WriteString(s string) (n int, err error) {
	return 0, &os.PathError{Op: "write", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Truncate(size int64) error {
	return &os.PathError{Op: "truncate", Path: f.Name(), Err: syscall.EBADF}
}

func (f *InvalidFile) Readdirnames(n int) (names []string, err error) {
	return nil, &os.PathError{Op: "readdirnames", Path: f.Name(), Err: syscall.EBADF}
}
