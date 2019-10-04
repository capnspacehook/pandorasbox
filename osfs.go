package pandorasbox

import (
	"os"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

func OSOpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return box.osfs.OpenFile(name, flag, perm)
}

func OSMkdir(name string, perm os.FileMode) error {
	return box.osfs.Mkdir(name, perm)
}

func OSRemove(name string) error {
	return box.osfs.Remove(name)
}

func OSRename(oldpath, newpath string) error {
	return box.osfs.Rename(oldpath, newpath)
}

func OSStat(name string) (os.FileInfo, error) {
	return box.osfs.Stat(name)
}

func OSChmod(name string, mode os.FileMode) error {
	return box.osfs.Chmod(name, mode)
}

func OSChtimes(name string, atime time.Time, mtime time.Time) error {
	return box.osfs.Chtimes(name, atime, mtime)
}

func OSChown(name string, uid, gid int) error {
	return box.osfs.Chown(name, uid, gid)
}

func OSOpen(name string) (absfs.File, error) {
	return box.osfs.Open(name)
}

func OSCreate(name string) (absfs.File, error) {
	return box.osfs.Create(name)
}

func OSMkdirAll(name string, perm os.FileMode) error {
	return box.osfs.MkdirAll(name, perm)
}

func OSRemoveAll(path string) error {
	return box.osfs.RemoveAll(path)
}

func OSTruncate(name string, size int64) error {
	return box.osfs.Truncate(name, size)
}

func OSLstat(name string) (os.FileInfo, error) {
	return box.osfs.Lstat(name)
}

func OSLchown(name string, uid, gid int) error {
	return box.osfs.Lchown(name, uid, gid)
}

func OSReadlink(name string) (string, error) {
	return box.osfs.Readlink(name)
}

func OSSymlink(oldname, newname string) error {
	return box.osfs.Symlink(oldname, newname)
}

// io/ioutil methods

func OSReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(box.osfs, filename)
}

func OSWriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(box.osfs, filename, data, perm)
}

func OSReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(box.osfs, dirname)
}

func OSTempFile(dir, prefix string) (absfs.File, error) {
	return ioutil.TempFile(box.osfs, dir, prefix)
}

func OSTempDir(dir, prefix string) (string, error) {
	return ioutil.TempDir(box.osfs, dir, prefix)
}
