package inode

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	filepath "path" // force forward slash separators on all OSs
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// An Inode represents the basic metadata of a file.
type Inode struct {
	sync.RWMutex

	Ino   uint64      // should never change
	Mode  fs.FileMode // should never change
	Nlink uint64
	Size  int64

	Ctime time.Time // creation time
	Atime time.Time // access time
	Mtime time.Time // modification time

	Dir Directory
}

type DirEntry struct {
	Name  string
	Inode *Inode
}

func (e *DirEntry) IsDir() bool {
	if e.Inode == nil {
		return false
	}

	return e.Inode.IsDir()
}

type Directory []*DirEntry

func (d Directory) Len() int           { return len(d) }
func (d Directory) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d Directory) Less(i, j int) bool { return d[i].Name < d[j].Name }

type Ino uint64

func (n *Ino) New(mode os.FileMode) *Inode {
	atomic.AddUint64((*uint64)(unsafe.Pointer(n)), 1)
	now := time.Now()

	return &Inode{
		Ino:   uint64(*n),
		Atime: now,
		Mtime: now,
		Ctime: now,
		Mode:  mode,
	}
}

func (n *Ino) SubIno() {
	atomic.AddUint64((*uint64)(unsafe.Pointer(n)), ^uint64(0))
}

func (n *Ino) NewDir(mode os.FileMode) *Inode {
	dir := n.New(mode)
	dir.Mode = os.ModeDir | mode

	if err := dir.Link(".", dir); err != nil {
		panic(err)
	}
	if err := dir.Link("..", dir); err != nil {
		panic(err)
	}

	return dir
}

// Link - link adds a directory entry (DirEntry) for the given node (assumed to be a directory) to the provided child Inode.
func (n *Inode) Link(name string, child *Inode) error {
	// Return an error if a regular file is used as a link target
	if !n.IsDir() {
		return errors.New("not a directory")
	}

	n.Lock()
	defer n.Unlock()

	x := n.find(name)

	entry := &DirEntry{name, child}

	if x < len(n.Dir) && n.Dir[x].Name == name {
		n.linkSwapi(x, entry)
		return nil
	}
	n.linki(x, entry)

	return nil
}

// Unlink - removes the directory entry (DirEntry).
func (n *Inode) Unlink(name string) error {
	// It is an error to unlink an Inode that is not a directory
	if !n.IsDir() {
		return errors.New("not a directory")
	}

	n.Lock()
	defer n.Unlock()

	x := n.find(name)

	if x == n.Dir.Len() || n.Dir[x].Name != name {
		return fs.ErrNotExist
	}

	n.unlinki(x)

	return nil
}

func (n *Inode) UnlinkAll() {
	n.Lock()

	for _, e := range n.Dir {
		if e.Name == ".." {
			continue
		}
		if e.Inode.Ino == n.Ino {
			e.Inode.countDown()
			continue
		}

		n.Unlock()
		e.Inode.UnlinkAll()
		n.Lock()
		e.Inode.countDown()
	}

	n.Dir = n.Dir[:0]
	n.Unlock()
}

func (n *Inode) IsDir() bool {
	return n.Mode&fs.ModeDir != 0
}

func (n *Inode) Rename(oldpath, newpath string) error {
	dir, name := filepath.Split(oldpath)
	dir = filepath.Clean(dir)

	snode, err := n.Resolve(oldpath)
	if err != nil {
		return err
	}

	p, err := n.Resolve(dir)
	if err != nil {
		return err
	}

	var rename string
	tnode, err := n.Resolve(newpath)
	if err == nil && tnode.IsDir() {
		return fs.ErrExist
	}
	if (err == nil && !tnode.IsDir()) || (err != nil && errors.Is(err, fs.ErrNotExist)) {
		var tdir string
		tdir, rename = filepath.Split(newpath)
		tdir = filepath.Clean(tdir)
		tnode, err = n.Resolve(tdir)
	}
	if err != nil {
		return err
	}

	if len(rename) > 0 {
		name, rename = rename, name
	}
	err = tnode.Link(name, snode)
	if err != nil {
		return err
	}
	if len(rename) > 0 {
		name = rename
	}
	err = p.Unlink(name)
	if err != nil {
		return err
	}

	return nil
}

func (n *Inode) Resolve(path string) (*Inode, error) {
	n.RLock()
	defer n.RUnlock()

	name, trim := PopPath(path)
	if name == "/" {
		if trim == "" {
			return n, nil
		}
		nn, err := n.Resolve(trim)
		if err != nil {
			return nil, err
		}
		if nn == nil {
			return n, nil
		}
		return nn, err
	}

	x := n.find(name)
	if x < len(n.Dir) && n.Dir[x].Name == name {
		nn := n.Dir[x].Inode
		if len(trim) == 0 {
			return nn, nil
		}
		return nn.Resolve(trim)
	}

	return nil, fs.ErrNotExist
}

func (n *Inode) accessed() {
	n.Atime = time.Now()
}

func (n *Inode) modified() {
	now := time.Now()
	n.Atime = now
	n.Mtime = now
}

func (n *Inode) countUp() {
	n.Nlink++
	n.accessed() // (I don't think link count mod counts as node mod )
}

func (n *Inode) countDown() {
	if n.Nlink == 0 {
		panic(fmt.Sprintf("inode %d negative link count", n.Ino))
	}
	n.Nlink--
	n.accessed() // (I don't think link count mod counts as node mod )
}

func (n *Inode) unlinki(i int) {
	n.Dir[i].Inode.countDown()
	copy(n.Dir[i:], n.Dir[i+1:])
	n.Dir = n.Dir[:len(n.Dir)-1]

	n.modified()
}

func (n *Inode) linkSwapi(i int, entry *DirEntry) {
	n.Dir[i].Inode.countDown()
	n.Dir[i] = entry
	n.Dir[i].Inode.countUp()

	n.modified()
}

func (n *Inode) linki(i int, entry *DirEntry) {
	n.Dir = append(n.Dir, nil)
	copy(n.Dir[i+1:], n.Dir[i:])

	n.Dir[i] = entry
	n.Dir[i].Inode.countUp()

	n.modified()
}

func (n *Inode) find(name string) int {
	return sort.Search(len(n.Dir), func(i int) bool {
		return n.Dir[i].Name >= name
	})
}
