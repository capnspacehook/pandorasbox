package vfs

import (
	"fmt"
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

type vfsFile struct {
	fs *virtualFS

	// protects dirOffset
	sync.Mutex

	name   string
	flags  int
	node   *inode.Inode
	closed atomic.Bool

	sfile *sealedFile

	offset    atomic.Int64
	dirOffset int
}

// sealedFile contains authenticated and encrypted file contents, as
// well as a key used to decrypt the file contents
type sealedFile struct {
	// protects ciphertext and sealedKey; since sealedFiles are shared
	// between multiple files this ensures read/write operations
	// don't race
	sync.RWMutex

	ciphertext []byte
	sealedKey  *memguard.Enclave
}

func (s *sealedFile) size() int {
	return max(len(s.ciphertext)-core.Overhead, 0)
}

func (f *vfsFile) Name() string {
	return f.name
}

func (f *vfsFile) setOffset(offset int64) {
	if offset < 0 {
		panic(fmt.Sprintf("%s: negative offset: %d", f.name, offset))
	}
	f.offset.Store(offset)
}

func (f *vfsFile) addOffset(offset int64) int64 {
	newOffset := f.offset.Add(offset)
	if newOffset < 0 {
		panic(fmt.Sprintf("%s: negative offset: %d", f.name, newOffset))
	}
	return newOffset
}

// decrypt returns the plaintext of the sealed file. It must be called
// under lock.
func (f *vfsFile) decrypt(plaintext []byte) error {
	key, err := f.sfile.sealedKey.Open()
	if err != nil {
		return err
	}
	_, err = core.Decrypt(f.sfile.ciphertext, key.Bytes(), plaintext)
	key.Destroy()
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	return nil
}

// encrypt encrypts plaintext, stores the ciphertext in the sealed file
// and updates the file size. It must be called under lock.
func (f *vfsFile) encrypt(plaintext []byte) error {
	var err error

	newKey := memguard.NewBufferFromBytes(fastrand.Bytes(keySize))
	f.sfile.ciphertext, err = core.Encrypt(plaintext, newKey.Bytes())
	core.Wipe(plaintext)
	if err != nil {
		return fmt.Errorf("failed to enrypt: %w", err)
	}

	newKey.Freeze()
	f.sfile.sealedKey = newKey.Seal()
	f.node.Size = int64(len(plaintext))

	return nil
}

func (f *vfsFile) Read(p []byte) (int, error) {
	n, err := f.read(p, f.offset.Load())
	f.addOffset(int64(n))

	return n, err
}

func (f *vfsFile) read(p []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	if f.closed.Load() {
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

	f.node.RLock()
	defer f.node.RUnlock()

	if offset >= f.node.Size {
		return 0, io.EOF
	}
	if f.node.Size == 0 {
		return 0, io.EOF
	}

	f.sfile.RLock()
	defer f.sfile.RUnlock()

	plaintext := make([]byte, f.sfile.size())
	if err := f.decrypt(plaintext); err != nil {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: err}
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

func (f *vfsFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.read(p, off)
}

func (f *vfsFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.closed.Load() {
		return nil, &fs.PathError{Op: "readdir", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_WRONLY {
		return nil, &fs.PathError{Op: "readat", Path: f.name, Err: fs.ErrPermission}
	}
	if !f.node.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: f.Name(), Err: syscall.ENOTDIR}
	}

	// protect f.dirOffset
	f.Lock()
	defer f.Unlock()

	// protect f.node.Dir
	f.node.RLock()
	defer f.node.RUnlock()

	dirs := f.node.Dir
	if f.dirOffset >= len(dirs) {
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
	if f.dirOffset == 0 {
		f.dirOffset = 2
	}

	infosLen := n - f.dirOffset
	if infosLen <= 0 {
		infosLen = n
	}

	infos := make([]fs.DirEntry, infosLen)
	for i, entry := range dirs[f.dirOffset:] {
		if i == n {
			break
		}

		infos[i] = &DirEntry{entry.Name, entry.Inode}
	}
	f.dirOffset += n

	return infos, nil
}

func (f *vfsFile) Write(p []byte) (int, error) {
	n, err := f.write(p, f.offset.Load())
	f.addOffset(int64(n))

	return n, err
}

func (f *vfsFile) write(p []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrInvalid}
	}
	if f.closed.Load() {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: syscall.EISDIR}
	}
	// writing past the end of the file is allowed as part of the POSIX spec
	// and we want to be roughly compatible with that, so we allow it too

	f.node.Lock()
	defer f.node.Unlock()

	f.sfile.Lock()
	defer f.sfile.Unlock()

	size := f.sfile.size()
	if writeSize := len(p) + int(offset); writeSize > size {
		size = writeSize
	}
	plaintext := make([]byte, size)
	if len(f.sfile.ciphertext) > 0 {
		if err := f.decrypt(plaintext); err != nil {
			return 0, &fs.PathError{Op: "write", Path: f.name, Err: err}
		}
	}

	core.Copy(plaintext[offset:], p)
	err := f.encrypt(plaintext)
	if err != nil {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: err}
	}

	var n int
	if len(p) < len(plaintext[offset:]) {
		n = len(p)
	} else {
		n = len(plaintext[offset:])
	}

	return n, nil
}

