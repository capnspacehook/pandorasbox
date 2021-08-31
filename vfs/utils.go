package vfs

func IsPathSeparator(c uint8) bool {
	return PathSeparator == c
}

func SameFile(fi1, fi2 *FileInfo) bool {
	return fi1.node.Ino == fi2.node.Ino
}
