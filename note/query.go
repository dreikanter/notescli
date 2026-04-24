package note

import "time"

// query captures the filter state built up from QueryOpts. It is unexported
// so that only Store implementations inside the note package can inspect it;
// consumers compose filters by passing QueryOpts.
type query struct {
	typeSet    bool
	noteType   string
	slugSet    bool
	slug       string
	tags       []string
	dateSet    bool
	date       time.Time
	beforeSet  bool
	beforeDate time.Time
}

// QueryOpt configures Store.All and Store.Find. Opts are combinable; multiple
// WithTag opts are AND-combined.
type QueryOpt func(*query)

// WithType matches entries whose Meta.Type equals t.
func WithType(t string) QueryOpt {
	return func(q *query) {
		q.typeSet = true
		q.noteType = t
	}
}

// WithSlug matches entries whose Meta.Slug equals s. When multiple entries
// share a slug the newest match is returned first.
func WithSlug(s string) QueryOpt {
	return func(q *query) {
		q.slugSet = true
		q.slug = s
	}
}

// WithTag matches entries whose Meta.Tags contains t (case-insensitive).
// Multiple WithTag opts combine with AND semantics.
func WithTag(t string) QueryOpt {
	return func(q *query) {
		q.tags = append(q.tags, t)
	}
}

// WithExactDate matches entries whose Meta.CreatedAt falls on the same
// calendar day as d (comparison is at day precision, in d's location).
func WithExactDate(d time.Time) QueryOpt {
	return func(q *query) {
		q.dateSet = true
		q.date = d
	}
}

// WithBeforeDate matches entries whose Meta.CreatedAt falls on a calendar
// day strictly before d (day precision, in d's location).
func WithBeforeDate(d time.Time) QueryOpt {
	return func(q *query) {
		q.beforeSet = true
		q.beforeDate = d
	}
}

// buildQuery applies opts to a fresh query value.
func buildQuery(opts []QueryOpt) query {
	var q query
	for _, opt := range opts {
		opt(&q)
	}
	return q
}

// matches reports whether entry satisfies every filter in q. Tag comparison
// is case-insensitive; date comparisons are at day precision in the filter's
// location.
func matches(entry Entry, q query) bool {
	if q.typeSet && entry.Meta.Type != q.noteType {
		return false
	}
	if q.slugSet && entry.Meta.Slug != q.slug {
		return false
	}
	if len(q.tags) > 0 && !hasAllTags(entry.Meta.Tags, q.tags) {
		return false
	}
	if q.dateSet && !sameDay(entry.Meta.CreatedAt, q.date) {
		return false
	}
	if q.beforeSet && !beforeDay(entry.Meta.CreatedAt, q.beforeDate) {
		return false
	}
	return true
}

// sameDay reports whether a and b fall on the same calendar day, using b's
// location for the comparison.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.In(b.Location()).Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// beforeDay reports whether a's calendar day is strictly earlier than b's,
// using b's location.
func beforeDay(a, b time.Time) bool {
	aDay := startOfDay(a.In(b.Location()))
	bDay := startOfDay(b)
	return aDay.Before(bDay)
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
