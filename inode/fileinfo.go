package inode

import (
	"os"
	"time"
)

// Stat - implements the os.FileInfo interface
type Stat struct {
	Filename string
	Node     *Inode
}

// base name of the file
func (i *Stat) Name() string {
	return i.Filename
}

// length in bytes for regular files; system-dependent for others
func (i *Stat) Size() int64 {
	return i.Node.Size
}

// file mode bits
func (i *Stat) Mode() os.FileMode {
	return i.Node.Mode
}

// modification time
func (i *Stat) ModTime() time.Time {
	return i.Node.Mtime
}

// abbreviation for Mode().IsDir()
func (i *Stat) IsDir() bool {
	return i.Mode()&os.ModeDir != 0
}

// underlying data source (can return nil)
func (i *Stat) Sys() interface{} {
	return i.Node
}
