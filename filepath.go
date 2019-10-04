package pandorasbox

import (
	"path/filepath"

	"github.com/capnspacehook/pandorasbox/vfs"
)

func IsAbs(path string) bool {
	if _, ok := ConvertVFSPath(path); ok {
		return vfs.IsAbs(path)
	}

	return filepath.IsAbs(path)
}

func Clean(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(vfs.Clean(path))
	}

	return filepath.Clean(path)
}

func Split(path string) (string, string) {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		dir, file := vfs.Split(path)
		dir = MakeVFSPath(dir)
		return dir, file
	}

	return filepath.Split(path)
}

func Join(elem ...string) string {
	if vfsPath, ok := ConvertVFSPath(elem[0]); ok {
		elem[0] = vfsPath
		return MakeVFSPath(vfs.Join(elem...))
	}

	return filepath.Join(elem...)
}

func Ext(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(vfs.Ext(path))
	}

	return filepath.Ext(path)
}

func Base(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(vfs.Base(path))
	}

	return filepath.Base(path)
}

func Dir(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(vfs.Dir(path))
	}

	return filepath.Dir(path)
}
