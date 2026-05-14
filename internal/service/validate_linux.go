//go:build linux

package service

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// checkOwnership rejects binaries not owned by root or the invoking user.
func checkOwnership(info os.FileInfo, invoker *resolvedUser) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("cannot read file owner metadata")
	}
	if stat.Uid == 0 {
		return nil
	}
	if invoker != nil && stat.Uid == invoker.UID {
		return nil
	}
	return fmt.Errorf("owner uid %d is neither root nor the invoking user (uid %d)", stat.Uid, invoker.UID)
}
