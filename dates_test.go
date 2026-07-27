package dates

import (
	"errors"
	"testing"
	"time"
)

// specimen is one real-world date string and the instant it must resolve to.
type specimen struct {
	name   string
	in     string
	want   time.Time
	offset int // expected zone offset in seconds, for zone assertions
}

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	got, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", s, err)
	}
	return got
}

// TestParseSpecimens exercises Parse against real-world specimens, including the
// mandated self-validation set.
func TestParseSpecimens(t *testing.T) {
	utc := time.UTC
	specs := []specimen{
		{
			name:   "NNTP two-digit year named zone (Free bug)",
			in:     "Tue, 07 Jul 26 11:13:37 UTC",
			want:   time.Date(2026, 7, 7, 11, 13, 37, 0, utc),
			offset: 0,
		},
		{
			name:   "RSS RFC1123Z numeric offset",
			in:     "Mon, 02 Jan 2006 15:04:05 -0700",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "RFC3339 Z",
			in:     "2026-07-27T14:05:00Z",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "RFC3339Nano",
			in:     "2026-07-27T14:05:00.500000000+02:00",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 500000000, time.FixedZone("", 2*3600)),
			offset: 2 * 3600,
		},
		{
			name:   "ISO8601 compact numeric",
			in:     "2026-07-27T14:05:00+0200",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, time.FixedZone("", 2*3600)),
			offset: 2 * 3600,
		},
		{
			name:   "ISO8601 space numeric",
			in:     "2026-07-27 14:05:00+0000",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "ISO8601 space zoneless (UTC default)",
			in:     "2026-07-27 14:05:00",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "ISO8601 T zoneless",
			in:     "2026-07-27T14:05:00",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "date only",
			in:     "2026-07-27",
			want:   time.Date(2026, 7, 27, 0, 0, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "slash date only",
			in:     "2026/07/27",
			want:   time.Date(2026, 7, 27, 0, 0, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "slash datetime",
			in:     "2026/07/27 14:05:00",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "asctime",
			in:     "Mon Jul 27 14:05:00 2026",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "asctime single-digit day (double space collapsed)",
			in:     "Mon Jul  7 14:05:00 2026",
			want:   time.Date(2026, 7, 7, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "unix date named zone",
			in:     "Mon Jul 27 14:05:00 CEST 2026",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, time.FixedZone("CEST", 2*3600)),
			offset: 2 * 3600,
		},
		{
			name:   "unix date numeric zone",
			in:     "Mon Jul 27 14:05:00 +0200 2026",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, time.FixedZone("", 2*3600)),
			offset: 2 * 3600,
		},
		{
			name:   "HTTP IMF-fixdate (GMT via named layout)",
			in:     "Sun, 06 Nov 1994 08:49:37 GMT",
			want:   time.Date(1994, 11, 6, 8, 49, 37, 0, utc),
			offset: 0,
		},
		{
			name:   "EST resolves to -0500 not silent UTC",
			in:     "Wed, 15 Jan 2025 10:30:00 EST",
			want:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "PDT resolves to -0700",
			in:     "Wed, 15 Jul 2025 10:30:00 PDT",
			want:   time.Date(2025, 7, 15, 10, 30, 0, 0, time.FixedZone("PDT", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "RFC1123 four-digit named WET (zero offset, non-UTC name)",
			in:     "Wed, 15 Jan 2025 10:30:00 WET",
			want:   time.Date(2025, 1, 15, 10, 30, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "single-digit day with weekday, numeric",
			in:     "Mon, 2 Jan 2006 15:04:05 -0700",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "single-digit day with weekday, named",
			in:     "Mon, 2 Jan 2006 15:04:05 EST",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("EST", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "no weekday four-digit numeric",
			in:     "02 Jan 2006 15:04:05 -0700",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "no weekday four-digit named",
			in:     "02 Jan 2006 15:04:05 CST",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("CST", -6*3600)),
			offset: -6 * 3600,
		},
		{
			name:   "no weekday single-digit day numeric",
			in:     "2 Jan 2006 15:04:05 -0700",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "no weekday single-digit day named",
			in:     "2 Jan 2006 15:04:05 MST",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "two-digit year numeric weekday",
			in:     "Tue, 07 Jul 26 11:13:37 -0500",
			want:   time.Date(2026, 7, 7, 11, 13, 37, 0, time.FixedZone("", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "two-digit year seconds no weekday numeric",
			in:     "07 Jul 26 11:13:37 -0500",
			want:   time.Date(2026, 7, 7, 11, 13, 37, 0, time.FixedZone("", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "two-digit year seconds no weekday named (RFC822Seconds)",
			in:     "07 Jul 26 11:13:37 MST",
			want:   time.Date(2026, 7, 7, 11, 13, 37, 0, time.FixedZone("MST", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "RFC822 minute precision numeric weekday",
			in:     "Tue, 07 Jul 26 11:13 -0500",
			want:   time.Date(2026, 7, 7, 11, 13, 0, 0, time.FixedZone("", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "RFC822 minute precision named weekday",
			in:     "Tue, 07 Jul 26 11:13 CET",
			want:   time.Date(2026, 7, 7, 11, 13, 0, 0, time.FixedZone("CET", 1*3600)),
			offset: 1 * 3600,
		},
		{
			name:   "RFC822Z minute precision no weekday numeric",
			in:     "07 Jul 26 11:13 -0500",
			want:   time.Date(2026, 7, 7, 11, 13, 0, 0, time.FixedZone("", -5*3600)),
			offset: -5 * 3600,
		},
		{
			name:   "RFC822 minute precision no weekday named",
			in:     "07 Jul 26 11:13 GMT",
			want:   time.Date(2026, 7, 7, 11, 13, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "RFC850 form",
			in:     "Sunday, 06-Nov-94 08:49:37 EDT",
			want:   time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("EDT", -4*3600)),
			offset: -4 * 3600,
		},
		{
			name:   "trailing parenthetical comment stripped",
			in:     "Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*3600)),
			offset: -7 * 3600,
		},
		{
			name:   "surrounding whitespace trimmed",
			in:     "   2026-07-27T14:05:00Z   ",
			want:   time.Date(2026, 7, 27, 14, 5, 0, 0, utc),
			offset: 0,
		},
		{
			name:   "unknown three-letter zone kept as -0000",
			in:     "Mon, 02 Jan 2006 15:04:05 FOO",
			want:   time.Date(2006, 1, 2, 15, 4, 5, 0, utc),
			offset: 0,
		},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			got := mustParse(t, s.in)
			if !got.Equal(s.want) {
				t.Fatalf("Parse(%q) = %v, want %v (instant mismatch)", s.in, got, s.want)
			}
			if _, off := got.Zone(); off != s.offset {
				t.Fatalf("Parse(%q) zone offset = %d, want %d", s.in, off, s.offset)
			}
		})
	}
}

// TestParseErrors covers the failure paths.
func TestParseErrors(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"not a date at all",
		"the quick brown fox",
		"2026-13-40",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := Parse(in)
			if err == nil {
				t.Fatalf("Parse(%q) = nil error, want *ParseError", in)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%q) error type = %T, want *ParseError", in, err)
			}
		})
	}
}

// TestParseErrorMessage covers ParseError.Error formatting and Value retention.
func TestParseErrorMessage(t *testing.T) {
	_, err := Parse("garbage input string")
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Value != "garbage input string" {
		t.Fatalf("ParseError.Value = %q, want %q", pe.Value, "garbage input string")
	}
	want := `dates: cannot parse "garbage input string" as a known date/time format`
	if pe.Error() != want {
		t.Fatalf("ParseError.Error() = %q, want %q", pe.Error(), want)
	}
}

// TestParseInLocation covers the zoneless-in-loc path and the nil-loc default.
func TestParseInLocation(t *testing.T) {
	plus5 := time.FixedZone("PLUS5", 5*3600)

	// Zoneless input is interpreted in the supplied location.
	got, err := ParseIn("2026-07-27 14:05:00", plus5)
	if err != nil {
		t.Fatalf("ParseIn error: %v", err)
	}
	want := time.Date(2026, 7, 27, 14, 5, 0, 0, plus5)
	if !got.Equal(want) {
		t.Fatalf("ParseIn zoneless = %v, want %v", got, want)
	}
	if _, off := got.Zone(); off != 5*3600 {
		t.Fatalf("ParseIn zoneless offset = %d, want %d", off, 5*3600)
	}

	// A zoned input ignores loc.
	got, err = ParseIn("2026-07-27T14:05:00Z", plus5)
	if err != nil {
		t.Fatalf("ParseIn zoned error: %v", err)
	}
	if _, off := got.Zone(); off != 0 {
		t.Fatalf("ParseIn zoned offset = %d, want 0", off)
	}

	// nil loc defaults to UTC.
	got, err = ParseIn("2026-07-27 14:05:00", nil)
	if err != nil {
		t.Fatalf("ParseIn nil loc error: %v", err)
	}
	if !got.Equal(time.Date(2026, 7, 27, 14, 5, 0, 0, time.UTC)) {
		t.Fatalf("ParseIn nil loc = %v, want UTC interpretation", got)
	}

	// Error path through ParseIn as well.
	if _, err := ParseIn("nonsense", plus5); err == nil {
		t.Fatal("ParseIn(nonsense) = nil error, want error")
	}
}

// TestParseZoneAbbrev covers known, ambiguous-documented, and unknown lookups.
func TestParseZoneAbbrev(t *testing.T) {
	known := map[string]int{
		"UT":   0,
		"UTC":  0,
		"GMT":  0,
		"Z":    0,
		"EST":  -5 * 3600,
		"EDT":  -4 * 3600,
		"CST":  -6 * 3600,
		"CDT":  -5 * 3600,
		"MST":  -7 * 3600,
		"MDT":  -6 * 3600,
		"PST":  -8 * 3600,
		"PDT":  -7 * 3600,
		"WET":  0,
		"WEST": 1 * 3600,
		"CET":  1 * 3600,
		"CEST": 2 * 3600,
		"EET":  2 * 3600,
		"EEST": 3 * 3600,
		"BST":  1 * 3600,
	}
	for abbr, want := range known {
		off, ok := ParseZoneAbbrev(abbr)
		if !ok {
			t.Errorf("ParseZoneAbbrev(%q) ok = false, want true", abbr)
			continue
		}
		if off != want {
			t.Errorf("ParseZoneAbbrev(%q) = %d, want %d", abbr, off, want)
		}
	}

	// Unknown abbreviations, including obsolete single-letter military zones
	// other than Z, are reported as unknown (RFC 5322 -0000 semantics).
	for _, abbr := range []string{"A", "J", "Y", "FOO", "XYZ", ""} {
		if off, ok := ParseZoneAbbrev(abbr); ok || off != 0 {
			t.Errorf("ParseZoneAbbrev(%q) = (%d, %v), want (0, false)", abbr, off, ok)
		}
	}
}

// TestSetZoneAbbrev covers overriding the table and the corresponding parse
// behavior.
func TestSetZoneAbbrev(t *testing.T) {
	// Disambiguate CST toward China Standard Time for this test, then restore.
	orig, _ := ParseZoneAbbrev("CST")
	defer SetZoneAbbrev("CST", orig)

	SetZoneAbbrev("CST", 8*3600)
	if off, ok := ParseZoneAbbrev("CST"); !ok || off != 8*3600 {
		t.Fatalf("after override ParseZoneAbbrev(CST) = (%d, %v), want (%d, true)", off, ok, 8*3600)
	}

	got, err := Parse("Mon, 02 Jan 2006 15:04:05 CST")
	if err != nil {
		t.Fatalf("Parse after override error: %v", err)
	}
	if _, off := got.Zone(); off != 8*3600 {
		t.Fatalf("Parse CST after override offset = %d, want %d", off, 8*3600)
	}

	// Add a brand-new abbreviation.
	SetZoneAbbrev("NPT", 5*3600+45*60) // Nepal, +05:45
	defer delete(zoneOffsets, "NPT")
	if off, ok := ParseZoneAbbrev("NPT"); !ok || off != 5*3600+45*60 {
		t.Fatalf("ParseZoneAbbrev(NPT) = (%d, %v), want (%d, true)", off, ok, 5*3600+45*60)
	}
}
