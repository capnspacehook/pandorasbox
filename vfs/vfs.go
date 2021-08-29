package vfs

import (
	"errors"
	"io"
	stdfs "io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/awnumar/fastrand"
	"github.com/awnumar/memguard"
	"github.com/awnumar/memguard/core"

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
	*pbFS
}

func (fs stdFS) Open(name string) (stdfs.File, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.pbFS.Open(name)
}

func (fs stdFS) ReadDir(name string) ([]stdfs.DirEntry, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.pbFS.ReadDir(name)
}

func (fs stdFS) ReadFile(name string) ([]byte, error) {
	if err := checkPath(name, "open"); err != nil {
		return nil, err
	}

	return fs.pbFS.ReadFile(name)
}

func (fs stdFS) StatFS(name string) (stdfs.FileInfo, error) {
	if err := checkPath(name, "stat"); err != nil {
		return nil, err
	}

	return fs.pbFS.Stat(name)
}

func checkPath(name, op string) error {
	if path.IsAbs(name) {
		// if the name starts with a slash, return an error
		// to remain compatible with io/fs
		return &stdfs.PathError{Op: op, Path: name, Err: stdfs.ErrInvalid}
	}

	return nil
}

type pbFS struct {
	mtx *sync.RWMutex

	root *inode.Inode
	cwd  string
	dir  *inode.Inode
	ino  *inode.Ino

	data []*sealedFile
}

func NewFS() absfs.FileSystem {
	fs := new(pbFS)
	fs.mtx = new(sync.RWMutex)
	fs.ino = new(inode.Ino)

	fs.root = fs.ino.NewDir(0755)
	fs.cwd = "/"
	fs.dir = fs.root
	fs.data = make([]*sealedFile, 2)

	return fs
}

func (fs *pbFS) FS() stdfs.FS {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	// set cwd to root, as paths are not allowed to start with a slash
	// in io/fs filesystems
	return stdFS{pbFS: &pbFS{
		mtx:  fs.mtx,
		root: fs.root,
		cwd:  "/",
		dir:  fs.dir,
		ino:  fs.ino,
		data: fs.data,
	}}
}

func (fs *pbFS) Open(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *pbFS) OpenFile(name string, flag int, perm stdfs.FileMode) (absfs.File, error) {
	if name == "/" {
		data := fs.data[int(fs.root.Ino)]
		return &file{
			fs:    fs,
			name:  name,
			flags: flag,
			node:  fs.root,
			data:  data,
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
		data := fs.data[int(fs.dir.Ino)]
		file := &file{
			fs:    fs,
			name:  name,
			flags: flag,
			node:  fs.dir,
			data:  data,
		}
		if data != nil {
			if appendFile {
				file.offset = fs.dir.Size
			}
			data.f = file
		}

		return file, nil
	}

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
		return nil, &stdfs.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
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

		// if we must truncate the file
		if truncate {
			sfile := fs.data[int(node.Ino)]
			sfile.ciphertext = nil
			sfile.key = nil
		}
	} else {
		// error if we cannot create the file
		if !create {
			return nil, &stdfs.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
		}

		// Create write-able file
		node = fs.ino.New(perm)
		err := parent.Link(filename, node)
		if err != nil {
			fs.ino.SubIno()
			return nil, &stdfs.PathError{Op: "open", Path: name, Err: err}
		}
		fs.data = append(fs.data, new(sealedFile))
	}
	data := fs.data[int(node.Ino)]

	file := &file{
		fs:    fs,
		name:  name,
		flags: flag,
		node:  node,
		data:  data,
	}
	if data != nil {
		if truncate {
			node.Size = 0
		}
		if appendFile {
			file.offset = node.Size
		}
		data.f = file
	}

	return file, nil
}

func (fs *pbFS) Create(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
}

func (fs *pbFS) ReadFile(name string) ([]byte, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// will never error
	fi, _ := f.Stat()

	data := make([]byte, fi.Size())
	n, err := f.Read(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}

	return data, err
}

func (fs *pbFS) ReadDir(name string) ([]stdfs.DirEntry, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dirs, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	return dirs, nil
}

