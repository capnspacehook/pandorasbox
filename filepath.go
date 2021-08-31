package pandorasbox

import (
	stdpath "path"
	"path/filepath"
)

func IsAbs(path string) bool {
	if _, ok := ConvertVFSPath(path); ok {
		return stdpath.IsAbs(path)
	}

	return filepath.IsAbs(path)
}

func Clean(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(stdpath.Clean(path))
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
		dir, file := stdpath.Split(path)
		dir = MakeVFSPath(dir)
		return dir, file
	}

	return filepath.Split(path)
}

func Join(elem ...string) string {
	var isVFS bool
	for i := range elem {
		vfsPath, ok := ConvertVFSPath(elem[i])
		if ok {
			elem[i] = vfsPath
		}

		if i == 0 {
			isVFS = ok
		}
	}

	if isVFS {
		return MakeVFSPath(stdpath.Join(elem...))
	}

	return filepath.Join(elem...)
}

func Ext(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(stdpath.Ext(path))
	}

	return filepath.Ext(path)
}

func Base(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(stdpath.Base(path))
	}

	return filepath.Base(path)
}

func Dir(path string) string {
	if vfsPath, ok := ConvertVFSPath(path); ok {
		path = vfsPath
		return MakeVFSPath(stdpath.Dir(path))
	}

	return filepath.Dir(path)
}
