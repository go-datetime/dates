// Package dates is a lenient, dependency-free parser for the messy real-world
// date/time strings that appear on the wire: NNTP overview dates, RSS/Atom and
// JSONFeed pubDate values, RFC 5322 email Date headers, HTTP dates, and the
// like. It is pure Go (CGO_ENABLED=0) and uses only the standard library.
//
// The package deliberately does not introduce a new time value type: the
// standard library time.Time remains the value type. dates provides only the
// curated multi-format Parse/ParseIn entry points and a timezone-abbreviation
// resolver.
//
// # Why a dedicated parser
//
// Every consumer of wire dates tends to reinvent its own []string of time
// layouts and still misses formats. A concrete example: some servers emit a
// two-digit year together with a named zone, such as
//
//	Tue, 07 Jul 26 11:13:37 UTC
//
// The four-digit RFC 1123 layouts do not match this, so a naive parser silently
// returns the zero time. dates fixes this once, for every consumer.
//
// # Timezone abbreviations
//
// The standard library time.Parse, given a layout with a named zone (the "MST"
// reference token), will accept an arbitrary abbreviation but assign it a zero
// offset unless the abbreviation happens to match the local zone. As a result
// "EST" is parsed as UTC+0 — silently wrong. dates resolves a curated,
// overridable table of common abbreviations to their canonical offsets and
// reconstructs the instant accordingly. See ParseZoneAbbrev and SetZoneAbbrev.
package dates

import (
	"fmt"
	"strings"
	"time"
)

// Exported layout constants for the formats dates understands. Callers may use
// them with time.Format as well as rely on Parse for input.
const (
	// RFC822 is the classic two-digit-year, minute-precision form with a named
	// zone: "01 Jan 06 15:04 MST".
	RFC822 = "02 Jan 06 15:04 MST"
	// RFC822Z is RFC822 with a numeric zone offset.
	RFC822Z = "02 Jan 06 15:04 -0700"
	// RFC822Seconds is RFC822 with seconds and a named zone.
	RFC822Seconds = "02 Jan 06 15:04:05 MST"
	// RFC822ZSeconds is RFC822 with seconds and a numeric zone offset.
	RFC822ZSeconds = "02 Jan 06 15:04:05 -0700"

	// RFC1123 is the four-digit-year form with a named zone.
	RFC1123 = "Mon, 02 Jan 2006 15:04:05 MST"
	// RFC1123Z is the four-digit-year form with a numeric zone offset. This is
	// the form most RSS/Atom feeds emit.
	RFC1123Z = "Mon, 02 Jan 2006 15:04:05 -0700"
	// RFC2822 is, for date/time purposes, identical to RFC1123Z.
	RFC2822 = RFC1123Z

	// RFC1123TwoDigit is the two-digit-year RFC 1123 form with a named zone,
	// e.g. "Tue, 07 Jul 26 11:13:37 UTC".
	RFC1123TwoDigit = "Mon, 02 Jan 06 15:04:05 MST"
	// RFC1123ZTwoDigit is the two-digit-year RFC 1123 form with a numeric zone.
	RFC1123ZTwoDigit = "Mon, 02 Jan 06 15:04:05 -0700"

	// RFC3339 is the canonical Internet timestamp.
	RFC3339 = "2006-01-02T15:04:05Z07:00"
	// RFC3339Nano is RFC3339 with fractional seconds.
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

	// ISO8601 is the compact ISO 8601 form with a numeric zone and no colon in
	// the offset.
	ISO8601 = "2006-01-02T15:04:05Z0700"
	// ISO8601Space is the space-separated ISO 8601 variant with a numeric zone.
	ISO8601Space = "2006-01-02 15:04:05Z0700"

	// ANSIC is the C asctime(3) form, as produced by ctime and many older
	// tools.
	ANSIC = "Mon Jan _2 15:04:05 2006"
	// Asctime is an alias of ANSIC.
	Asctime = ANSIC

	// HTTP is the RFC 7231 IMF-fixdate used in HTTP headers. The trailing "GMT"
	// is literal and denotes UTC.
	HTTP = "Mon, 02 Jan 2006 15:04:05 GMT"
)

