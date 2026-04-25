package note

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDate string
		wantID   string
		wantSlug string
		wantType string
		wantErr  bool
	}{
		{
			name:     "simple without slug or type",
			input:    "20260106_8823",
			wantDate: "20260106",
			wantID:   "8823",
			wantSlug: "",
			wantType: "",
		},
		{
			name:     "with slug only",
			input:    "20241203_6973_disable-letter_opener",
			wantDate: "20241203",
			wantID:   "6973",
			wantSlug: "disable-letter_opener",
			wantType: "",
		},
		{
			name:     "with type only",
			input:    "20260102_8814.todo",
			wantDate: "20260102",
			wantID:   "8814",
			wantSlug: "",
			wantType: "todo",
		},
		{
			name:     "with slug and type",
			input:    "20260102_8814_standup.todo",
			wantDate: "20260102",
			wantID:   "8814",
			wantSlug: "standup",
			wantType: "todo",
		},
		{
			name:     "backlog type",
			input:    "20260312_9219.backlog",
			wantDate: "20260312",
			wantID:   "9219",
			wantSlug: "",
			wantType: "backlog",
		},
		{
			name:     "weekly type",
			input:    "20260312_9219.weekly",
			wantDate: "20260312",
			wantID:   "9219",
			wantSlug: "",
			wantType: "weekly",
		},
		{
			name:     "unknown dot suffix treated as filename-reported type",
			input:    "20260312_9219_foo.bar",
			wantDate: "20260312",
			wantID:   "9219",
			wantSlug: "foo",
			wantType: "bar",
		},
		{
			// Multi-dot basenames can't come from Filename (unsafe types
			// are omitted), so a stray '.' in the prefix means the suffix is
			// not a cached type — leave Type empty.
			name:    "multi-dot basename rejects suffix as type",
			input:   "20260312_9219.foo.bar",
			wantErr: true,
		},
		{
			name:     "custom type name (no registry gate)",
			input:    "20260106_8823.meeting",
			wantDate: "20260106",
			wantID:   "8823",
			wantSlug: "",
			wantType: "meeting",
		},
		{
			name:    "missing parts",
			input:   "20260106",
			wantErr: true,
		},
		{
			name:    "non-numeric date",
			input:   "abcdefgh_1234",
			wantErr: true,
		},
		{
			name:     "short year in date",
			input:    "2026010_1234",
			wantDate: "2026010",
			wantID:   "1234",
			wantSlug: "",
			wantType: "",
		},
		{
			name:     "distant future date",
			input:    "120260106_8823",
			wantDate: "120260106",
			wantID:   "8823",
			wantSlug: "",
			wantType: "",
		},
		{
			name:    "date too short for MMDD",
			input:   "2601_1234",
			wantErr: true,
		},
		{
			name:    "non-numeric id",
			input:   "20260106_abc",
			wantErr: true,
		},
		{
			name:    "empty id",
			input:   "20260106_",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilename(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDate, got.Date)
			assert.Equal(t, tt.wantID, got.ID)
			assert.Equal(t, tt.wantSlug, got.Slug)
			assert.Equal(t, tt.wantType, got.Type)
		})
	}
}

func TestIsDigits(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"0", true},
		{"1", true},
		{"8823", true},
		{"0001", true},
		{"abc", false},
		{"12a", false},
		{"a12", false},
		{"12 ", false},
		{" 12", false},
		{"-12", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, IsDigits(c.in))
		})
	}
}

func TestHasSpecialBehavior(t *testing.T) {
	assert.True(t, HasSpecialBehavior("todo"))
	assert.True(t, HasSpecialBehavior("backlog"))
	assert.True(t, HasSpecialBehavior("weekly"))
	assert.False(t, HasSpecialBehavior("random"))
	assert.False(t, HasSpecialBehavior(""))
}

func TestFilename(t *testing.T) {
	tests := []struct {
		date     string
		id       int
		slug     string
		noteType string
		want     string
	}{
		{"20260312", 9219, "", "", "20260312_9219.md"},
		{"20260312", 9219, "my-note", "", "20260312_9219_my-note.md"},
		{"20260312", 9219, "", "todo", "20260312_9219.todo.md"},
		{"20260312", 9219, "standup", "todo", "20260312_9219_standup.todo.md"},
		{"20260312", 9219, "", "backlog", "20260312_9219.backlog.md"},
		// Unsafe types are omitted from the filename (frontmatter remains canonical).
		{"20260312", 9219, "", "foo.bar", "20260312_9219.md"},
		{"20260312", 9219, "", "a/b", "20260312_9219.md"},
		{"20260312", 9219, "", `a\b`, "20260312_9219.md"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Filename(tt.date, tt.id, tt.slug, tt.noteType))
	}
}

func TestDirPath(t *testing.T) {
	got := DirPath("/archive", "20260312")
	want := "/archive/2026/03"
	assert.Equal(t, want, got)
}
