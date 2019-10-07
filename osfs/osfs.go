package osfs

import (
	"os"
	"path/filepath"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
)

func IsPathSeparator(c uint8) bool {
	return os.IsPathSeparator(c)
}

type FileSystem struct {
}

func NewFS() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) Separator() uint8 {
	return filepath.Separator
}

func (fs *FileSystem) ListSeparator() uint8 {
	return filepath.ListSeparator
}

func (fs *FileSystem) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (fs *FileSystem) Chdir(name string) error {
	return os.Chdir(name)
}

func (fs *FileSystem) Getwd() (dir string, err error) {
	return os.Getwd()
}

func (fs *FileSystem) TempDir() string {
	return os.TempDir()
}

func (fs *FileSystem) Open(name string) (absfs.File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return &File{fs, f}, nil
}

func (fs *FileSystem) Create(name string) (absfs.File, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return &File{fs, f}, nil
}

func (fs *FileSystem) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
}

func (fs *FileSystem) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (fs *FileSystem) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}

func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	return &File{fs, f}, err
}

func (fs *FileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (fs *FileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (fs *FileSystem) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *FileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (fs *FileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}

func (fs *FileSystem) Chown(name string, uid, gid int) error {
	return os.Chown(name, uid, gid)
}

func (fs *FileSystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

// ess

func (fs *FileSystem) Lchown(name string, uid, gid int) error {
	return os.Lchown(name, uid, gid)
}

func (fs *FileSystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (fs *FileSystem) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (fs *FileSystem) Walk(path string, fn func(string, os.FileInfo, error) error) error {
	return filepath.Walk(path, fn)
}