// zoneKind describes how a layout carries (or omits) a timezone.
type zoneKind int

const (
	// zoneNone: the layout has no zone; a zoneless input is interpreted in the
	// caller-supplied location.
	zoneNone zoneKind = iota
	// zoneNumeric: the layout has a numeric offset (e.g. -0700); the parsed
	// offset is authoritative.
	zoneNumeric
	// zoneNamed: the layout has a named zone (the "MST" token); the abbreviation
	// is resolved through the zone table.
	zoneNamed
)

// layoutSpec pairs a time layout string with the way it carries a zone.
type layoutSpec struct {
	fmt  string
	kind zoneKind
}

// layouts is the ordered, curated list tried by ParseIn, from most specific and
// most common to least. Numeric-offset forms precede the corresponding named
// forms so that an explicit offset is never shadowed by an abbreviation match.
var layouts = []layoutSpec{
	{RFC3339Nano, zoneNumeric},
	{RFC3339, zoneNumeric},
	{ISO8601, zoneNumeric},
	{ISO8601Space, zoneNumeric},

	{RFC1123Z, zoneNumeric},
	{RFC1123, zoneNamed},
	{"Mon, 2 Jan 2006 15:04:05 -0700", zoneNumeric},
	{"Mon, 2 Jan 2006 15:04:05 MST", zoneNamed},
	{"02 Jan 2006 15:04:05 -0700", zoneNumeric},
	{"02 Jan 2006 15:04:05 MST", zoneNamed},
	{"2 Jan 2006 15:04:05 -0700", zoneNumeric},
	{"2 Jan 2006 15:04:05 MST", zoneNamed},

	{RFC1123ZTwoDigit, zoneNumeric},
	{RFC1123TwoDigit, zoneNamed},
	{RFC822ZSeconds, zoneNumeric},
	{RFC822Seconds, zoneNamed},

	{"Mon, 02 Jan 06 15:04 -0700", zoneNumeric},
	{"Mon, 02 Jan 06 15:04 MST", zoneNamed},
	{RFC822Z, zoneNumeric},
	{RFC822, zoneNamed},

	{"Mon Jan _2 15:04:05 -0700 2006", zoneNumeric},
	{"Mon Jan _2 15:04:05 MST 2006", zoneNamed},
	{ANSIC, zoneNone},

	{"Monday, 02-Jan-06 15:04:05 MST", zoneNamed},

	{"2006-01-02T15:04:05", zoneNone},
	{"2006-01-02 15:04:05", zoneNone},
	{"2006/01/02 15:04:05", zoneNone},
	{"2006-01-02", zoneNone},
	{"2006/01/02", zoneNone},
}

// ParseError is returned by Parse and ParseIn when the input does not match any
// known layout. It preserves the (trimmed) input for diagnostics.
type ParseError struct {
	// Value is the input string, after trimming and normalization.
	Value string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return fmt.Sprintf("dates: cannot parse %q as a known date/time format", e.Value)
}

// Parse trims s, tries the curated list of layouts in order, and returns the
// first that parses. Zoneless inputs are interpreted as UTC. It returns a
// *ParseError if none match; an empty or blank input is an error.
func Parse(s string) (time.Time, error) {
	return ParseIn(s, time.UTC)
}

