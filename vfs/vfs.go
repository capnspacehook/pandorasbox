package vfs

import (
	"errors"
	"io"
	stdfs "io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/inode"
)

const (
	PathSeparator     = '/'
	PathListSeparator = ':'

	tempDir = "/tmp"

	_O_ACCESS = 0x3 // masks the access mode (os.O_RDONLY, os.O_WRONLY, or os.O_RDWR)
)

type stdFS struct {
	*virtualFS
}

func (fs stdFS) Open(name string) (stdfs.File, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.virtualFS.Open(name)
}

func (fs stdFS) ReadDir(name string) ([]stdfs.DirEntry, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.virtualFS.ReadDir(name)
}

func (fs stdFS) ReadFile(name string) ([]byte, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.virtualFS.ReadFile(name)
}

func (fs stdFS) StatFS(name string) (stdfs.FileInfo, error) {
	if err := checkPath(name, "stat"); err != nil {
		return nil, err
	}

	return fs.virtualFS.Stat(name)
}

func checkPath(name, op string) error {
	if path.IsAbs(name) {
		// if the name starts with a slash, return an error
		// to remain compatible with io/fs
		return &stdfs.PathError{Op: op, Path: name, Err: stdfs.ErrInvalid}
	}

	return nil
}

type virtualFS struct {
	mtx *sync.RWMutex

	root *inode.Inode
	cwd  string
	dir  *inode.Inode
	ino  *inode.Ino

	sfiles []*sealedFile
}

func NewFS() absfs.FileSystem {
	fs := new(virtualFS)
	fs.mtx = new(sync.RWMutex)
	fs.ino = new(inode.Ino)

	fs.root = fs.ino.NewDir(0o755)
	fs.cwd = "/"
	fs.dir = fs.root
	fs.sfiles = make([]*sealedFile, 2)

	return fs
}

func (fs *virtualFS) FS() stdfs.FS {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	// set cwd to root, as paths are not allowed to start with a slash
	// in io/fs filesystems
	return stdFS{virtualFS: &virtualFS{
		mtx:    fs.mtx,
		root:   fs.root,
		cwd:    "/",
		dir:    fs.dir,
		ino:    fs.ino,
		sfiles: fs.sfiles,
	}}
}

func (fs *virtualFS) Open(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *virtualFS) OpenFile(name string, flag int, perm stdfs.FileMode) (absfs.File, error) {
	if name == "/" {
		fs.mtx.RLock()
		sfile := fs.sfiles[int(fs.root.Ino)]
		fs.mtx.RUnlock()
		return &vfsFile{
			fs:    fs,
			name:  name,
			flags: flag,
			node:  fs.root,
			sfile: sfile,
		}, nil
	}

	// check that the path is valid
	var validPath bool
	if len(name) > 1 && name[0] == '/' {
		// if the path starts with a slash, don't call io/fs.ValidPath
		// with the leading slash, as we accept that but io/fs doesn't
		validPath = stdfs.ValidPath(name[1:])
	} else {
		validPath = stdfs.ValidPath(name)
	}
	if !validPath {
		return nil, &stdfs.PathError{Op: "open", Path: name, Err: stdfs.ErrInvalid}
	}

	appendFile := flag&os.O_APPEND != 0
	if name == "." {
		fs.mtx.Lock()
		sfile := fs.sfiles[int(fs.dir.Ino)]
		fs.mtx.Unlock()

		file := &vfsFile{
			fs:    fs,
			name:  name,
			flags: flag,
			node:  fs.dir,
			sfile: sfile,
		}
		if sfile != nil {
			if appendFile {
				fs.dir.RLock()
				file.offset.Store(fs.dir.Size)
				fs.dir.RUnlock()
			}
		}

		return file, nil
	}

	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	wd := fs.root
	if !path.IsAbs(name) {
		wd = fs.dir
	}

	var exists bool
	node, err := wd.Resolve(name)
	if err == nil {
		exists = true
	}

	dir, filename := path.Split(name)
	dir = path.Clean(dir)
	parent, err := wd.Resolve(dir)
	if err != nil {
		return nil, err
	}

	access := flag & _O_ACCESS
	create := flag&os.O_CREATE != 0
	truncate := flag&os.O_TRUNC != 0

	// error if it does not exist, and we are not allowed to create it.
	if !exists && !create {
		return nil, &stdfs.PathError{Op: "open", Path: name, Err: stdfs.ErrNotExist}
	}
	if exists {
		// err if exclusive create is required
		if create && flag&os.O_EXCL != 0 {
			return nil, &stdfs.PathError{Op: "open", Path: name, Err: stdfs.ErrExist}
		}
		if node.IsDir() {
			if access != os.O_RDONLY || truncate {
				return nil, &stdfs.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
			}
		}
	} else {
		// Create write-able file
		node = fs.ino.New(perm)
		err := parent.Link(filename, node)
		if err != nil {
			fs.ino.SubIno()
			return nil, &stdfs.PathError{Op: "open", Path: name, Err: err}
		}

		file := sealedFile{}
		fs.sfiles = append(fs.sfiles, &file)
	}
	sfile := fs.sfiles[int(node.Ino)]

	file := &vfsFile{
		fs:    fs,
		name:  name,
		flags: flag,
		node:  node,
		sfile: sfile,
	}
	if sfile != nil && (truncate || appendFile) {
		if truncate {
			if err := file.Truncate(0); err != nil {
				return nil, &stdfs.PathError{Op: "open", Path: name, Err: err}
			}
		}
		if appendFile {
			file.offset.Store(node.Size)
		}
	}

	return file, nil
}

