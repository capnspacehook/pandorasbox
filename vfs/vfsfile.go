package vfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/awnumar/fastrand"
	"github.com/awnumar/memguard"
	"github.com/awnumar/memguard/core"

	"github.com/capnspacehook/pandorasbox/absfs"
	"github.com/capnspacehook/pandorasbox/inode"
)

const keySize = 32

type File struct {
	sync.RWMutex

	fs *FileSystem

	name  string
	flags int
	node  *inode.Inode
	data  *sealedFile

	offset    int64
	diroffset int
}

type sealedFile struct {
	f *File

	ciphertext []byte
	size       int64
	key        *memguard.Enclave
}

func (s *sealedFile) updateSize() {
	if len(s.ciphertext) == 0 {
		s.size = 0
		return
	}

	s.size = int64(len(s.ciphertext) - core.Overhead)
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Read(p []byte) (int, error) {
	if f.flags == 3712 {
		return 0, io.EOF
	}
	if f.flags&absfs.O_ACCESS == os.O_WRONLY {
		return 0, &os.PathError{Op: "read", Path: f.name, Err: syscall.EBADF} //os.ErrPermission
	}
	if f.node.IsDir() && atomic.LoadInt64(&f.data.size) == 0 {
		return 0, &os.PathError{Op: "read", Path: f.name, Err: syscall.EISDIR} //os.ErrPermission
	}
	if atomic.LoadInt64(&f.offset) >= atomic.LoadInt64(&f.data.size) {
		return 0, io.EOF
	}

	var (
		err error
		key *memguard.LockedBuffer
	)

	f.RLock()
	if f.data.size != 0 {
		key, err = f.data.key.Open()
		if err != nil {
			return 0, err
		}
	} else {
		return 0, io.EOF
	}

	plaintext := make([]byte, f.data.size)
	_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
	key.Destroy()
	f.RUnlock()
	if err != nil {
		return 0, err
	}

	n := len(plaintext[f.offset:])
	core.Copy(p, plaintext[f.offset:])
	core.Wipe(plaintext)
	f.offset += int64(n)

	return n, nil
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	if f.flags&absfs.O_ACCESS == os.O_WRONLY {
		return 0, os.ErrPermission
	}
	atomic.StoreInt64(&f.offset, off)
	return f.Read(b)
}

func (f *File) Write(p []byte) (int, error) {
	if f.flags&absfs.O_ACCESS == os.O_RDONLY {
		return 0, &os.PathError{Op: "write", Path: f.name, Err: syscall.EBADF}
	}

	var (
		err       error
		plaintext = make([]byte, f.data.size)
	)

	f.Lock()
	defer f.Unlock()

	if f.data.size != 0 {
		key, err := f.data.key.Open()
		if err != nil {
			return 0, err
		}
		_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return 0, err
		}
		key.Destroy()
	}

	data := plaintext
	size := len(p) + int(f.offset)
	if int64(size) > f.data.size {
		data = make([]byte, size)
		core.Copy(data, plaintext)
	}
	core.Wipe(plaintext)

	core.Copy(data[int(f.offset):], p)
	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))
	f.data.ciphertext, err = core.Encrypt(data, newKey.Bytes())
	f.data.key = newKey.Seal()
	core.Wipe(data)
	f.data.updateSize()
	if err != nil {
		return 0, err
	}

	n := len(p)
	f.offset += int64(n)

	return n, nil
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	atomic.StoreInt64(&f.offset, off)
	return f.Write(b)
}

func (f *File) Close() error {
	err := f.Sync()
	if err != nil {
		return err
	}

	f.node = nil
	return nil
}

func (f *File) Seek(offset int64, whence int) (ret int64, err error) {
	switch whence {
	case io.SeekStart:
		atomic.StoreInt64(&f.offset, offset)
	case io.SeekCurrent:
		atomic.AddInt64(&f.offset, offset)
	case io.SeekEnd:
		atomic.StoreInt64(&f.offset, atomic.LoadInt64(&f.data.size)+offset)
	}
	if f.offset < 0 {
		atomic.StoreInt64(&f.offset, 0)
	}
	return f.offset, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return &fileinfo{filepath.Base(f.name), f.node}, nil
}

func (f *File) Sync() error {
	if f.flags&absfs.O_ACCESS == os.O_RDONLY {
		return nil
	}
	f.fs.Lock()
	f.fs.data[int(f.node.Ino)] = f.data
	f.fs.Unlock()
	f.node.Size = atomic.LoadInt64(&f.data.size)

	return nil
}

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	if f.flags&absfs.O_ACCESS == os.O_WRONLY {
		return nil, os.ErrPermission
	}
	if !f.node.IsDir() {
		return nil, errors.New("not a directory")
	}

	f.Lock()
	defer f.Unlock()

	dirs := f.node.Dir
	if f.diroffset >= len(dirs) {
		return nil, io.EOF
	}
	if n < 1 {
		n = len(dirs)
		f.diroffset = 0
	}
	infos := make([]os.FileInfo, n-f.diroffset)
	for i, entry := range dirs[f.diroffset:n] {
		infos[i] = &fileinfo{entry.Name, entry.Inode}
	}
	f.diroffset += n
	return infos, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	var list []string
	if f.flags&absfs.O_ACCESS == os.O_WRONLY {
		return list, os.ErrPermission
	}
	if !f.node.IsDir() {
		return list, errors.New("not a directory")
	}

	f.Lock()
	defer f.Unlock()

	dirs := f.node.Dir
	if f.diroffset >= len(dirs) {
		return list, io.EOF
	}
	if n < 1 {
		n = len(dirs)
	}
	list = make([]string, n-f.diroffset)
	for i, entry := range dirs[f.diroffset:n] {
		list[i] = entry.Name
	}
	f.diroffset += n
	return list, nil
}

func (f *File) Truncate(size int64) error {
	if f.flags&absfs.O_ACCESS == os.O_RDONLY {
		return os.ErrPermission
	}
	if size < 0 {
		return &os.PathError{Op: "truncate", Path: f.name, Err: os.ErrClosed}
	}

	f.Lock()
	defer f.Unlock()

	var (
		err       error
		plaintext []byte
	)

	if f.data.size != 0 {
		key, err := f.data.key.Open()
		if err != nil {
			return err
		}
		plaintext = make([]byte, f.data.size)
		_, err = core.Decrypt(f.data.ciphertext, key.Bytes(), plaintext)
		if err != nil {
			return err
		}
		key.Destroy()
	} else if size == 0 { // data is already nil, no-op
		return nil
	}

	// TODO: should this be copied in constant time?
	if size <= f.data.size {
		plaintext = plaintext[:int(size)]
		newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))
		f.data.ciphertext, err = core.Encrypt(plaintext, newKey.Bytes())
		f.data.key = newKey.Seal()
		core.Wipe(plaintext)
		f.data.updateSize()
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
	f.data.updateSize()
	if err != nil {
		return err
	}

	return nil
}

func (f *File) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}

type fileinfo struct {
	name string
	node *inode.Inode
}

func (i *fileinfo) Name() string {
	return i.name
}

func (i *fileinfo) Size() int64 {
	return i.node.Size
}

func (i *fileinfo) ModTime() time.Time {
	return i.node.Mtime
}

func (i *fileinfo) Mode() os.FileMode {
	return i.node.Mode
}

func (i *fileinfo) Sys() interface{} {
	return i.node
}

func (i *fileinfo) IsDir() bool {
	return i.node.IsDir()
}
