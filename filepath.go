package pandorasbox

import (
	"errors"
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

func ToSlash(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(filepath.ToSlash(path))
	}

	return filepath.ToSlash(path)
}

func FromSlash(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(filepath.FromSlash(path))
	}

	return filepath.FromSlash(path)
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

	for i := range elem[1:] {
		if vfsPath, ok := ConvertVFSPath(elem[i+1]); ok {
			elem[i+1] = vfsPath
		}
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

func Rel(basepath, targpath string) (string, error) {
	vfsBasepath, basepathVfs := ConvertVFSPath(basepath)
	vfsTargpath, targpathVfs := ConvertVFSPath(targpath)

	if (basepathVfs && !targpathVfs) || (!basepathVfs && targpathVfs) {
		return "", errors.New("basepath and targpath must both be a VFS path")
	} else if basepathVfs && targpathVfs {
		return vfs.Rel(vfsBasepath, vfsTargpath)
	}

	return filepath.Rel(basepath, targpath)
}
