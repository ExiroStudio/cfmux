//go:build !linux

package service

import (
	"errors"
	"os"
)

// checkOwnership is a stub on non-linux builds. The preflight check rejects
// non-linux systems before this is reachable in real flows.
func checkOwnership(info os.FileInfo, invoker *resolvedUser) error {
	_ = info
	_ = invoker
	return errors.New("cfmux service requires linux")
}