func (fs *pbFS) WriteFile(name string, data []byte, perm os.FileMode) error {
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

func (fs *pbFS) Mkdir(name string, perm stdfs.FileMode) error {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()

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
	parent.Link(filename, child)
	child.Link("..", parent)
	fs.data = append(fs.data, new(sealedFile))

	return nil
}

func (fs *pbFS) MkdirAll(name string, perm stdfs.FileMode) error {
	fs.mtx.RLock()
	name = inode.Abs(fs.cwd, name)
	fs.mtx.RUnlock()

	dirpath := ""
	for _, p := range strings.Split(name, string(fs.Separator())) {
		if p == "" {
			p = "/"
		}

		dirpath = path.Join(dirpath, p)
		if err := fs.Mkdir(dirpath, perm); err != nil {
			if !errors.Is(err, stdfs.ErrExist) {
				return err
			}
		}
	}

	return nil
}

func (fs *pbFS) Stat(name string) (stdfs.FileInfo, error) {
	if name == "/" {
		return &FileInfo{"/", fs.root}, nil
	}
	node, err := fs.fileStat(fs.cwd, name)
	if err != nil {
		return nil, err
	}

	return &FileInfo{path.Base(name), node}, nil
}

func (fs *pbFS) fileStat(cwd, name string) (*inode.Inode, error) {
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

func (fs *pbFS) Lstat(name string) (stdfs.FileInfo, error) {
	return fs.Stat(name)
}

func (fs *pbFS) Rename(oldpath, newpath string) error {
	linkErr := os.LinkError{
		Op:  "rename",
		Old: oldpath,
		New: newpath,
	}

	if oldpath == "/" {
		linkErr.Err = errors.New("the root folder may not be moved or renamed")
		return &linkErr
	}

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

func (fs *pbFS) Remove(name string) (err error) {
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

	return parent.Unlink(filename)
}

func (fs *pbFS) RemoveAll(name string) error {
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

func (fs *pbFS) Truncate(name string, size int64) error {
	if size < 0 {
		return &stdfs.PathError{Op: "truncate", Path: name, Err: stdfs.ErrClosed}
	}

	path := inode.Abs(fs.cwd, name)
	child, err := fs.root.Resolve(path)
	if err != nil {
		return err
	}

	fs.mtx.RLock()
	file := fs.data[child.Ino]
	fs.mtx.RUnlock()

	var plaintext []byte
	if file.f.node.Size != 0 {
		file.f.mtx.RLock()
		key, err := file.key.Open()
		if err != nil {
			return err
		}

		plaintext = make([]byte, file.f.node.Size)
		_, err = core.Decrypt(file.ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return err
		}

		key.Destroy()
		file.f.mtx.RUnlock()
	} else if size == 0 { // data is already nil, no-op
		return nil
	}

	// TODO: should this be copied in constant time?
	if size <= file.f.node.Size {
		plaintext = plaintext[:int(size)]
		newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))

		file.f.mtx.Lock()
		file.ciphertext, err = core.Encrypt(plaintext, newKey.Bytes())
		file.key = newKey.Seal()
		file.f.updateSize()
		file.f.mtx.Unlock()

		core.Wipe(plaintext)
		if err != nil {
			return err
		}
		return nil
	}

	data := make([]byte, int(size))
	core.Move(data, plaintext)

	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))

	file.f.mtx.Lock()
	file.ciphertext, err = core.Encrypt(data, newKey.Bytes())
	file.key = newKey.Seal()
	file.f.updateSize()
	file.f.mtx.Unlock()

	core.Wipe(data)
	if err != nil {
		return err
	}

	return nil
}

func (fs *pbFS) WalkDir(root string, fn stdfs.WalkDirFunc) error {
	if path.IsAbs(root) {
		if root == "/" {
			root = "."
		} else {
			root = root[1:]
		}
	}

	return stdfs.WalkDir(fs.FS(), root, fn)
}

func (fs *pbFS) Abs(p string) (string, error) {
	if strings.HasPrefix(p, string(PathSeparator)) {
		return path.Clean(p), nil
	}

	wd, err := fs.Getwd()
	if err != nil {
		return "", err
	}

	return path.Join(wd, p), nil
}

func (fs *pbFS) Separator() uint8 {
	return PathSeparator
}

func (fs *pbFS) ListSeparator() uint8 {
	return PathListSeparator
}

func (fs *pbFS) Chdir(name string) (err error) {
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

func (fs *pbFS) Getwd() (dir string, err error) {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	return fs.cwd, nil
}

func (fs *pbFS) TempDir() string {
	return tempDir
}
