package note

import (
	"testing"
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
			// Multi-dot basenames can't come from NoteFilename (unsafe types
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
				if err == nil {
					t.Errorf("ParseFilename(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFilename(%q) unexpected error: %v", tt.input, err)
			}
			if got.Date != tt.wantDate {
				t.Errorf("Date = %q, want %q", got.Date, tt.wantDate)
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", got.ID, tt.wantID)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.wantSlug)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestIsID(t *testing.T) {
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
			if got := IsID(c.in); got != c.want {
				t.Errorf("IsID(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestHasSpecialBehavior(t *testing.T) {
	if !HasSpecialBehavior("todo") {
		t.Error("expected todo to have special behavior")
	}
	if !HasSpecialBehavior("backlog") {
		t.Error("expected backlog to have special behavior")
	}
	if !HasSpecialBehavior("weekly") {
		t.Error("expected weekly to have special behavior")
	}
	if HasSpecialBehavior("random") {
		t.Error("expected random to have no special behavior")
	}
	if HasSpecialBehavior("") {
		t.Error("expected empty string to have no special behavior")
	}
}
