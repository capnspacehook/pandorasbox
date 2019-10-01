package vfs

import (
	"errors"
	"os"
	filepath "path" // force forward slash separators on all OSs.
	pathfilepath "path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/awnumar/fastrand"
	"github.com/awnumar/memguard"
	"github.com/awnumar/memguard/core"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/inode"
)

type FileSystem struct {
	sync.RWMutex

	Umask   os.FileMode
	Tempdir string

	root *inode.Inode
	cwd  string
	dir  *inode.Inode
	ino  *inode.Ino

	symlinks map[uint64]string
	data     []*sealedFile
}

func NewFS() *FileSystem {
	fs := new(FileSystem)
	fs.ino = new(inode.Ino)
	fs.Tempdir = "/tmp"

	fs.Umask = 0755
	fs.root = fs.ino.NewDir(fs.Umask)
	fs.cwd = "/"
	fs.dir = fs.root
	fs.data = make([]*sealedFile, 2)
	fs.symlinks = make(map[uint64]string)

	return fs
}

func (fs *FileSystem) Separator() uint8 {
	return '/'
}

func (fs *FileSystem) ListSeparator() uint8 {
	return ':'
}

func (fs *FileSystem) Rename(oldpath, newpath string) error {
	linkErr := &os.LinkError{
		Op:  "rename",
		Old: oldpath,
		New: newpath,
	}
	if oldpath == "/" {
		linkErr.Err = errors.New("the root folder may not be moved or renamed")
		return linkErr
	}

	if !filepath.IsAbs(oldpath) {
		oldpath = filepath.Join(fs.cwd, oldpath)
	}

	if !filepath.IsAbs(newpath) {
		newpath = filepath.Join(fs.cwd, newpath)
	}
	err := fs.root.Rename(oldpath, newpath)
	if err != nil {
		linkErr.Err = err
		return linkErr
	}
	return nil
}

func (fs *FileSystem) Chdir(name string) (err error) {
	fs.Lock()
	defer fs.Unlock()

	if name == "/" {
		fs.cwd = "/"
		fs.dir = fs.root
		return nil
	}
	wd := fs.root
	cwd := name
	if !filepath.IsAbs(name) {
		cwd = filepath.Join(fs.cwd, name)
		wd = fs.dir
	}

	node, err := wd.Resolve(name)
	if err != nil {
		return &os.PathError{Op: "chdir", Path: name, Err: err}
	}
	if !node.IsDir() {
		return &os.PathError{Op: "chdir", Path: name, Err: errors.New("not a directory")}
	}

	fs.cwd = cwd
	fs.dir = node
	return nil
}

func (fs *FileSystem) Getwd() (dir string, err error) {
	fs.RLock()
	defer fs.RUnlock()

	return fs.cwd, nil
}

func (fs *FileSystem) TempDir() string {
	fs.RLock()
	defer fs.RUnlock()

	return fs.Tempdir
}

func (fs *FileSystem) Open(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *FileSystem) Create(name string) (absfs.File, error) {
	return fs.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
}

func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	if name == "/" {
		data := fs.data[int(fs.root.Ino)]
		return &File{fs: fs, name: name, flags: flag, node: fs.root, data: data}, nil
	}
	appendFile := flag&absfs.O_APPEND != 0
	if name == "." {
		data := fs.data[int(fs.dir.Ino)]
		file := &File{fs: fs, name: name, flags: flag, node: fs.dir, data: data}
		if data != nil {
			if appendFile {
				file.offset = data.size
			}
			data.f = file
		}

		return file, nil
	}

	wd := fs.root
	if !filepath.IsAbs(name) {
		wd = fs.dir
	}
	var exists bool
	node, err := wd.Resolve(name)
	if err == nil {
		exists = true
	}

	dir, filename := filepath.Split(name)
	dir = filepath.Clean(dir)
	parent, err := wd.Resolve(dir)
	if err != nil {
		return nil, err
	}

	access := flag & absfs.O_ACCESS
	create := flag&absfs.O_CREATE != 0
	truncate := flag&absfs.O_TRUNC != 0

	// error if it does not exist, and we are not allowed to create it.
	if !exists && !create {
		return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	if exists {
		// err if exclusive create is required
		if create && flag&os.O_EXCL != 0 {
			return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.EEXIST}
		}
		if node.IsDir() {
			if access != os.O_RDONLY || truncate {
				return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.EISDIR} // os.ErrNotExist}
			}
		}

		// if we must truncate the file
		if truncate {
			fs.data[int(node.Ino)] = &sealedFile{}
		}

	} else { // !exists
		// error if we cannot create the file
		if !create {
			return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT} //os.ErrNotExist}
		}

		// Create write-able file
		node = fs.ino.New(fs.Umask & perm)
		err := parent.Link(filename, node)
		if err != nil {
			return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: err}
		}
		fs.data = append(fs.data, &sealedFile{})
	}
	data := fs.data[int(node.Ino)]

	if !create {
		if access == os.O_RDONLY && node.Mode&absfs.OS_ALL_R == 0 ||
			access == os.O_WRONLY && node.Mode&absfs.OS_ALL_W == 0 ||
			access == os.O_RDWR && node.Mode&(absfs.OS_ALL_W|absfs.OS_ALL_R) == 0 {
			return &absfs.InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: os.ErrPermission}
		}
	}

	file := &File{fs: fs, name: name, flags: flag, node: node, data: data}
	if data != nil {
		if appendFile {
			file.offset = data.size
		}
		data.f = file
	}

	return file, nil
}

