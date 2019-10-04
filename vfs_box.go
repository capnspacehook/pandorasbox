package pandorasbox

import (
	"os"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

func (b *Box) VFSOpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return b.vfs.OpenFile(name, flag, perm)
}

func (b *Box) VFSMkdir(name string, perm os.FileMode) error {
	return b.vfs.Mkdir(name, perm)
}

func (b *Box) VFSRemove(name string) error {
	return b.vfs.Remove(name)
}

func (b *Box) VFSRename(oldpath, newpath string) error {
	return b.vfs.Rename(oldpath, newpath)
}

func (b *Box) VFSStat(name string) (os.FileInfo, error) {
	return b.vfs.Stat(name)
}

func (b *Box) VFSChmod(name string, mode os.FileMode) error {
	return b.vfs.Chmod(name, mode)
}

func (b *Box) VFSChtimes(name string, atime time.Time, mtime time.Time) error {
	return b.vfs.Chtimes(name, atime, mtime)
}

func (b *Box) VFSChown(name string, uid, gid int) error {
	return b.vfs.Chown(name, uid, gid)
}

func (b *Box) VFSOpen(name string) (absfs.File, error) {
	return b.vfs.Open(name)
}

func (b *Box) VFSCreate(name string) (absfs.File, error) {
	return b.vfs.Create(name)
}

func (b *Box) VFSMkdirAll(name string, perm os.FileMode) error {
	return b.vfs.MkdirAll(name, perm)
}

func (b *Box) VFSRemoveAll(path string) error {
	return b.vfs.RemoveAll(path)
}

func (b *Box) VFSTruncate(name string, size int64) error {
	return b.vfs.Truncate(name, size)
}

func (b *Box) VFSLstat(name string) (os.FileInfo, error) {
	return b.vfs.Lstat(name)
}

func (b *Box) VFSLchown(name string, uid, gid int) error {
	return b.vfs.Lchown(name, uid, gid)
}

func (b *Box) VFSReadlink(name string) (string, error) {
	return b.vfs.Readlink(name)
}

func (b *Box) VFSSymlink(oldname, newname string) error {
	return b.vfs.Symlink(oldname, newname)
}

// io/ioutil methods

func (b *Box) VFSReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(b.vfs, filename)
}

func (b *Box) VFSWriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(b.vfs, filename, data, perm)
}

func (b *Box) VFSReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(b.vfs, dirname)
}

func (b *Box) VFSTempFile(dir, prefix string) (absfs.File, error) {
	return ioutil.TempFile(b.vfs, dir, prefix)
}

func (b *Box) VFSTempDir(dir, prefix string) (string, error) {
	return ioutil.TempDir(b.vfs, dir, prefix)
}
