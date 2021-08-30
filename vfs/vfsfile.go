package vfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/awnumar/fastrand"
	"github.com/awnumar/memguard"
	"github.com/awnumar/memguard/core"

	"github.com/capnspacehook/pandorasbox/inode"
)

const keySize = 32

type file struct {
	mtx sync.RWMutex

	fs *pbFS

	name  string
	flags int
	node  *inode.Inode
	data  *sealedFile

	offset    int64
	diroffset int
}

type sealedFile struct {
	f *file

	ciphertext []byte
	key        *memguard.Enclave
}

func (f *file) updateSize() {
	if len(f.data.ciphertext) == 0 {
		f.node.Size = 0
		return
	}

	f.node.Size = int64(len(f.data.ciphertext) - core.Overhead)
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Read(p []byte) (int, error) {
	n, err := f.read(p, atomic.LoadInt64(&f.offset))
	atomic.AddInt64(&f.offset, int64(n))

	return n, err
}

func (f *file) read(p []byte, offset int64) (int, error) {
	if f.node == nil {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrClosed}
	}
	if len(p) == 0 {
		return 0, nil
	}
	if f.flags&_O_ACCESS == os.O_WRONLY {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: syscall.EISDIR}
	}
	if offset >= atomic.LoadInt64(&f.node.Size) {
		return 0, io.EOF
	}

	var (
		err error
		key *memguard.LockedBuffer
	)

	if atomic.LoadInt64(&f.node.Size) != 0 {
		f.mtx.RLock()
		key, err = f.data.key.Open()
		f.mtx.RUnlock()
		if err != nil {
			return 0, err
		}
	} else {
		return 0, io.EOF
	}

	plaintext := make([]byte, f.node.Size)
	f.mtx.RLock()
	_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
	key.Destroy()
	f.mtx.RUnlock()
	if err != nil {
		return 0, err
	}

	core.Copy(p, plaintext[offset:])
	core.Wipe(plaintext)

	var n int
	if len(p) < len(plaintext[offset:]) {
		n = len(p)
	} else {
		n = len(plaintext[offset:])
	}

	if len(p) > n {
		return n, io.EOF
	}

	return n, nil
}

func (f *file) ReadAt(b []byte, off int64) (n int, err error) {
	if f.node == nil {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: fs.ErrClosed}
	}
	if off < 0 {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: errors.New("negative offset")}
	}
	if f.flags&_O_ACCESS == os.O_WRONLY {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: syscall.EISDIR}
	}

	return f.read(b, off)
}

func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.node == nil {
		return nil, &fs.PathError{Op: "readdir", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_WRONLY {
		return nil, &fs.PathError{Op: "readat", Path: f.name, Err: fs.ErrPermission}
	}
	if !f.node.IsDir() {
		// TODO: is this the correct error?
		return nil, &fs.PathError{Op: "readdir", Path: f.Name(), Err: syscall.ENOTDIR}
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()

	dirs := f.node.Dir
	if f.diroffset >= len(dirs) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}

	if n <= 0 {
		// if there are only 2 dirs ('.' and '..'), return
		// since we are skipping them below
		if len(dirs) == 2 {
			return nil, nil
		}
		n = len(dirs)
	}
	// skip '.' and '..' to retain compatibility with os.ReadDir
	if f.diroffset == 0 {
		f.diroffset = 2
	}

	infosLen := n - f.diroffset
	if infosLen <= 0 {
		infosLen = n
	}

	infos := make([]fs.DirEntry, infosLen)
	for i, entry := range dirs[f.diroffset:] {
		if i == n {
			break
		}

		infos[i] = &DirEntry{entry.Name, entry.Inode}
	}
	f.diroffset += n

	return infos, nil
}

func (f *file) Write(p []byte) (int, error) {
	n, err := f.write(p, atomic.LoadInt64(&f.offset))
	atomic.AddInt64(&f.offset, int64(n))

	return n, err
}

func (f *file) write(p []byte, offset int64) (int, error) {
	if f.node == nil {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: syscall.EISDIR}
	}

	var (
		err       error
		plaintext = make([]byte, f.node.Size)
	)

	if atomic.LoadInt64(&f.node.Size) != 0 {
		f.mtx.RLock()
		key, err := f.data.key.Open()
		if err != nil {
			return 0, err
		}
		_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return 0, err
		}
		key.Destroy()
		f.mtx.RUnlock()

	}

	data := plaintext
	size := len(p) + int(offset)
	if int64(size) > f.node.Size {
		data = make([]byte, size)
		core.Copy(data, plaintext)
	}

	core.Copy(data[offset:], p)
	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))

	f.mtx.Lock()
	f.data.ciphertext, err = core.Encrypt(data, newKey.Bytes())
	f.data.key = newKey.Seal()
	f.updateSize()
	core.Wipe(data)
	f.mtx.Unlock()

	if err != nil {
		return 0, err
	}

	var n int
	if len(p) < len(data[offset:]) {
		n = len(p)
	} else {
		n = len(data[offset:])
	}

	return n, nil
}