func (f *vfsFile) WriteAt(b []byte, off int64) (n int, err error) {
	return f.write(b, off)
}

func (f *vfsFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}

func (f *vfsFile) Stat() (os.FileInfo, error) {
	if f.closed.Load() {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: fs.ErrClosed}
	}

	return &FileInfo{filepath.Base(f.name), f.node}, nil
}

func (f *vfsFile) Seek(offset int64, whence int) (int64, error) {
	if f.closed.Load() {
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

		f.setOffset(offset)
		ret = offset
	case io.SeekCurrent:
		f.node.RLock()
		if offset < 0 && (f.node.Size+offset) < 0 {
			f.node.RUnlock()
			return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrInvalid}
		}
		f.node.RUnlock()

		ret = f.addOffset(offset)
	case io.SeekEnd:
		f.node.RLock()
		ret = f.node.Size + offset
		f.node.RUnlock()
		f.setOffset(ret)
	}

	return ret, nil
}

func (f *vfsFile) Sync() error {
	return nil
}

func (f *vfsFile) Truncate(size int64) error {
	if size < 0 {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrInvalid}
	}
	if f.closed.Load() {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrClosed}
	}
	if f.flags&_O_ACCESS == os.O_RDONLY {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: fs.ErrPermission}
	}
	if f.node.IsDir() {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: syscall.EISDIR}
	}

	// protect f.node.Size
	f.node.Lock()
	defer f.node.Unlock()

	if f.node.Size == size {
		return nil
	}
	if f.node.Size == 0 && size == 0 {
		// file is already empty, no-op
		return nil
	}

	f.sfile.Lock()
	defer f.sfile.Unlock()

	if f.node.Size == 0 {
		// the file is empty and we are extending the file
		data := make([]byte, size)
		if err := f.encrypt(data); err != nil {
			return &fs.PathError{Op: "truncate", Path: f.name, Err: err}
		}
		return nil
	} else if size == 0 {
		// the file is not empty and we are making it empty
		f.sfile.ciphertext = nil
		f.sfile.sealedKey = nil
		f.node.Size = 0
		return nil
	}

	// shrink or extend the file
	plaintext := make([]byte, f.sfile.size())
	if err := f.decrypt(plaintext); err != nil {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: err}
	}

	data := make([]byte, size)
	core.Move(data, plaintext)

	if err := f.encrypt(data); err != nil {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: err}
	}

	return nil
}

func (f *vfsFile) Close() error {
	if f.closed.Load() {
		return fs.ErrClosed
	}

	f.closed.Store(true)

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
	i.node.RLock()
	defer i.node.RUnlock()

	return i.node.Size
}

func (i *FileInfo) Mode() os.FileMode {
	return i.node.Mode
}

func (i *FileInfo) ModTime() time.Time {
	i.node.RLock()
	defer i.node.RUnlock()

	return i.node.Mtime
}

func (i *FileInfo) IsDir() bool {
	return i.node.IsDir()
}

func (i *FileInfo) Sys() interface{} {
	return i.node
}
