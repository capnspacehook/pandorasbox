package pandorasbox

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

type OSFileSystem struct {
	cwd string
}

func NewOSFS() (*OSFileSystem, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil
	}

	return &OSFileSystem{dir}, nil
}

func (fs *OSFileSystem) Separator() uint8 {
	return filepath.Separator
}

func (fs *OSFileSystem) ListSeparator() uint8 {
	return filepath.ListSeparator
}

func (fs *OSFileSystem) isDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func (fs *OSFileSystem) fixPath(name string) string {
	if !filepath.IsAbs(name) {
		name = filepath.Join(fs.cwd, name)
	}
	return name
}

func (fs *OSFileSystem) Chdir(name string) error {
	name = fs.fixPath(name)
	if !fs.isDir(name) {
		return &os.PathError{Op: "chdir", Path: name, Err: errors.New("not a directory")}
	}
	fs.cwd = name
	return nil
}

func (fs *OSFileSystem) Getwd() (dir string, err error) {
	return fs.cwd, nil
}

func (fs *OSFileSystem) TempDir() string {
	return os.TempDir()
}

func (fs *OSFileSystem) Open(name string) (File, error) {
	f, err := os.Open(fs.fixPath(name))
	if err != nil {
		return nil, err
	}

	return &OSFile{fs, f}, nil
}

func (fs *OSFileSystem) Create(name string) (File, error) {
	f, err := os.Create(fs.fixPath(name))
	if err != nil {
		return nil, err
	}

	return &OSFile{fs, f}, nil
}

// func (fs *OSFileSystem) MkdirAll(name string, perm os.FileMode) error {
// 	return os.MkdirAll(fs.fixPath(name), perm)
// }

// func (fs *OSFileSystem) RemoveAll(name string) (err error) {
// 	return os.RemoveAll(fs.fixPath(name))
// }

func (fs *OSFileSystem) Truncate(name string, size int64) error {
	return os.Truncate(fs.fixPath(name), size)
}

func (fs *OSFileSystem) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(fs.fixPath(name), perm)
}

func (fs *OSFileSystem) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(fs.fixPath(name), perm)
}

func (fs *OSFileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	f, err := os.OpenFile(fs.fixPath(name), flag, perm)
	if err != nil {
		return nil, err
	}

	return File(&OSFile{fs, f}), err
}

// func (fs *OSFileSystem) Lstat(name string) (os.FileInfo, error) {
// 	return os.Lstat(fs.fixPath(name))
// }

func (fs *OSFileSystem) Remove(name string) error {
	return os.Remove(fs.fixPath(name))
}

func (fs *OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(fs.fixPath(oldpath), fs.fixPath(newpath))
}

func (fs *OSFileSystem) RemoveAll(name string) error {
	return os.RemoveAll(fs.fixPath(name))
}

func (fs *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(fs.fixPath(name))
}

//Chmod changes the mode of the named file to mode.
func (fs *OSFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(fs.fixPath(name), mode)
}

//Chtimes changes the access and modification times of the named file
func (fs *OSFileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(fs.fixPath(name), atime, mtime)
}

//Chown changes the owner and group ids of the named file
func (fs *OSFileSystem) Chown(name string, uid, gid int) error {
	return os.Chown(fs.fixPath(name), uid, gid)
}

func (fs *OSFileSystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(fs.fixPath(name))
}

// ess

func (fs *OSFileSystem) Lchown(name string, uid, gid int) error {
	return os.Lchown(fs.fixPath(name), uid, gid)
}

func (fs *OSFileSystem) Readlink(name string) (string, error) {
	return os.Readlink(fs.fixPath(name))
}

func (fs *OSFileSystem) Symlink(oldname, newname string) error {
	return os.Symlink(fs.fixPath(oldname), fs.fixPath(newname))
}

func (fs *OSFileSystem) Walk(path string, fn func(string, os.FileInfo, error) error) error {
	return filepath.Walk(path, fn) //(filepath.WalkFunc)(fn))
}
