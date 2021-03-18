package pandorasbox

import (
	"errors"
	"io"
	"os"

	"github.com/awnumar/memguard"
	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
	"github.com/capnspacehook/pandorasbox/osfs"
	"github.com/capnspacehook/pandorasbox/vfs"
)

type Box struct {
	osfs *osfs.FileSystem
	vfs  *vfs.FileSystem
}

func NewBox() *Box {
	box := new(Box)
	box.osfs = osfs.NewFS()
	box.vfs = vfs.NewFS()

	return box
}

func (b *Box) Open(name string) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Open(vfsName)
	}

	return b.osfs.Open(name)
}

func (b *Box) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.OpenFile(vfsName, flag, perm)
	}

	return b.osfs.OpenFile(name, flag, perm)
}

func (b *Box) Create(name string) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Create(vfsName)
	}

	return b.osfs.Create(name)
}

func (b *Box) Mkdir(name string, perm os.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Mkdir(vfsName, perm)
	}

	return b.osfs.Mkdir(name, perm)
}

func (b *Box) MkdirAll(name string, perm os.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.MkdirAll(vfsName, perm)
	}

	return b.osfs.MkdirAll(name, perm)
}

func (b *Box) Stat(name string) (os.FileInfo, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Stat(vfsName)
	}

	return b.osfs.Stat(name)
}

func (b *Box) Rename(oldpath, newpath string) error {
	vfsOldPath, oldPathVFS := ConvertVFSPath(oldpath)
	vfsNewPath, newPathVFS := ConvertVFSPath(newpath)
	if oldPathVFS && newPathVFS {
		return b.vfs.Rename(vfsOldPath, vfsNewPath)
	} else if (oldPathVFS && !newPathVFS) || (!oldPathVFS && newPathVFS) {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: errors.New("oldpath and newpath must both either be a VFS path, or normal path")}
	}

	return b.osfs.Rename(oldpath, newpath)
}

func (b *Box) Remove(name string) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Remove(vfsName)
	}

	return b.osfs.Remove(name)
}

func (b *Box) RemoveAll(path string) error {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		return b.vfs.RemoveAll(vfsPath)
	}

	return b.osfs.RemoveAll(path)
}

func (b *Box) Truncate(name string, size int64) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Truncate(vfsName, size)
	}

	return b.osfs.Truncate(name, size)
}

func (b *Box) Separator(vfsMode bool) uint8 {
	if vfsMode {
		return b.vfs.Separator()
	}

	return b.osfs.Separator()
}

func (b *Box) ListSeparator(vfsMode bool) uint8 {
	if vfsMode {
		return b.vfs.ListSeparator()
	}

	return b.osfs.ListSeparator()
}

func (b *Box) IsPathSeparator(c uint8, vfsMode bool) bool {
	if vfsMode {
		return vfs.IsPathSeparator(c)
	}

	return osfs.IsPathSeparator(c)
}

func (b *Box) Chdir(dir string, vfsMode bool) error {
	if vfsMode {
		return b.vfs.Chdir(dir)
	}

	return b.osfs.Chdir(dir)
}

func (b *Box) Getwd(vfsMode bool) (string, error) {
	if vfsMode {
		return b.vfs.Getwd()
	}

	return b.osfs.Getwd()
}

func (b *Box) GetTempDir(vfsMode bool) string {
	if vfsMode {
		return b.vfs.TempDir()
	}

	return b.osfs.TempDir()
}

// io/ioutil methods

func (b *Box) ReadAll(r io.Reader) ([]byte, error) {
	return ioutil.ReadAll(r)
}

func (b *Box) ReadFile(filename string) ([]byte, error) {
	if vfsFilename, ok := ConvertVFSPath(filename); ok {
		return ioutil.ReadFile(b.vfs, vfsFilename)
	}

	return ioutil.ReadFile(b.osfs, filename)
}

func (b *Box) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if vfsFilename, ok := ConvertVFSPath(filename); ok {
		return ioutil.WriteFile(b.vfs, vfsFilename, data, perm)
	}

	return ioutil.WriteFile(b.osfs, filename, data, perm)
}

func (b *Box) ReadDir(dirname string) ([]os.FileInfo, error) {
	if vfsDirname, ok := ConvertVFSPath(dirname); ok {
		return ioutil.ReadDir(b.vfs, vfsDirname)
	}

	return ioutil.ReadDir(b.osfs, dirname)
}

func (b *Box) TempFile(dir, prefix string) (absfs.File, error) {
	if vfsDir, ok := ConvertVFSPath(dir); ok {
		return ioutil.TempFile(b.vfs, vfsDir, prefix)
	}

	return ioutil.TempFile(b.osfs, dir, prefix)
}

func (b *Box) TempDir(dir, prefix string) (string, error) {
	if vfsDir, ok := ConvertVFSPath(dir); ok {
		return ioutil.TempDir(b.vfs, vfsDir, prefix)
	}

	return ioutil.TempDir(b.osfs, dir, prefix)
}

func (b *Box) Close() {
	memguard.Purge()
}

func (b *Box) Abs(path string) (string, error) {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		absPath, err := b.vfs.Abs(vfsPath)
		if err != nil {
			return "", err
		}
		return MakeVFSPath(absPath), nil
	}

	return b.osfs.Abs(path)
}
