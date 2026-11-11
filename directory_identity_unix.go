//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package assumpgo

import (
	"os"
	"syscall"
)

func directoryIdentity(info os.FileInfo) (directoryKey, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return directoryKey{}, false
	}
	return directoryKey{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, true
}
