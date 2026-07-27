# dates

[![CI](https://github.com/go-datetime/dates/actions/workflows/ci.yml/badge.svg)](https://github.com/go-datetime/dates/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-datetime/dates.svg)](https://pkg.go.dev/github.com/go-datetime/dates)
[![License: BSD-3-Clause](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)

A dependency-free, pure-Go **lenient parser for real-world date/time strings** —
the messy dates that arrive on the wire from NNTP overview headers, RSS/Atom and
JSONFeed `pubDate` values, RFC 5322 email `Date` headers, HTTP dates, and
similar sources. It uses only the Go standard library (`time`, `strings`,
`fmt`), builds with `CGO_ENABLED=0`, and pulls in **zero third-party
dependencies**.

`dates` does **not** introduce a new time value type: `time.Time` stays the
value type. This package is the curated multi-format parser plus a
timezone-abbreviation resolver, mutualized so every consumer stops reinventing
its own list of layouts (and stops missing formats).

## Why

Each consumer tends to hand-roll a `[]string` of `time` layouts and still misses
cases. A concrete one, fixed here once for everyone: some servers emit a
**two-digit year together with a named zone**, e.g.

```
Tue, 07 Jul 26 11:13:37 UTC
```

The four-digit RFC 1123 layouts do not match this, so a naive parser silently
returns the zero time. `dates.Parse` handles it.

It also fixes a subtler standard-library trap: `time.Parse` with a named-zone
layout (`MST`) accepts an abbreviation but assigns it a **zero offset** unless it
happens to match the host's local zone — so `EST` parses as UTC+0, silently
wrong. `dates` resolves a curated, overridable table of abbreviations to their
canonical offsets.

## Install

```sh
go get github.com/go-datetime/dates
```

Requires Go 1.26.4 or newer.

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-datetime/dates"
)

func main() {
	t, err := dates.Parse("Tue, 07 Jul 26 11:13:37 UTC")
	if err != nil {
		panic(err)
	}
	fmt.Println(t.UTC()) // 2026-07-07 11:13:37 +0000 UTC

	// EST resolves to -0500, not a silent UTC.
	t, _ = dates.Parse("Wed, 15 Jan 2025 10:30:00 EST")
	fmt.Println(t) // 2025-01-15 10:30:00 -0500 EST
}
```

## API

```go
// Parse trims s, tries a curated, ordered list of layouts, and returns the
// first that parses. Zoneless inputs are interpreted as UTC. Returns a
// *ParseError if none match; empty/blank input is an error.
func Parse(s string) (time.Time, error)

// ParseIn is like Parse but interprets zoneless inputs in loc (nil == UTC).
// Inputs carrying their own zone ignore loc.
func ParseIn(s string, loc *time.Location) (time.Time, error)

// ParseZoneAbbrev resolves a timezone abbreviation (e.g. "EST") to its offset
// from UTC in seconds; ok is false for unknown abbreviations.
func ParseZoneAbbrev(abbr string) (offsetSeconds int, ok bool)

// SetZoneAbbrev adds or overrides an entry in the abbreviation table.
// Not safe to call concurrently with Parse/ParseIn/ParseZoneAbbrev.
func SetZoneAbbrev(abbr string, offsetSeconds int)

type ParseError struct{ Value string } // implements error
```

Exported layout constants (usable with `time.Format` too): `RFC822`, `RFC822Z`,
`RFC822Seconds`, `RFC822ZSeconds`, `RFC1123`, `RFC1123Z`, `RFC2822`
(≡ `RFC1123Z`), `RFC1123TwoDigit`, `RFC1123ZTwoDigit`, `RFC3339`, `RFC3339Nano`,
`ISO8601`, `ISO8601Space`, `ANSIC` / `Asctime`, and `HTTP`.

## Formats recognized

Tried from most-specific/most-common to least, so the first match is the right
one:

- RFC 3339 / ISO 8601, with `Z`, numeric offset, or zoneless — `T`-separated
  and space-separated, with optional fractional seconds; date-only
  (`2006-01-02`, `2006/01/02`).
- RFC 1123 / RFC 5322 / RSS — four-digit year, numeric offset (`-0700`) or
  named zone (`MST`), with or without a leading weekday, zero- or
  space-padded day.
- **Two-digit-year RFC 822 forms** — `Mon, 02 Jan 06 15:04:05 MST`,
  `... -0700`, the weekday-less `02 Jan 06 ...`, and the minute-precision
  variants without seconds. (This is the class that includes the NNTP bug
  above.)
- `asctime(3)` / Unix `date` — `Mon Jan _2 15:04:05 2006`, and with a named or
  numeric zone.
- RFC 850 — `Monday, 02-Jan-06 15:04:05 MST`.
- HTTP IMF-fixdate — `Sun, 06 Nov 1994 08:49:37 GMT`.

Inputs are normalized first: surrounding whitespace is trimmed, internal runs of
whitespace are collapsed to a single space (so the space-padded `asctime` day
still parses), and a single trailing RFC 5322 parenthetical comment such as
`(UTC)` is stripped.

## Timezone abbreviations

The resolver ships with a documented default table:

| Abbrev | Offset | | Abbrev | Offset |
|--------|--------|-|--------|--------|
| `UT`, `UTC`, `GMT`, `Z` | `+0000` | | `CET` | `+0100` |
| `EST` | `-0500` | | `CEST` | `+0200` |
| `EDT` | `-0400` | | `EET` | `+0200` |
| `CST` | `-0600` | | `EEST` | `+0300` |
| `CDT` | `-0500` | | `WET` | `+0000` |
| `MST` | `-0700` | | `WEST` | `+0100` |
| `MDT` | `-0600` | | `BST` | `+0100` |
| `PST` | `-0800` | | | |
| `PDT` | `-0700` | | | |

### Documented choices for ambiguous abbreviations

Some abbreviations are genuinely undecidable from the string alone. `dates`
picks the most common interpretation and documents it; override with
`SetZoneAbbrev` if your corpus differs:

- **`CST` → US/Canada Central Standard Time (`-0600`)** — not China Standard
  Time (`+0800`) nor Cuba.
- **`CDT` → US/Canada Central Daylight Time (`-0500`)**.
- **`BST` → British Summer Time (`+0100`)** — not Bangladesh Standard Time
  (`+0600`) nor Brazil.
- **`EST`/`EDT`, `MST`/`MDT`, `PST`/`PDT` → the US/Canada zones.**

### Single-letter military zones (RFC 5322 §4.3)

The obsolete single-letter military zones are **not** populated except for `Z`
(`+0000`). Per RFC 5322 §4.3, obsolete single-letter zones other than `Z`
SHOULD be treated as `-0000` — an unknown local offset. `dates` honors that:
`ParseZoneAbbrev("A")` reports the zone as unknown, and a parsed instant bearing
such a zone keeps a zero offset. Add explicit overrides with `SetZoneAbbrev` if
you need the historical military offsets.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
