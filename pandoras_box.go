package pandorasbox

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/capnspacehook/pandorasbox/absfs"
)

var box *Box

func InitGlobalBox() {
	box = NewBox()
}

func Abs(path string) (string, error) {
	return box.Abs(path)
}

func OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return box.OpenFile(name, flag, perm)
}

func Mkdir(name string, perm os.FileMode) error {
	return box.Mkdir(name, perm)
}

func Remove(name string) error {
	return box.Remove(name)
}

func Rename(oldpath, newpath string) error {
	return box.Rename(oldpath, newpath)
}

func Stat(name string) (os.FileInfo, error) {
	return box.Stat(name)
}

func Chmod(name string, mode os.FileMode) error {
	return box.Chmod(name, mode)
}

func Chtimes(name string, atime time.Time, mtime time.Time) error {
	return box.Chtimes(name, atime, mtime)
}

func Chown(name string, uid, gid int) error {
	return box.Chown(name, uid, gid)
}

func Separator(vfs bool) uint8 {
	return box.Separator(vfs)
}

func ListSeparator(vfs bool) uint8 {
	return box.ListSeparator(vfs)
}

func IsPathSeparator(c uint8, vfs bool) bool {
	return box.IsPathSeparator(c, vfs)
}

func Chdir(dir string, vfs bool) error {
	return box.Chdir(dir, vfs)
}

func Getwd(vfs bool) (string, error) {
	return box.Getwd(vfs)
}

func GetTempDir(vfs bool) string {
	return box.GetTempDir(vfs)
}

func Open(name string) (absfs.File, error) {
	return box.Open(name)
}

func Create(name string) (absfs.File, error) {
	return box.Create(name)
}

func MkdirAll(name string, perm os.FileMode) error {
	return box.MkdirAll(name, perm)
}

func RemoveAll(path string) error {
	return box.RemoveAll(path)
}

func Truncate(name string, size int64) error {
	return box.Truncate(name, size)
}

func Lstat(name string) (os.FileInfo, error) {
	return box.Lstat(name)
}

func Lchown(name string, uid, gid int) error {
	return box.Lchown(name, uid, gid)
}

func Readlink(name string) (string, error) {
	return box.Readlink(name)
}

func Symlink(oldname, newname string) error {
	return box.Symlink(oldname, newname)
}

func Walk(root string, walkFn filepath.WalkFunc) error {
	return box.Walk(root, walkFn)
}

// io/ioutil methods

func ReadAll(r io.Reader) ([]byte, error) {
	return box.ReadAll(r)
}

func ReadFile(filename string) ([]byte, error) {
	return box.ReadFile(filename)
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	return box.WriteFile(filename, data, perm)
}

func ReadDir(dirname string) ([]os.FileInfo, error) {
	return box.ReadDir(dirname)
}

func TempFile(dir, prefix string) (absfs.File, error) {
	return box.TempFile(dir, prefix)
}

func TempDir(dir, prefix string) (string, error) {
	return box.TempDir(dir, prefix)
}

func Close() {
	box.Close()
}
