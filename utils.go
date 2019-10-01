package pandorasbox

import "strings"

const VFSPrefix = "vfs://"

func ConvertVFSPath(path string) (string, bool) {
	if IsVFSPath(path) {
		return strings.Replace(path, VFSPrefix, "/", 1), true
	}

	return path, false
}

func IsVFSPath(path string) bool {
	if strings.HasPrefix(path, VFSPrefix) {
		return true
	}

	return false
}
