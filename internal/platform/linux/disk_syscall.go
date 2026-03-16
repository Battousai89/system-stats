//go:build linux
// +build linux

package linux

import (
	"syscall"
)

// doStatfs performs the actual statfs syscall
func doStatfs(path string, stat *syscallStatfs) error {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return err
	}

	stat.Blocks = fs.Blocks
	stat.Bfree = fs.Bfree
	stat.Bavail = fs.Bavail
	stat.Bsize = int64(fs.Bsize)
	stat.Files = fs.Files
	stat.Ffree = fs.Ffree

	return nil
}
