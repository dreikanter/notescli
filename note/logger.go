package note

// Logger receives non-fatal errors from Load, Scan, and Index.Reload — e.g.
// per-note frontmatter parse failures or unreadable subdirectories that the
// walk chooses to skip rather than abort on. Install one via WithLogger (or
// WithScanLogger when calling Scan directly). A nil Logger discards the
// message; the package does not write to os.Stderr on its own.
type Logger func(error)

func (l Logger) log(err error) {
	if l == nil || err == nil {
		return
	}
	l(err)
}