// ParseIn behaves like Parse but interprets zoneless inputs in loc. A nil loc is
// treated as time.UTC. Inputs that carry their own zone (numeric offset or a
// resolvable abbreviation) ignore loc.
func ParseIn(s string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	s = normalize(s)
	if s == "" {
		return time.Time{}, &ParseError{Value: s}
	}
	for _, spec := range layouts {
		var (
			t   time.Time
			err error
		)
		if spec.kind == zoneNone {
			t, err = time.ParseInLocation(spec.fmt, s, loc)
		} else {
			// Parse numeric- and named-zone layouts in UTC so the host's local
			// zone can never influence the result; named zones are then fixed up
			// from the abbreviation table below.
			t, err = time.ParseInLocation(spec.fmt, s, time.UTC)
		}
		if err != nil {
			continue
		}
		if spec.kind == zoneNamed {
			if name, _ := t.Zone(); name != "" {
				if off, ok := ParseZoneAbbrev(name); ok {
					t = time.Date(t.Year(), t.Month(), t.Day(),
						t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
						time.FixedZone(name, off))
				}
			}
		}
		return t, nil
	}
	return time.Time{}, &ParseError{Value: s}
}

// normalize trims surrounding whitespace, collapses internal runs of whitespace
// to a single space (which keeps the space-padded asctime day working), and
// strips a single trailing RFC 5322 parenthetical comment such as "(UTC)".
func normalize(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if strings.HasSuffix(s, ")") {
		if i := strings.LastIndex(s, "("); i >= 0 {
			s = strings.TrimSpace(s[:i])
		}
	}
	return s
}

// zoneOffsets maps timezone abbreviations to their offset from UTC, in seconds.
// It is consulted by ParseZoneAbbrev and may be extended or overridden with
// SetZoneAbbrev.
//
// Some abbreviations are genuinely ambiguous; dates picks the most common
// interpretation and documents the choice here:
//
//   - CST -> US/Canada Central Standard Time (-0600), not China Standard Time
//     (+0800) nor Cuba Standard Time.
//   - CDT -> US/Canada Central Daylight Time (-0500).
//   - BST -> British Summer Time (+0100), not Bangladesh Standard Time (+0600)
//     nor Brazil.
//   - EST/EDT, MST/MDT, PST/PDT -> the US/Canada zones.
//
// Single-letter military zones (RFC 5322 §4.3) are deliberately not populated
// except for "Z". The specification says obsolete single-letter zones other
// than "Z" SHOULD be treated as "-0000" (an unknown local offset); dates honors
// that by leaving them unresolved, so ParseZoneAbbrev reports them as unknown
// and the parsed instant keeps a zero offset. Add explicit overrides with
// SetZoneAbbrev if a stricter interpretation is desired.
//
// The table is a package-level map: SetZoneAbbrev is not safe to call
// concurrently with Parse/ParseIn.
var zoneOffsets = map[string]int{
	// Zero-offset / universal.
	"UT":  0,
	"UTC": 0,
	"GMT": 0,
	"Z":   0, // RFC 5322 "Zulu" / military Z.

	// North American zones.
	"EST": -5 * 3600,
	"EDT": -4 * 3600,
	"CST": -6 * 3600,
	"CDT": -5 * 3600,
	"MST": -7 * 3600,
	"MDT": -6 * 3600,
	"PST": -8 * 3600,
	"PDT": -7 * 3600,

	// European zones.
	"WET":  0,
	"WEST": 1 * 3600,
	"CET":  1 * 3600,
	"CEST": 2 * 3600,
	"EET":  2 * 3600,
	"EEST": 3 * 3600,
	"BST":  1 * 3600,
}

// ParseZoneAbbrev resolves a timezone abbreviation (for example "EST") to its
// offset from UTC in seconds. ok is false when the abbreviation is unknown, in
// which case callers should treat it as an unknown local offset ("-0000" in RFC
// 5322 terms). Lookup is case-sensitive; wire abbreviations are upper-case.
func ParseZoneAbbrev(abbr string) (offsetSeconds int, ok bool) {
	off, ok := zoneOffsets[abbr]
	return off, ok
}

// SetZoneAbbrev adds or overrides an entry in the abbreviation table, making the
// table user-extensible (for example to disambiguate CST toward China). It is
// not safe to call concurrently with Parse, ParseIn, or ParseZoneAbbrev.
func SetZoneAbbrev(abbr string, offsetSeconds int) {
	zoneOffsets[abbr] = offsetSeconds
}
