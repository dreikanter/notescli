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
		wantErr  bool
	}{
		{
			name:     "simple without slug",
			input:    "20260106_8823",
			wantDate: "20260106",
			wantID:   "8823",
			wantSlug: "",
		},
		{
			name:     "with slug",
			input:    "20260102_8814_todo",
			wantDate: "20260102",
			wantID:   "8814",
			wantSlug: "todo",
		},
		{
			name:     "slug with underscores",
			input:    "20241203_6973_disable-letter_opener",
			wantDate: "20241203",
			wantID:   "6973",
			wantSlug: "disable-letter_opener",
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
			name:    "short date",
			input:   "2026010_1234",
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
			if got.BaseName != tt.input {
				t.Errorf("BaseName = %q, want %q", got.BaseName, tt.input)
			}
		})
	}
}
