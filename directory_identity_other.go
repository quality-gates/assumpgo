//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package assumpgo

import "os"

func directoryIdentity(os.FileInfo) (directoryKey, bool) {
	return directoryKey{}, false
}
