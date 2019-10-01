package pandorasbox

import (
	"errors"
	"io"
	"os"
	"time"

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

func (b *Box) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.OpenFile(vfsName, flag, perm)
	}

	return b.osfs.OpenFile(name, flag, perm)
}

func (b *Box) Mkdir(name string, perm os.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Mkdir(vfsName, perm)
	}

	return b.osfs.Mkdir(name, perm)
}

func (b *Box) Remove(name string) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Remove(vfsName)
	}

	return b.osfs.Remove(name)
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

func (b *Box) Stat(name string) (os.FileInfo, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Stat(vfsName)
	}

	return b.osfs.Stat(name)
}

func (b *Box) Chmod(name string, mode os.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Chmod(vfsName, mode)
	}

	return b.osfs.Chmod(name, mode)
}

func (b *Box) Chtimes(name string, atime time.Time, mtime time.Time) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Chtimes(vfsName, atime, mtime)
	}

	return b.osfs.Chtimes(name, atime, mtime)
}

func (b *Box) Chown(name string, uid, gid int) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Chown(vfsName, uid, gid)
	}

	return b.osfs.Chown(name, uid, gid)
}

func (b *Box) Separator(vfs bool) uint8 {
	if vfs {
		return b.vfs.Separator()
	}

	return b.osfs.Separator()
}

func (b *Box) ListSeparator(vfs bool) uint8 {
	if vfs {
		return b.vfs.ListSeparator()
	}

	return b.osfs.ListSeparator()
}

func (b *Box) Chdir(dir string, vfs bool) error {
	if vfs {
		return b.vfs.Chdir(dir)
	}

	return b.osfs.Chdir(dir)
}

func (b *Box) Getwd(vfs bool) (string, error) {
	if vfs {
		return b.vfs.Getwd()
	}

	return b.osfs.Getwd()
}

func (b *Box) GetTempDir(vfs bool) string {
	if vfs {
		return b.vfs.TempDir()
	}

	return b.osfs.TempDir()
}

func (b *Box) Open(name string) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Open(vfsName)
	}

	return b.osfs.Open(name)
}

func (b *Box) Create(name string) (absfs.File, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Create(vfsName)
	}

	return b.osfs.Create(name)
}

func (b *Box) MkdirAll(name string, perm os.FileMode) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.MkdirAll(vfsName, perm)
	}

	return b.osfs.MkdirAll(name, perm)
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

func (b *Box) Lstat(name string) (os.FileInfo, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Lstat(vfsName)
	}

	return b.osfs.Lstat(name)
}

func (b *Box) Lchown(name string, uid, gid int) error {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Lchown(vfsName, uid, gid)
	}

	return b.osfs.Lchown(name, uid, gid)
}

func (b *Box) Readlink(name string) (string, error) {
	if vfsName, ok := ConvertVFSPath(name); ok {
		return b.vfs.Readlink(vfsName)
	}

	return b.osfs.Readlink(name)
}

func (b *Box) Symlink(oldname, newname string) error {
	vfsOldName, oldNameVFS := ConvertVFSPath(oldname)
	vfsNewName, newNameVFS := ConvertVFSPath(newname)
	if oldNameVFS && newNameVFS {
		return b.vfs.Rename(vfsOldName, vfsNewName)
	} else if (oldNameVFS && !newNameVFS) || (!oldNameVFS && newNameVFS) {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: errors.New("oldname and newname must both either be a VFS path, or normal path")}
	}

	return b.osfs.Rename(oldname, newname)
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
