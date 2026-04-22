package note

import (
	"io/fs"
	"testing"
	"time"
)

func TestNoteTime(t *testing.T) {
	cases := []struct {
		name   string
		date   string
		want   time.Time
		wantOK bool
	}{
		{
			name:   "valid YYYYMMDD",
			date:   "20260106",
			want:   time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
			wantOK: true,
		},
		{
			name:   "leap day",
			date:   "20240229",
			want:   time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			wantOK: true,
		},
		{name: "empty", date: ""},
		{name: "too short", date: "2026010"},
		{name: "too long", date: "120260106"},
		{name: "non-numeric", date: "2026010a"},
		{name: "invalid month", date: "20261301"},
		{name: "invalid day", date: "20260230"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := Note{Date: tc.date}
			got, ok := n.Time()
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				if !got.IsZero() {
					t.Errorf("expected zero time on failure, got %v", got)
				}
				return
			}
			if !got.Equal(tc.want) {
				t.Errorf("time = %v, want %v", got, tc.want)
			}
		})
	}
}

// fakeFileInfo is a minimal fs.FileInfo for mtime-priority tests.
type fakeFileInfo struct {
	mtime time.Time
}

func (f fakeFileInfo) Name() string       { return "" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return f.mtime }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestResolveEntryDate(t *testing.T) {
	uidTime := time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC)
	fmTime := time.Date(2025, 7, 4, 12, 30, 0, 0, time.UTC)
	mtime := time.Date(2024, 11, 11, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		entry      Entry
		fi         fs.FileInfo
		wantTime   time.Time
		wantSource string
	}{
		{
			name:       "uid wins over frontmatter and mtime",
			entry:      Entry{Note: Note{Date: "20260106"}, Frontmatter: Frontmatter{Date: fmTime}},
			fi:         fakeFileInfo{mtime: mtime},
			wantTime:   uidTime,
			wantSource: "uid",
		},
		{
			name:       "frontmatter when uid malformed",
			entry:      Entry{Note: Note{Date: "bogus"}, Frontmatter: Frontmatter{Date: fmTime}},
			fi:         fakeFileInfo{mtime: mtime},
			wantTime:   fmTime,
			wantSource: "frontmatter",
		},
		{
			name:       "mtime when uid malformed and frontmatter zero",
			entry:      Entry{Note: Note{Date: ""}},
			fi:         fakeFileInfo{mtime: mtime},
			wantTime:   mtime,
			wantSource: "mtime",
		},
		{
			name:       "nil fi skips mtime fallback",
			entry:      Entry{Note: Note{Date: "bad"}},
			fi:         nil,
			wantTime:   time.Time{},
			wantSource: "",
		},
		{
			name:       "nil fi still uses uid when valid",
			entry:      Entry{Note: Note{Date: "20260106"}},
			fi:         nil,
			wantTime:   uidTime,
			wantSource: "uid",
		},
		{
			name:       "nil fi still uses frontmatter when uid malformed",
			entry:      Entry{Note: Note{Date: ""}, Frontmatter: Frontmatter{Date: fmTime}},
			fi:         nil,
			wantTime:   fmTime,
			wantSource: "frontmatter",
		},
		{
			name:       "uid wins even when frontmatter is zero",
			entry:      Entry{Note: Note{Date: "20260106"}},
			fi:         fakeFileInfo{mtime: mtime},
			wantTime:   uidTime,
			wantSource: "uid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, src := ResolveEntryDate(tc.entry, tc.fi)
			if src != tc.wantSource {
				t.Errorf("source = %q, want %q", src, tc.wantSource)
			}
			if !got.Equal(tc.wantTime) {
				t.Errorf("time = %v, want %v", got, tc.wantTime)
			}
		})
	}
}
