package osfs

import (
	"fmt"
	"os"
	"path/filepath"
)

type File struct {
	filer *FileSystem
	f     *os.File
}

func (f *File) Name() string {
	return f.f.Name()
}

func (f *File) Read(p []byte) (int, error) {
	return f.f.Read(p)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return f.f.ReadAt(b, off)
}

func (f *File) Write(p []byte) (int, error) {
	return f.f.Write(p)
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return f.f.WriteAt(b, off)
}

func (f *File) Close() error {
	return f.f.Close()
}

func (f *File) Seek(offset int64, whence int) (ret int64, err error) {
	return f.f.Seek(offset, whence)
}

func (f *File) Stat() (os.FileInfo, error) {
	if !filepath.IsAbs(f.f.Name()) {
		panic("not absolute path: " + f.f.Name())
	}
	info, err := os.Lstat(f.f.Name())
	if err != nil {
		return info, err
	}
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		link, err := os.Readlink(f.f.Name())
		if err != nil {
			panic(err)
		}
		panic(fmt.Sprintf("symlink %q -> %q", f.f.Name(), link))
	}
	return info, err
}

func (f *File) Sync() error {
	return f.f.Sync()
}

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	return f.f.Readdir(n)
}

func (f *File) Readdirnames(n int) ([]string, error) {
	return f.f.Readdirnames(n)
}

func (f *File) Truncate(size int64) error {
	return f.f.Truncate(size)
}

func (f *File) WriteString(s string) (n int, err error) {
	return f.f.WriteString(s)
}