func (fs *FileSystem) Truncate(name string, size int64) error {
	fs.Lock()
	defer fs.Unlock()

	path := inode.Abs(fs.cwd, name)
	child, err := fs.root.Resolve(path)
	if err != nil {
		return err
	}

	var (
		i         = int(child.Ino)
		plaintext []byte
	)

	if fs.data[i].size != 0 {
		fs.data[i].f.RLock()
		key, err := fs.data[i].key.Open()
		if err != nil {
			return err
		}
		plaintext = make([]byte, fs.data[i].size)
		_, err = core.Decrypt(fs.data[i].ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return err
		}
		key.Destroy()
		fs.data[i].f.RUnlock()
	} else if size == 0 { // data is already nil, no-op
		return nil
	}

	// TODO: should this be copied in constant time?
	if size <= fs.data[i].size {
		plaintext = plaintext[:int(size)]
		newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))

		fs.data[i].f.Lock()
		fs.data[i].ciphertext, err = core.Encrypt(plaintext, newKey.Bytes())
		fs.data[i].key = newKey.Seal()
		fs.data[i].updateSize()
		fs.data[i].f.Unlock()

		core.Wipe(plaintext)
		if err != nil {
			return err
		}
		return nil
	}

	data := make([]byte, int(size))
	core.Move(data, plaintext)

	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))

	fs.data[i].f.Lock()
	fs.data[i].ciphertext, err = core.Encrypt(data, newKey.Bytes())
	fs.data[i].key = newKey.Seal()
	fs.data[i].updateSize()
	fs.data[i].f.Unlock()

	core.Wipe(data)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) Mkdir(name string, perm os.FileMode) error {
	fs.Lock()
	defer fs.Unlock()

	wd := fs.root
	abs := name
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(fs.cwd, abs)
		wd = fs.dir
	}
	_, err := wd.Resolve(name)
	if err == nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: os.ErrExist}
	}

	parent := fs.root
	dir, filename := filepath.Split(abs)
	dir = filepath.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &os.PathError{Op: "mkdir", Path: dir, Err: err}
		}
	}

	child := fs.ino.NewDir(fs.Umask & perm)
	parent.Link(filename, child)
	child.Link("..", parent)
	fs.data = append(fs.data, &sealedFile{})

	return nil
}

func (fs *FileSystem) MkdirAll(name string, perm os.FileMode) error {
	fs.RLock()
	name = inode.Abs(fs.cwd, name)
	fs.RUnlock()
	path := ""
	for _, p := range strings.Split(name, string(fs.Separator())) {
		if p == "" {
			p = "/"
		}
		path = filepath.Join(path, p)
		fs.Mkdir(path, perm)
	}

	return nil
}

func (fs *FileSystem) Remove(name string) (err error) {
	wd := fs.root
	abs := name
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(fs.cwd, abs)
		wd = fs.dir
	}
	child, err := wd.Resolve(name)
	if err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}

	if child.IsDir() {
		if len(child.Dir) > 0 {
			return &os.PathError{Op: "remove", Path: name, Err: errors.New("directory not empty")}
		}
	}

	parent := fs.root
	dir, filename := filepath.Split(abs)
	dir = filepath.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &os.PathError{Op: "remove", Path: dir, Err: err}
		}
	}
	return parent.Unlink(filename)
}

func (fs *FileSystem) RemoveAll(name string) error {
	wd := fs.root
	abs := name
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(fs.cwd, abs)
		wd = fs.dir
	}
	child, err := wd.Resolve(name)
	if err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}

	parent := fs.root
	dir, filename := filepath.Split(abs)
	dir = filepath.Clean(dir)
	if dir != "/" {
		parent, err = fs.root.Resolve(strings.TrimLeft(dir, "/"))
		if err != nil {
			return &os.PathError{Op: "remove", Path: dir, Err: err}
		}
	}
	child.UnlinkAll()
	return parent.Unlink(filename)
}

