package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// errHelp is returned by parseArgs when -h/--help was requested. main prints
// usage (owned by task 11) and exits 0 when it sees this sentinel.
var errHelp = errors.New("help requested")

// options holds the parsed command line.
type options struct {
	date      time.Time // positional YYYY-MM-DD; zero if absent
	week      *int      // nil if --week absent; 0 for bare --week
	yesterday bool
	all       bool
	since     time.Time // zero if absent
	until     time.Time // zero if absent
	project   string
	harness   string // "", "pi", or "claude"
	active    bool
	temp      bool
	jsonOut   bool
}

// whenFlagName describes which mutually-exclusive "when" flag has already
// been set, for error messages; "" means none yet.
func setWhen(current *string, name string) error {
	if *current != "" && *current != name {
		return fmt.Errorf("argument --%s: not allowed with argument --%s", name, *current)
	}
	*current = name
	return nil
}

// parseArgs parses argv (excluding the program name). On error, returns a
// non-nil error whose message names the offending flag/value. Handles -h and
// --help by returning a sentinel errHelp after usage is available to main.
func parseArgs(argv []string) (*options, error) {
	o := &options{}
	haveDate := false
	when := "" // tracks which of --week/--yesterday/--all was set first

	// next returns the value for a flag given either "--flag=value" (val,
	// true) or by consuming the next argv token ("--flag value").
	next := func(i *int, flag, inlineVal string, hasInline bool) (string, error) {
		if hasInline {
			return inlineVal, nil
		}
		if *i+1 >= len(argv) {
			return "", fmt.Errorf("argument --%s: expected one argument", flag)
		}
		*i++
		return argv[*i], nil
	}

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if arg == "-h" || arg == "--help" {
			return nil, errHelp
		}

		if len(arg) >= 2 && arg[0] == '-' && arg[1] == '-' && arg != "--" {
			name := arg[2:]
			inlineVal, hasInline := "", false
			if before, after, found := strings.Cut(name, "="); found {
				name, inlineVal, hasInline = before, after, true
			}

			switch name {
			case "week":
				if err := setWhen(&when, "week"); err != nil {
					return nil, err
				}
				val := 0
				if hasInline {
					n, err := strconv.Atoi(inlineVal)
					if err != nil {
						return nil, fmt.Errorf("argument --week: invalid int value: %q", inlineVal)
					}
					val = n
				} else if i+1 < len(argv) {
					if n, err := strconv.Atoi(argv[i+1]); err == nil {
						val = n
						i++
					}
				}
				v := val
				o.week = &v

			case "yesterday":
				if err := setWhen(&when, "yesterday"); err != nil {
					return nil, err
				}
				if hasInline {
					return nil, fmt.Errorf("argument --yesterday: ignored explicit argument %q", inlineVal)
				}
				o.yesterday = true

			case "all":
				if err := setWhen(&when, "all"); err != nil {
					return nil, err
				}
				if hasInline {
					return nil, fmt.Errorf("argument --all: ignored explicit argument %q", inlineVal)
				}
				o.all = true

			case "since":
				val, err := next(&i, "since", inlineVal, hasInline)
				if err != nil {
					return nil, err
				}
				d, err := parseDay(val)
				if err != nil {
					return nil, err
				}
				o.since = d

			case "until":
				val, err := next(&i, "until", inlineVal, hasInline)
				if err != nil {
					return nil, err
				}
				d, err := parseDay(val)
				if err != nil {
					return nil, err
				}
				o.until = d

			case "project":
				val, err := next(&i, "project", inlineVal, hasInline)
				if err != nil {
					return nil, err
				}
				o.project = val

			case "harness":
				val, err := next(&i, "harness", inlineVal, hasInline)
				if err != nil {
					return nil, err
				}
				if val != HarnessPi && val != HarnessClaude {
					return nil, fmt.Errorf("argument --harness: invalid choice: %q (choose from %q, %q)", val, HarnessPi, HarnessClaude)
				}
				o.harness = val

			case "active":
				if hasInline {
					return nil, fmt.Errorf("argument --active: ignored explicit argument %q", inlineVal)
				}
				o.active = true

			case "temp":
				if hasInline {
					return nil, fmt.Errorf("argument --temp: ignored explicit argument %q", inlineVal)
				}
				o.temp = true

			case "json":
				if hasInline {
					return nil, fmt.Errorf("argument --json: ignored explicit argument %q", inlineVal)
				}
				o.jsonOut = true

			default:
				return nil, fmt.Errorf("unrecognized arguments: %s", arg)
			}
			continue
		}

		// Positional.
		if haveDate {
			return nil, fmt.Errorf("unrecognized arguments: %s", arg)
		}
		d, err := parseDay(arg)
		if err != nil {
			return nil, err
		}
		o.date = d
		haveDate = true
	}

	return o, nil
}

// parseDay parses "YYYY-MM-DD" strictly (time.ParseInLocation with layout
// "2006-01-02" in time.Local, then dayOf). Error message:
// `expected YYYY-MM-DD, got "<value>"`.
func parseDay(value string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD, got %q", value)
	}
	return dayOf(t), nil
}

// weekBounds returns the Monday and Sunday (inclusive) of the calendar week
// `offset` weeks from the current one. Monday-based: Go's Weekday() has
// Sunday=0, Python's weekday() has Monday=0 — convert carefully:
// daysSinceMonday := (int(today.Weekday()) + 6) % 7. Ports week_bounds.
func weekBounds(offset int, today time.Time) (time.Time, time.Time) {
	daysSinceMonday := (int(today.Weekday()) + 6) % 7
	monday := dayOf(today).AddDate(0, 0, -daysSinceMonday+offset*7)
	sunday := monday.AddDate(0, 0, 6)
	return monday, sunday
}

// resolveRange maps options to an inclusive (start, end) day pair, both zero
// for --all. Precedence, matching Python resolve_range exactly:
// all → (zero, zero); positional date → (date, date); since/until set →
// (since or 1970-01-01, until or today); week non-nil → weekBounds;
// yesterday → (yesterday, yesterday); default → (today, today).
func resolveRange(o *options, today time.Time) (time.Time, time.Time) {
	if o.all {
		return time.Time{}, time.Time{}
	}
	if !o.date.IsZero() {
		return o.date, o.date
	}
	if !o.since.IsZero() || !o.until.IsZero() {
		since := o.since
		if since.IsZero() {
			since = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
		}
		until := o.until
		if until.IsZero() {
			until = dayOf(today)
		}
		return since, until
	}
	if o.week != nil {
		return weekBounds(*o.week, today)
	}
	if o.yesterday {
		y := dayOf(today).AddDate(0, 0, -1)
		return y, y
	}
	d := dayOf(today)
	return d, d
}
