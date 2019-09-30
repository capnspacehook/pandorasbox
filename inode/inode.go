package inode

import (
	"errors"
	"fmt"
	"os"
	filepath "path"
	"sort"
	"strings"
	"syscall"
	"time"
)

// An Inode represents the basic metadata of a file.
type Inode struct {
	Ino   uint64
	Mode  os.FileMode
	Nlink uint64
	Size  int64

	Ctime time.Time // creation time
	Atime time.Time // access time
	Mtime time.Time // modification time
	Uid   uint32
	Gid   uint32

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

func (e *DirEntry) String() string {
	nodeStr := "(nil)"
	if e.Inode != nil {
		nodeStr = fmt.Sprintf("{Ino:%d ...}", e.Inode.Ino)
	}
	return fmt.Sprintf("entry{%q, inode%s", e.Name, nodeStr)
}

type Directory []*DirEntry

func (d Directory) Len() int           { return len(d) }
func (d Directory) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d Directory) Less(i, j int) bool { return d[i].Name < d[j].Name }

func (n *Inode) String() string {
	if n == nil {
		return "<nil>"
	}

	list := make([]string, len(n.Dir))
	for i, e := range n.Dir {
		list[i] = e.String()
	}
	return fmt.Sprintf("Inode{Ino:%d,Mode:%s,Nlink:%d}\n\t%s", n.Ino, n.Mode, n.Nlink, strings.Join(list, ",\n"))
}

type Ino uint64

func (n *Ino) New(mode os.FileMode) *Inode {
	*n++
	now := time.Now()
	return &Inode{
		Ino:   uint64(*n),
		Atime: now,
		Mtime: now,
		Ctime: now,
		Mode:  mode,
	}
}

func (n *Ino) NewDir(mode os.FileMode) *Inode {
	dir := n.New(mode)
	var err error
	dir.Mode = os.ModeDir | mode
	err = dir.Link(".", dir)
	if err != nil {
		panic(err)
	}
	err = dir.Link("..", dir)
	if err != nil {
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

	x := n.find(name)

	entry := &DirEntry{name, child}

	if x < len(n.Dir) && n.Dir[x].Name == name {
		n.linkswapi(x, entry)
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

	x := n.find(name)

	if x == n.Dir.Len() || n.Dir[x].Name != name {
		return syscall.ENOENT // os.ErrNotExist
	}

	n.unlinki(x)
	return nil
}

func (n *Inode) UnlinkAll() {
	for _, e := range n.Dir {
		if e.Name == ".." {
			continue
		}
		if e.Inode.Ino == n.Ino {
			e.Inode.countDown()
			continue
		}
		e.Inode.UnlinkAll()
		e.Inode.countDown()
	}
	n.Dir = n.Dir[:0]
}

func (n *Inode) IsDir() bool {
	return os.ModeDir&n.Mode != 0
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
	if (err == nil && !tnode.IsDir()) || (err != nil && os.IsNotExist(err)) {
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
		name, rename = rename, name
	}
	err = p.Unlink(name)
	if err != nil {
		return err
	}

	return nil
}

func (n *Inode) Resolve(path string) (*Inode, error) {
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
	return nil, syscall.ENOENT // os.ErrNotExist
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

func (n *Inode) linkswapi(i int, entry *DirEntry) {
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