func (fs *virtualFS) Create(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
}

func (fs *virtualFS) ReadFile(name string) ([]byte, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}

	vf := f.(*vfsFile)
	vf.node.RLock()
	size := vf.node.Size
	vf.node.RUnlock()

	data := make([]byte, size)
	n, err := f.Read(data)
	if err == nil && n < len(data) {
		err = io.ErrUnexpectedEOF
	}
	if closeErr := f.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	return data, err
}

func (fs *virtualFS) ReadDir(name string) ([]stdfs.DirEntry, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dirs, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(dirs, func(de1, de2 stdfs.DirEntry) int {
		return strings.Compare(de1.Name(), de2.Name())
	})

	return dirs, nil
}

func (fs *virtualFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	f, err := fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}

	return err
}

func (fs *virtualFS) Mkdir(name string, perm stdfs.FileMode) error {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	return fs.mkdir(name, perm)
}

func (fs *virtualFS) mkdir(name string, perm stdfs.FileMode) error {
	wd := fs.root
	abs := name
	if !path.IsAbs(abs) {
		abs = path.Join(fs.cwd, abs)
		wd = fs.dir
	}
	_, err := wd.Resolve(name)
	if err == nil {
		return &stdfs.PathError{Op: "mkdir", Path: name, Err: stdfs.ErrExist}
	}

	parent := fs.root
	dir, filename := path.Split(abs)
	dir = path.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &stdfs.PathError{Op: "mkdir", Path: dir, Err: err}
		}
	}

	child := fs.ino.NewDir(perm)
	if err := parent.Link(filename, child); err != nil {
		return &stdfs.PathError{Op: "mkdir", Path: filename, Err: err}
	}
	if err := child.Link("..", parent); err != nil {
		return &stdfs.PathError{Op: "mkdir", Path: "..", Err: err}
	}
	fs.sfiles = append(fs.sfiles, new(sealedFile))

	return nil
}

func (fs *virtualFS) MkdirAll(name string, perm stdfs.FileMode) error {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	name = inode.Abs(fs.cwd, name)

	dirpath := ""
	for _, p := range strings.Split(name, string(fs.Separator())) {
		if p == "" {
			p = "/"
		}

		dirpath = path.Join(dirpath, p)
		if err := fs.mkdir(dirpath, perm); err != nil {
			if !errors.Is(err, stdfs.ErrExist) {
				return err
			}
		}
	}

	return nil
}

func (fs *virtualFS) Stat(name string) (stdfs.FileInfo, error) {
	if name == "/" {
		return &FileInfo{"/", fs.root}, nil
	}

	fs.mtx.RLock()
	node, err := fs.fileStat(fs.cwd, name)
	fs.mtx.RUnlock()
	if err != nil {
		return nil, err
	}

	return &FileInfo{path.Base(name), node}, nil
}

func (fs *virtualFS) fileStat(cwd, name string) (*inode.Inode, error) {
	name = inode.Abs(cwd, name)
	if name != "/" {
		name = strings.TrimLeft(name, "/")
	}
	node, err := fs.root.Resolve(name)
	if err != nil {
		return nil, &stdfs.PathError{Op: "stat", Path: name, Err: err}
	}

	return node, nil
}

func (fs *virtualFS) Lstat(name string) (stdfs.FileInfo, error) {
	return fs.Stat(name)
}

