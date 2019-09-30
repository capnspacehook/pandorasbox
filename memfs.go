package pandorasbox

import (
	"errors"
	"os"
	filepath "path" // force forward slash separators on all OSs.
	pathfilepath "path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type MemFileSystem struct {
	Umask   os.FileMode
	Tempdir string

	root *Inode
	cwd  string
	dir  *Inode
	ino  *Ino

	symlinks map[uint64]string
	data     [][]byte
}

func NewMemFS() (*MemFileSystem, error) {
	fs := new(MemFileSystem)
	fs.ino = new(Ino)
	fs.Tempdir = "/tmp"

	fs.Umask = 0755
	fs.root = fs.ino.NewDir(fs.Umask)
	fs.cwd = "/"
	fs.dir = fs.root
	fs.data = make([][]byte, 2)
	fs.symlinks = make(map[uint64]string)
	return fs, nil
}

func (fs *MemFileSystem) Separator() uint8 {
	return '/'
}

func (fs *MemFileSystem) ListSeparator() uint8 {
	return ':'
}

func (fs *MemFileSystem) Rename(oldpath, newpath string) error {
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

func (fs *MemFileSystem) Chdir(name string) (err error) {
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

func (fs *MemFileSystem) Getwd() (dir string, err error) {
	return fs.cwd, nil
}

func (fs *MemFileSystem) TempDir() string {
	return fs.Tempdir
}

func (fs *MemFileSystem) Open(name string) (File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *MemFileSystem) Create(name string) (File, error) {
	return fs.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
}

func (fs *MemFileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if name == "/" {
		data := fs.data[int(fs.root.Ino)]
		return &MemFile{fs: fs, name: name, flags: flag, node: fs.root, data: data}, nil
	}
	if name == "." {
		data := fs.data[int(fs.dir.Ino)]
		return &MemFile{fs: fs, name: name, flags: flag, node: fs.dir, data: data}, nil
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

	access := flag & O_ACCESS
	create := flag&os.O_CREATE != 0
	truncate := flag&os.O_TRUNC != 0

	// error if it does not exist, and we are not allowed to create it.
	if !exists && !create {
		return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	if exists {
		// err if exclusive create is required
		if create && flag&os.O_EXCL != 0 {
			return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.EEXIST}
		}
		if node.IsDir() {
			if access != os.O_RDONLY || truncate {
				return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.EISDIR} // os.ErrNotExist}
			}
		}

		// if we must truncate the file
		if truncate {
			fs.data[int(node.Ino)] = fs.data[int(node.Ino)][:0]
		}

	} else { // !exists
		// error if we cannot create the file
		if !create {
			return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT} //os.ErrNotExist}
		}

		// Create write-able file
		node = fs.ino.New(fs.Umask & perm)
		err := parent.Link(filename, node)
		if err != nil {
			return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: err}
		}
		fs.data = append(fs.data, []byte{})
	}
	data := fs.data[int(node.Ino)]

	if !create {
		if access == os.O_RDONLY && node.Mode&OS_ALL_R == 0 ||
			access == os.O_WRONLY && node.Mode&OS_ALL_W == 0 ||
			access == os.O_RDWR && node.Mode&(OS_ALL_W|OS_ALL_R) == 0 {
			return &InvalidFile{name}, &os.PathError{Op: "open", Path: name, Err: os.ErrPermission}
		}
	}
	return &MemFile{fs: fs, name: name, flags: flag, node: node, data: data}, nil
}

func (fs *MemFileSystem) Truncate(name string, size int64) error {
	path := Abs(fs.cwd, name)
	child, err := fs.root.Resolve(path)
	if err != nil {
		return err
	}

	i := int(child.Ino)
	if size <= child.Size {
		fs.data[i] = fs.data[i][:int(size)]
		return nil
	}
	data := make([]byte, int(size))
	copy(data, fs.data[i])
	fs.data[i] = data
	return nil
}

func (fs *MemFileSystem) Mkdir(name string, perm os.FileMode) error {
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
	fs.data = append(fs.data, []byte{})
	return nil
}

func (fs *MemFileSystem) MkdirAll(name string, perm os.FileMode) error {
	name = Abs(fs.cwd, name)
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

func (fs *MemFileSystem) Remove(name string) (err error) {
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

func (fs *MemFileSystem) RemoveAll(name string) error {
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
func (fs *MemFileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	var err error
	node := fs.root

	name = Abs(fs.cwd, name)
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
func (fs *MemFileSystem) Chown(name string, uid, gid int) error {
	var err error
	node := fs.root

	name = Abs(fs.cwd, name)
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
func (fs *MemFileSystem) Chmod(name string, mode os.FileMode) error {
	var err error
	node := fs.root

	name = Abs(fs.cwd, name)

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
func (fs *MemFileSystem) fileStat(cwd, name string) (*Inode, error) {
	name = Abs(cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}

	if node.Mode&os.ModeSymlink == 0 {
		return node, nil
	}
	return fs.fileStat(filepath.Dir(name), fs.symlinks[node.Ino])
}

func (fs *MemFileSystem) Stat(name string) (os.FileInfo, error) {
	if name == "/" {
		return &fileinfo{"/", fs.root}, nil
	}
	node, err := fs.fileStat(fs.cwd, name)
	return &fileinfo{filepath.Base(name), node}, err
}

func (fs *MemFileSystem) Lstat(name string) (os.FileInfo, error) {
	if name == "/" {
		return &fileinfo{"/", fs.root}, nil
	}
	name = Abs(fs.cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return nil, &os.PathError{Op: "remove", Path: name, Err: err}
	}

	return &fileinfo{filepath.Base(name), node}, nil
}

func (fs *MemFileSystem) Lchown(name string, uid, gid int) error {
	if name == "/" {
		fs.root.Uid = uint32(uid)
		fs.root.Gid = uint32(gid)
		return nil
	}
	name = Abs(fs.cwd, name)
	node, err := fs.root.Resolve(strings.TrimLeft(name, "/"))
	if err != nil {
		return err
	}

	node.Uid = uint32(uid)
	node.Gid = uint32(gid)
	return nil
}

func (fs *MemFileSystem) Readlink(name string) (string, error) {
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

	return fs.symlinks[ino], nil
}

func (fs *MemFileSystem) Symlink(oldname, newname string) error {
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

func (fs *MemFileSystem) Walk(name string, fn pathfilepath.WalkFunc) error {
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