func (f *file) WriteAt(b []byte, off int64) (n int, err error) {
	if f.node == nil {
		return 0, &fs.PathError{Op: "writeat", Path: f.name, Err: fs.ErrClosed}
	}
	if off < 0 {
		return 0, &fs.PathError{Op: "writeat", Path: f.name, Err: errors.New("negative offset")}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return 0, fs.ErrPermission
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "writeat", Path: f.name, Err: syscall.EISDIR}
	}

	return f.write(b, off)
}

func (f *file) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}

func (f *file) Stat() (os.FileInfo, error) {
	if f.node == nil {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: fs.ErrClosed}
	}

	return &FileInfo{filepath.Base(f.name), f.node}, nil
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f.node == nil {
		return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrClosed}
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "seek", Path: f.name, Err: syscall.EISDIR}
	}

	var ret int64
	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrInvalid}
		}

		atomic.StoreInt64(&f.offset, offset)
		ret = offset
	case io.SeekCurrent:
		if offset < 0 && (atomic.LoadInt64(&f.node.Size)+offset) < 0 {
			return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrInvalid}
		}

		ret = atomic.AddInt64(&f.offset, offset)
	case io.SeekEnd:
		ret = atomic.LoadInt64(&f.node.Size) + offset
		atomic.StoreInt64(&f.offset, ret)
	}

	return ret, nil
}

func (f *file) Sync() error {
	if f.node == nil {
		return &fs.PathError{Op: "sync", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return nil
	}

	f.fs.mtx.Lock()
	f.fs.data[int(f.node.Ino)] = f.data
	f.fs.mtx.Unlock()

	return nil
}

func (f *file) Truncate(size int64) error {
	if f.node == nil {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: syscall.EISDIR}
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()

	var (
		err       error
		plaintext []byte
	)

	if f.node.Size != 0 {
		key, err := f.data.key.Open()
		if err != nil {
			return err
		}
		plaintext = make([]byte, f.node.Size)
		_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return err
		}
		key.Destroy()
	} else if size == 0 { // data is already nil, no-op
		return nil
	}

	// TODO: should this be copied in constant time?
	if size <= f.node.Size {
		plaintext = plaintext[:int(size)]
		newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))
		f.data.ciphertext, err = core.Encrypt(plaintext, newKey.Bytes())
		f.data.key = newKey.Seal()
		core.Wipe(plaintext)
		f.updateSize()
		if err != nil {
			return err
		}
		return nil
	}

	data := make([]byte, int(size))
	core.Move(data, plaintext)

	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))
	f.data.ciphertext, err = core.Encrypt(data, newKey.Bytes())
	f.data.key = newKey.Seal()
	core.Wipe(data)
	f.updateSize()
	if err != nil {
		return err
	}

	return nil
}

func (f *file) Close() error {
	if err := f.Sync(); err != nil {
		return err
	}
	f.mtx.Lock()
	f.node = nil
	f.mtx.Unlock()

	return nil
}

type DirEntry struct {
	name string
	node *inode.Inode
}

func (d *DirEntry) Name() string {
	return d.name
}

func (d *DirEntry) IsDir() bool {
	return d.node.Mode.IsDir()
}

func (d *DirEntry) Type() fs.FileMode {
	return d.node.Mode.Type()
}

func (d *DirEntry) Info() (fs.FileInfo, error) {
	return &FileInfo{name: d.name, node: d.node}, nil
}

type FileInfo struct {
	name string
	node *inode.Inode
}

func (i *FileInfo) Name() string {
	return i.name
}

func (i *FileInfo) Size() int64 {
	return i.node.Size
}

func (i *FileInfo) Mode() os.FileMode {
	return i.node.Mode
}

func (i *FileInfo) ModTime() time.Time {
	return i.node.Mtime
}

func (i *FileInfo) IsDir() bool {
	return i.node.IsDir()
}

func (i *FileInfo) Sys() interface{} {
	return i.node
}
