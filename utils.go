package pandorasbox

import (
	"io/fs"
	"os"
	"strings"

	"github.com/capnspacehook/pandorasbox/osfs"
	"github.com/capnspacehook/pandorasbox/vfs"
)

const VFSPrefix = "vfs://"

func ConvertVFSPath(path string) (string, bool) {
	if IsVFSPath(path) {
		return convertVFSPath(path), true
	}

	return path, false
}

func convertVFSPath(path string) string {
	return strings.Replace(path, VFSPrefix, "/", 1)
}

func IsVFSPath(path string) bool {
	return strings.HasPrefix(path, VFSPrefix)
}

func MakeVFSPath(path string) string {
	vfsPath := strings.Replace(path, "/", VFSPrefix, 1)
	if vfsPath == path {
		vfsPath = VFSPrefix + path
	}

	return vfsPath
}

func IsVFS(fi fs.FileInfo) bool {
	_, fivfs := fi.(*vfs.FileInfo)

	return fivfs
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
