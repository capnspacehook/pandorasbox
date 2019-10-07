package pandorasbox

import (
	"os"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

func (b *Box) OSAbs(path string) (string, error) {
	return b.osfs.Abs(path)
}

func (b *Box) OSOpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return b.osfs.OpenFile(name, flag, perm)
}

func (b *Box) OSMkdir(name string, perm os.FileMode) error {
	return b.osfs.Mkdir(name, perm)
}

func (b *Box) OSRemove(name string) error {
	return b.osfs.Remove(name)
}

func (b *Box) OSRename(oldpath, newpath string) error {
	return b.osfs.Rename(oldpath, newpath)
}

func (b *Box) OSStat(name string) (os.FileInfo, error) {
	return b.osfs.Stat(name)
}

func (b *Box) OSChmod(name string, mode os.FileMode) error {
	return b.osfs.Chmod(name, mode)
}

func (b *Box) OSChtimes(name string, atime time.Time, mtime time.Time) error {
	return b.osfs.Chtimes(name, atime, mtime)
}

func (b *Box) OSChown(name string, uid, gid int) error {
	return b.osfs.Chown(name, uid, gid)
}

func (b *Box) OSOpen(name string) (absfs.File, error) {
	return b.osfs.Open(name)
}

func (b *Box) OSCreate(name string) (absfs.File, error) {
	return b.osfs.Create(name)
}

func (b *Box) OSMkdirAll(name string, perm os.FileMode) error {
	return b.osfs.MkdirAll(name, perm)
}

func (b *Box) OSRemoveAll(path string) error {
	return b.osfs.RemoveAll(path)
}

func (b *Box) OSTruncate(name string, size int64) error {
	return b.osfs.Truncate(name, size)
}

func (b *Box) OSLstat(name string) (os.FileInfo, error) {
	return b.osfs.Lstat(name)
}

func (b *Box) OSLchown(name string, uid, gid int) error {
	return b.osfs.Lchown(name, uid, gid)
}

func (b *Box) OSReadlink(name string) (string, error) {
	return b.osfs.Readlink(name)
}

func (b *Box) OSSymlink(oldname, newname string) error {
	return b.osfs.Symlink(oldname, newname)
}

// io/ioutil methods

func (b *Box) OSReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(b.osfs, filename)
}

func (b *Box) OSWriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(b.osfs, filename, data, perm)
}

func (b *Box) OSReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(b.osfs, dirname)
}

func (b *Box) OSTempFile(dir, prefix string) (absfs.File, error) {
	return ioutil.TempFile(b.osfs, dir, prefix)
}

func (b *Box) OSTempDir(dir, prefix string) (string, error) {
	return ioutil.TempDir(b.osfs, dir, prefix)
}
