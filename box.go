package pandorasbox

import (
	"errors"
	"io/fs"
	"os"

	"github.com/awnumar/memguard"
	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
	"github.com/capnspacehook/pandorasbox/osfs"
	"github.com/capnspacehook/pandorasbox/vfs"
)

type Box struct {
	osfs absfs.FileSystem
	vfs  absfs.FileSystem
}

func NewBox() *Box {
	box := new(Box)
	box.osfs = osfs.NewFS()
	box.vfs = vfs.NewFS()

	return box
}

func (b *Box) OSFS() absfs.FileSystem {
	return b.osfs
}

func (b *Box) VFS() absfs.FileSystem {
	return b.vfs
}

func (b *Box) Open(name string) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Open(vfsName)
	}

	return b.osfs.Open(name)
}

func (b *Box) OpenFile(name string, flag int, perm fs.FileMode) (absfs.File, error) {
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

func (b *Box) ReadFile(filename string) ([]byte, error) {
	if vfsFilename, ok := ConvertVFSPath(filename); ok {
		return b.vfs.ReadFile(vfsFilename)
	}

	return b.osfs.ReadFile(filename)
}

func (b *Box) ReadDir(dirname string) ([]fs.DirEntry, error) {
	if vfsDirname, ok := ConvertVFSPath(dirname); ok {
		return b.vfs.ReadDir(vfsDirname)
	}

	return b.osfs.ReadDir(dirname)
}

func (b *Box) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	if vfsFilename, ok := ConvertVFSPath(filename); ok {
		return ioutil.WriteFile(b.vfs, vfsFilename, data, perm)
	}

	return ioutil.WriteFile(b.osfs, filename, data, perm)
}

func (b *Box) Mkdir(name string, perm fs.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Mkdir(vfsName, perm)
	}

	return b.osfs.Mkdir(name, perm)
}

func (b *Box) MkdirAll(name string, perm fs.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.MkdirAll(vfsName, perm)
	}

	return b.osfs.MkdirAll(name, perm)
}

func (b *Box) Stat(name string) (fs.FileInfo, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Stat(vfsName)
	}

	return b.osfs.Stat(name)
}

func (b *Box) Lstat(name string) (fs.FileInfo, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Lstat(vfsName)
	}

	return b.osfs.Lstat(name)
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

func (b *Box) WalkDir(root string, fn fs.WalkDirFunc) error {
	if vfsName, ok := ConvertVFSPath(root); ok {
		return b.vfs.WalkDir(vfsName, fn)
	}

	return b.osfs.WalkDir(root, fn)
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

func (b *Box) Close() {
	memguard.Purge()
}
