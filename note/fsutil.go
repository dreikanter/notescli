package note

import (
	"fmt"
	"os"
)

// StoreDirMode returns the permissions to use when creating subdirectories
// under root. It inherits root's permissions so MkdirAll doesn't widen a
// restrictive root (e.g. 0o700), defaulting to 0o700 if root cannot be
// stat'd.
func StoreDirMode(root string) os.FileMode {
	info, err := os.Stat(root)
	if err != nil {
		return 0o700
	}
	return info.Mode().Perm()
}

// WriteAtomic writes data to path via a tmp+rename so partial writes don't
// leave a corrupted file behind.
func WriteAtomic(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("cannot write note: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace note: %w", err)
	}
	return nil
}
