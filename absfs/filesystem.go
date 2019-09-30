package absfs

import (
	"errors"
	"os"
	"time"
)

var ErrNotImplemented = errors.New("not implemented")

type FileSystem interface {
	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flag int, perm os.FileMode) (File, error)

	// Mkdir creates a directory in the filesystem, return an error if any
	// happens.
	Mkdir(name string, perm os.FileMode) error

	// Remove removes a file identified by name, returning an error, if any
	// happens.
	Remove(name string) error

	// Rename renames (moves) oldpath to newpath. If newpath already exists and
	// is not a directory, Rename replaces it. OS-specific restrictions may apply
	// when oldpath and newpath are in different directories. If there is an
	// error, it will be of type *LinkError.
	Rename(oldpath, newpath string) error

	// Stat returns the FileInfo structure describing file. If there is an error,
	// it will be of type *PathError.
	Stat(name string) (os.FileInfo, error)

	//Chmod changes the mode of the named file to mode.
	Chmod(name string, mode os.FileMode) error

	//Chtimes changes the access and modification times of the named file
	Chtimes(name string, atime time.Time, mtime time.Time) error

	//Chown changes the owner and group ids of the named file
	Chown(name string, uid, gid int) error

	Separator() uint8
	ListSeparator() uint8
	Chdir(dir string) error
	Getwd() (dir string, err error)
	TempDir() string

	Open(name string) (File, error)
	Create(name string) (File, error)
	MkdirAll(name string, perm os.FileMode) error
	RemoveAll(path string) error
	Truncate(name string, size int64) error

	// Lstat returns a FileInfo describing the named file. If the file is a
	// symbolic link, the returned FileInfo describes the symbolic link. Lstat
	// makes no attempt to follow the link. If there is an error, it will be of type *PathError.
	Lstat(name string) (os.FileInfo, error)

	// Lchown changes the numeric uid and gid of the named file. If the file is a
	// symbolic link, it changes the uid and gid of the link itself. If there is
	// an error, it will be of type *PathError.
	//
	// On Windows, it always returns the syscall.EWINDOWS error, wrapped in
	// *PathError.
	Lchown(name string, uid, gid int) error

	// Readlink returns the destination of the named symbolic link. If there is an
	// error, it will be of type *PathError.
	Readlink(name string) (string, error)

	// Symlink creates newname as a symbolic link to oldname. If there is an
	// error, it will be of type *LinkError.
	Symlink(oldname, newname string) error
}
