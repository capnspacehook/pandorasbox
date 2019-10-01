package pandorasbox

import (
	"os"
	"strings"

	"github.com/capnspacehook/pandorasbox/osfs"
	"github.com/capnspacehook/pandorasbox/vfs"
)

const VFSPrefix = "vfs://"

func ConvertVFSPath(path string) (string, bool) {
	if IsVFSPath(path) {
		return strings.Replace(path, VFSPrefix, "/", 1), true
	}

	return path, false
}

func IsVFSPath(path string) bool {
	if strings.HasPrefix(path, VFSPrefix) {
		return true
	}

	return false
}

func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func IsExist(err error) bool {
	return os.IsExist(err)
}

func SameFile(fi1, fi2 os.FileInfo) bool {
	vfsfi1, fi1vfs := fi1.(*vfs.FileInfo)
	vfsfi2, fi2vfs := fi2.(*vfs.FileInfo)

	if (fi1vfs && !fi2vfs) || (!fi1vfs && fi2vfs) {
		return false
	} else if fi1vfs && fi2vfs {
		return vfs.SameFile(vfsfi1, vfsfi2)
	} else {
		return osfs.SameFile(fi1, fi2)
	}
}

func IsVFS(fi os.FileInfo) bool {
	_, fivfs := fi.(*vfs.FileInfo)

	return fivfs
}
