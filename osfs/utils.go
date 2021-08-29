package osfs

import (
	"os"
)

func IsPathSeparator(c uint8) bool {
	return os.IsPathSeparator(c)
}

func SameFile(fi1, fi2 os.FileInfo) bool {
	return os.SameFile(fi1, fi2)
}
