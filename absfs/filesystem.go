package absfs

import (
	"io/fs"
)

type FileSystem interface {
	FS() fs.FS

	// Open opens the named file for reading. If successful, methods on the returned
	// file can be used for reading; the associated file descriptor has mode O_RDONLY.
	// If there is an error, it will be of type *fs.PathError.
	Open(name string) (File, error)

	// OpenFile is the generalized open call; most users will use Open or
	// Create instead. It opens the named file with specified flag (O_RDONLY etc.).
	// If the file does not exist, and the O_CREATE flag is passed, it is created
	// with mode perm (before umask). If successful, methods on the returned File
	// can be used for I/O. If there is an error, it will be of type *fs.PathError.
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)

	// Create creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created with mode 0666
	// (before umask). If successful, methods on the returned File can be used for
	// I/O; the associated file descriptor has mode os.O_RDWR. If there is an error,
	// it will be of type *fs.PathError.
	Create(name string) (File, error)

	// ReadFile reads the named file and returns its contents.
	// A successful call returns a nil error, not io.EOF.
	// (Because ReadFile reads the whole file, the expected EOF
	// from the final Read is not treated as an error to be reported.)
	ReadFile(name string) ([]byte, error)

	// ReadDir reads the named directory
	// and returns a list of directory entries sorted by filename.
	ReadDir(name string) ([]fs.DirEntry, error)

	// WriteFile writes data to the named file, creating it if necessary. If the
	// file does not exist, WriteFile creates it with permissions perm (before umask);
	// otherwise WriteFile truncates it before writing, without changing permissions.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// Mkdir creates a new directory with the specified name and permission bits
	// (before umask). If there is an error, it will be of type *fs.PathError.
	Mkdir(name string, perm fs.FileMode) error

	// MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error. The permission bits perm (before umask)
	// are used for all directories that MkdirAll creates. If path is already a
	// directory, MkdirAll does nothing and returns nil.
	MkdirAll(name string, perm fs.FileMode) error

	// Stat returns the FileInfo structure describing file. If there is an error,
	// it will be of type *fs.PathError.
	Stat(name string) (fs.FileInfo, error)

	// Lstat returns a FileInfo describing the named file.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow the link.
	// If there is an error, it will be of type *PathError.
	Lstat(name string) (fs.FileInfo, error)

	// Rename renames (moves) oldpath to newpath. If newpath already exists and
	// is not a directory, Rename replaces it. OS-specific restrictions may apply
	// when oldpath and newpath are in different directories. If there is an
	// error, it will be of type *os.LinkError.
	Rename(oldpath, newpath string) error

	// Remove removes the named file or (empty) directory. If there is an error,
	// it will be of type *fs.PathError.
	Remove(name string) error

	// RemoveAll removes path and any children it contains. It removes everything
	// it can but returns the first error it encounters. If the path does not exist,
	// RemoveAll returns nil (no error). If there is an error, it will be of type
	// *fs.PathError.
	RemoveAll(path string) error

	// Truncate changes the size of the named file. If the file is a symbolic link,
	// it changes the size of the link's target. If there is an error, it will be
	// of type *fs.PathError.
	Truncate(name string, size int64) error

	// WalkDir walks the file tree rooted at root, calling fn for each file or directory
	// in the tree, including root. All errors that arise visiting files and directories
	// are filtered by fn: see the fs.WalkDirFunc documentation for details. The files may
	// or may not be walked in lexical order.
	WalkDir(root string, fn fs.WalkDirFunc) error

	// TODO: add docs
	Abs(path string) (string, error)
	Separator() uint8
	ListSeparator() uint8
	// IsPathSeparator
	Chdir(dir string) error
	Getwd() (dir string, err error)
	TempDir() string

	// TODO: add all *Temp functions
}
