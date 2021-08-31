package osfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/capnspacehook/pandorasbox/absfs"
)

type stdFS struct {
	pbFS
}

func (stdFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (stdFS) Sub(dir string) (fs.FS, error) {
	return os.DirFS(dir), nil
}

type pbFS struct{}

func NewFS() absfs.FileSystem {
	return pbFS{}
}

func (pbFS) FS() fs.FS {
	return stdFS{}
}

func (pbFS) Open(name string) (absfs.File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (pbFS) OpenFile(name string, flag int, perm fs.FileMode) (absfs.File, error) {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	return f, err
}

func (pbFS) Create(name string) (absfs.File, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (pbFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (pbFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (pbFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (pbFS) Mkdir(name string, perm fs.FileMode) error {
	return os.Mkdir(name, perm)
}

func (pbFS) MkdirAll(name string, perm fs.FileMode) error {
	return os.MkdirAll(name, perm)
}

func (pbFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (pbFS) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

func (pbFS) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (pbFS) Remove(name string) error {
	return os.Remove(name)
}

func (pbFS) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

func (pbFS) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
}

func (pbFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (pbFS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (pbFS) Separator() uint8 {
	return filepath.Separator
}

func (pbFS) ListSeparator() uint8 {
	return filepath.ListSeparator
}

func (pbFS) Chdir(name string) error {
	return os.Chdir(name)
}

func (pbFS) Getwd() (dir string, err error) {
	return os.Getwd()
}

func (pbFS) TempDir() string {
	return os.TempDir()
}
