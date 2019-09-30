package pandorasbox

import (
	"os"
	"strings"
)

const (
	O_ACCESS = 0x3 // masks the access mode (O_RDONLY, O_WRONLY, or O_RDWR)

	// Exactly one of O_RDONLY, O_WRONLY, or O_RDWR must be specified.
	O_RDONLY int = os.O_RDONLY // open the file read-only.
	O_WRONLY int = os.O_WRONLY // open the file write-only.
	O_RDWR   int = os.O_RDWR   // open the file read-write.

	// The remaining values may be or'ed in to control behavior.
	O_APPEND int = os.O_APPEND // append data to the file when writing.
	O_CREATE int = os.O_CREATE // create a new file if none exists.
	O_EXCL   int = os.O_EXCL   // used with O_CREATE, file must not exist.
	O_SYNC   int = os.O_SYNC   // open for synchronous I/O.
	O_TRUNC  int = os.O_TRUNC  // if possible, truncate file when opened.
)

type Flags int

func (f Flags) String() string {
	var out []string
	flags := int(f)
	switch flags & O_ACCESS {
	case O_RDONLY:
		out = append(out, "O_RDONLY")
	case O_RDWR:
		out = append(out, "O_RDWR")
	case O_WRONLY:
		out = append(out, "O_WRONLY")
	}

	names := []string{"O_APPEND", "O_CREATE", "O_EXCL", "O_SYNC", "O_TRUNC"}
	for i, flag := range []int{O_APPEND, O_CREATE, O_EXCL, O_SYNC, O_TRUNC} {
		if (flag & flags) != 0 {
			out = append(out, names[i])
		}
	}
	return strings.Join(out, "|")
}
