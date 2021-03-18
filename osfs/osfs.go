package osfs

import (
	"os"
	"path/filepath"

	"github.com/capnspacehook/pandorasbox/absfs"
)

func IsPathSeparator(c uint8) bool {
	return os.IsPathSeparator(c)
}

type FileSystem struct{}

func NewFS() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) Open(name string) (absfs.File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return &File{f: f}, nil
}

func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	return &File{f: f}, err
}

func (fs *FileSystem) Create(name string) (absfs.File, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return &File{f: f}, nil
}

func (fs *FileSystem) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (fs *FileSystem) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}

func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *FileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (fs *FileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (fs *FileSystem) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

func (fs *FileSystem) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
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
