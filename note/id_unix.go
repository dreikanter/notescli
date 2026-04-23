//go:build unix

package note

import (
	"fmt"
	"os"
	"syscall"
)

// lockStoreRoot acquires an exclusive flock on the store root directory,
// blocking until it is available. Locking the directory (rather than a
// sibling lockfile) avoids leaving artifacts behind and sidesteps the
// unlink-race that cleanable lockfiles suffer from. The returned function
// releases the lock and closes the file descriptor.
func lockStoreRoot(root string) (func(), error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf("cannot open notes root for locking: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("cannot lock notes root: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
