//go:build !unix

package note

// lockStoreRoot is a no-op on non-Unix platforms where syscall.Flock is not
// available. Concurrent NextID calls are not serialized on these platforms;
// the notesctl CLI is primarily used interactively on a single machine, so
// races are unlikely in practice.
func lockStoreRoot(_ string) (func(), error) {
	return func() {}, nil
}