//Chtimes changes the access and modification times of the named file
func (fs *FileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	var err error
	node := fs.root

	name = inode.Abs(fs.cwd, name)
	if name != "/" {
		node, err = fs.root.Resolve(strings.TrimLeft(name, "/"))
		if err != nil {
			return err
		}
	}

	node.Atime = atime
	node.Mtime = mtime

	return nil
}

//Chown changes the owner and group ids of the named file
func (fs *FileSystem) Chown(name string, uid, gid int) error {
	var err error
	node := fs.root

	name = inode.Abs(fs.cwd, name)
	if name != "/" {
		node, err = fs.root.Resolve(name)
		if err != nil {
			return err
		}
	}
	node.Uid = uint32(uid)
	node.Gid = uint32(gid)

	return nil
}

//Chmod changes the mode of the named file to mode.
func (fs *FileSystem) Chmod(name string, mode os.FileMode) error {
	var err error
	node := fs.root

	name = inode.Abs(fs.cwd, name)

	// return nil
	if name != "/" {
		node, err = fs.root.Resolve(strings.TrimLeft(name, "/"))
		if err != nil {
			return err
		}
	}
	node.Mode = mode

	return nil
}

// TODO: Avoid cyclical links
func (fs *FileSystem) fileStat(cwd, name string) (*inode.Inode, error) {
	name = inode.Abs(cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}

	if node.Mode&os.ModeSymlink == 0 {
		return node, nil
	}
	return fs.fileStat(filepath.Dir(name), fs.symlinks[node.Ino])
}

func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	if name == "/" {
		return &fileinfo{"/", fs.root}, nil
	}
	node, err := fs.fileStat(fs.cwd, name)
	return &fileinfo{filepath.Base(name), node}, err
}

func (fs *FileSystem) Lstat(name string) (os.FileInfo, error) {
	if name == "/" {
		return &fileinfo{"/", fs.root}, nil
	}
	name = inode.Abs(fs.cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return nil, &os.PathError{Op: "remove", Path: name, Err: err}
	}

	return &fileinfo{filepath.Base(name), node}, nil
}

func (fs *FileSystem) Lchown(name string, uid, gid int) error {
	if name == "/" {
		fs.root.Uid = uint32(uid)
		fs.root.Gid = uint32(gid)
		return nil
	}
	name = inode.Abs(fs.cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return err
	}

	node.Uid = uint32(uid)
	node.Gid = uint32(gid)
	return nil
}

func (fs *FileSystem) Readlink(name string) (string, error) {
	var ino uint64
	if name == "/" {
		ino = fs.root.Ino
	} else {
		node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
		if err != nil {
			return "", err
		}
		ino = node.Ino
	}

	fs.RLock()
	defer fs.Unlock()

	return fs.symlinks[ino], nil
}

func (fs *FileSystem) Symlink(oldname, newname string) error {
	wd := fs.root
	if !filepath.IsAbs(newname) {
		wd = fs.dir
	}
	var exists bool
	newNode, err := wd.Resolve(newname)
	if err == nil {
		exists = true
	}

	if exists && newNode.Mode&os.ModeSymlink == 0 {
		return &os.PathError{Op: "symlink", Path: newname, Err: syscall.EEXIST}
	}
	oldNode, err := wd.Resolve(oldname)
	if err != nil {
		return &os.PathError{Op: "symlink", Path: oldname, Err: syscall.ENOENT}
	}

	fs.Lock()
	defer fs.Unlock()

	if exists {
		newNode.Mode = oldNode.Mode | os.ModeSymlink
		fs.symlinks[newNode.Ino] = oldname
		return nil
	}

	dir, filename := filepath.Split(newname)
	dir = filepath.Clean(dir)
	parent, err := wd.Resolve(dir)
	if err != nil {
		return err
	}

	newNode = fs.ino.New(oldNode.Mode | os.ModeSymlink)

	err = parent.Link(filename, newNode)
	if err != nil {
		return &os.PathError{Op: "symlink", Path: newname, Err: err}
	}
	fs.symlinks[newNode.Ino] = oldname
	return nil
}

func (fs *FileSystem) Walk(name string, fn pathfilepath.WalkFunc) error {
	var stack []string
	push := func(path string) {
		stack = append(stack, path)
	}
	pop := func() string {
		path := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return path
	}

	push(name)
	for len(stack) > 0 {
		path := pop()
		info, err := fs.Stat(path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			f, err := fs.Open(path)
			if err != nil {
				return err
			}

			names, err := f.Readdirnames(-1)
			f.Close()
			if err != nil {
				return err
			}

			sort.Sort(sort.Reverse(sort.StringSlice(names)))
			for _, p := range names {
				if p == ".." || p == "." {
					continue
				}
				push(filepath.Join(path, p))
			}
		}

		err = fn(path, info, nil)
		if err != nil {
			return err
		}

	}
	return nil
}
