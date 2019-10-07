package pandorasbox

import (
	"os"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/ioutil"
)

func VFSAbs(path string) (string, error) {
	return box.vfs.Abs(path)
}

func VFSOpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return box.vfs.OpenFile(name, flag, perm)
}

func VFSMkdir(name string, perm os.FileMode) error {
	return box.vfs.Mkdir(name, perm)
}

func VFSRemove(name string) error {
	return box.vfs.Remove(name)
}

func VFSRename(oldpath, newpath string) error {
	return box.vfs.Rename(oldpath, newpath)
}

func VFSStat(name string) (os.FileInfo, error) {
	return box.vfs.Stat(name)
}

func VFSChmod(name string, mode os.FileMode) error {
	return box.vfs.Chmod(name, mode)
}

func VFSChtimes(name string, atime time.Time, mtime time.Time) error {
	return box.vfs.Chtimes(name, atime, mtime)
}

func VFSChown(name string, uid, gid int) error {
	return box.vfs.Chown(name, uid, gid)
}

func VFSOpen(name string) (absfs.File, error) {
	return box.vfs.Open(name)
}

func VFSCreate(name string) (absfs.File, error) {
	return box.vfs.Create(name)
}

func VFSMkdirAll(name string, perm os.FileMode) error {
	return box.vfs.MkdirAll(name, perm)
}

func VFSRemoveAll(path string) error {
	return box.vfs.RemoveAll(path)
}

func VFSTruncate(name string, size int64) error {
	return box.vfs.Truncate(name, size)
}

func VFSLstat(name string) (os.FileInfo, error) {
	return box.vfs.Lstat(name)
}

func VFSLchown(name string, uid, gid int) error {
	return box.vfs.Lchown(name, uid, gid)
}

func VFSReadlink(name string) (string, error) {
	return box.vfs.Readlink(name)
}

func VFSSymlink(oldname, newname string) error {
	return box.vfs.Symlink(oldname, newname)
}

// io/ioutil methods

func VFSReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(box.vfs, filename)
}

func VFSWriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(box.vfs, filename, data, perm)
}

func VFSReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(box.vfs, dirname)
}

func VFSTempFile(dir, prefix string) (absfs.File, error) {
	return ioutil.TempFile(box.vfs, dir, prefix)
}

func VFSTempDir(dir, prefix string) (string, error) {
	return ioutil.TempDir(box.vfs, dir, prefix)
}
