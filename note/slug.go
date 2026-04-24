package note

import (
	"fmt"
	"regexp"
)

var slugRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidateSlug returns an error if the slug cannot safely appear in a note
// filename. Empty slugs are accepted (they just omit the slug segment).
// All-digit slugs are rejected because they conflict with numeric ID lookup.
// Anything outside [A-Za-z0-9_-] is rejected to keep filenames portable and to
// avoid confusing the filename cache suffix.
func ValidateSlug(slug string) error {
	if slug == "" {
		return nil
	}
	if IsDigits(slug) {
		return fmt.Errorf("slug %q is all digits, which conflicts with note ID resolution", slug)
	}
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug %q contains invalid characters; only [A-Za-z0-9_-] are allowed", slug)
	}
	return nil
}