func (fs *virtualFS) Rename(oldpath, newpath string) error {
	linkErr := os.LinkError{
		Op:  "rename",
		Old: oldpath,
		New: newpath,
	}

	if oldpath == "/" {
		linkErr.Err = errors.New("the root folder may not be moved or renamed")
		return &linkErr
	}

	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	if !path.IsAbs(oldpath) {
		oldpath = path.Join(fs.cwd, oldpath)
	}

	if !path.IsAbs(newpath) {
		newpath = path.Join(fs.cwd, newpath)
	}

	err := fs.root.Rename(oldpath, newpath)
	if err != nil {
		linkErr.Err = err
		return &linkErr
	}

	return nil
}

func (fs *virtualFS) Remove(name string) (err error) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	wd := fs.root
	abs := name
	if !path.IsAbs(abs) {
		abs = path.Join(fs.cwd, abs)
		wd = fs.dir
	}

	child, err := wd.Resolve(name)
	if err != nil {
		return &stdfs.PathError{Op: "remove", Path: name, Err: err}
	}

	if child.IsDir() {
		if len(child.Dir) > 2 {
			return &stdfs.PathError{Op: "remove", Path: name, Err: errors.New("directory not empty")}
		}
	}

	parent := fs.root
	dir, filename := path.Split(abs)
	dir = path.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &stdfs.PathError{Op: "remove", Path: dir, Err: err}
		}
	}

	ino := parent.Ino
	if err := parent.Unlink(filename); err != nil {
		return &stdfs.PathError{Op: "remove", Path: name, Err: err}
	}
	fs.sfiles[int(ino)] = nil

	return nil
}

func (fs *virtualFS) RemoveAll(name string) error {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	wd := fs.root
	abs := name
	if !path.IsAbs(abs) {
		abs = path.Join(fs.cwd, abs)
		wd = fs.dir
	}

	child, err := wd.Resolve(name)
	if err != nil {
		return &stdfs.PathError{Op: "remove", Path: name, Err: err}
	}

	parent := fs.root
	dir, filename := path.Split(abs)
	dir = path.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &stdfs.PathError{Op: "remove", Path: dir, Err: err}
		}
	}

	child.UnlinkAll()

	return parent.Unlink(filename)
}

func (fs *virtualFS) Truncate(name string, size int64) error {
	if size < 0 {
		return &stdfs.PathError{Op: "truncate", Path: name, Err: stdfs.ErrInvalid}
	}

	fs.mtx.RLock()
	path := inode.Abs(fs.cwd, name)
	child, err := fs.root.Resolve(path)
	if err != nil {
		fs.mtx.RUnlock()
		return err
	}

	sfile := fs.sfiles[child.Ino]
	fs.mtx.RUnlock()

	file := vfsFile{
		fs:    fs,
		name:  name,
		flags: os.O_WRONLY,
		node:  child,
		sfile: sfile,
	}

	return file.Truncate(size)
}

func (fs *virtualFS) WalkDir(root string, fn stdfs.WalkDirFunc) error {
	if path.IsAbs(root) {
		if root == "/" {
			root = "."
		} else {
			root = root[1:]
		}
	}

	return stdfs.WalkDir(fs.FS(), root, fn)
}

func (fs *virtualFS) Abs(p string) (string, error) {
	if strings.HasPrefix(p, string(PathSeparator)) {
		return path.Clean(p), nil
	}

	wd, err := fs.Getwd()
	if err != nil {
		return "", err
	}

	return path.Join(wd, p), nil
}

func (fs *virtualFS) Separator() uint8 {
	return PathSeparator
}

func (fs *virtualFS) ListSeparator() uint8 {
	return PathListSeparator
}

func (fs *virtualFS) Chdir(name string) (err error) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

	if name == "/" {
		fs.cwd = "/"
		fs.dir = fs.root
		return nil
	}

	wd := fs.root
	cwd := name
	if !path.IsAbs(name) {
		cwd = path.Join(fs.cwd, name)
		wd = fs.dir
	}

	node, err := wd.Resolve(name)
	if err != nil {
		return &stdfs.PathError{Op: "chdir", Path: name, Err: err}
	}

	if !node.IsDir() {
		return &stdfs.PathError{Op: "chdir", Path: name, Err: syscall.ENOTDIR}
	}

	fs.cwd = cwd
	fs.dir = node

	return nil
}

func (fs *virtualFS) Getwd() (dir string, err error) {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	return fs.cwd, nil
}

func (fs *virtualFS) TempDir() string {
	return tempDir
}
