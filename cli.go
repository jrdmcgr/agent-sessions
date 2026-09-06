package main

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// errHelp is retained for callers of parseArgs. Commands themselves let Cobra
// render help directly.
var errHelp = fmt.Errorf("help requested")

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

// newListCommand constructs the default sessions command. Its flags and
// validation live in one place, so both the executable and parseArgs use the
// same Cobra parser.
func newListCommand(o *options, run func() error) *cobra.Command {
	var since, until string
	var showVersion bool
	cmd := &cobra.Command{
		Use:           "sessions [date]",
		Short:         "List pi and Claude Code sessions worked on in a date range.",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "sessions "+version)
				return err
			}
			if len(args) == 1 {
				d, err := parseDay(args[0])
				if err != nil {
					return err
				}
				o.date = d
			}
			if o.harness != "" && o.harness != HarnessPi && o.harness != HarnessClaude {
				return fmt.Errorf("invalid argument %q for --harness: choose from %q, %q", o.harness, HarnessPi, HarnessClaude)
			}
			if since != "" {
				d, err := parseDay(since)
				if err != nil {
					return err
				}
				o.since = d
			}
			if until != "" {
				d, err := parseDay(until)
				if err != nil {
					return err
				}
				o.until = d
			}
			if cmd.Flags().Changed("week") {
				week, err := cmd.Flags().GetInt("week")
				if err != nil {
					return err
				}
				o.week = &week
			}
			return run()
		},
	}
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print the version and exit")
	cmd.Flags().Int("week", 0, "calendar week, Monday-based; OFFSET -1 is last week")
	// pflag's NoOptDefVal preserves the useful `--week` shorthand while still
	// accepting `--week=-1` and `--week -1`.
	cmd.Flags().Lookup("week").NoOptDefVal = "0"
	cmd.Flags().BoolVar(&o.yesterday, "yesterday", false, "yesterday only")
	cmd.Flags().BoolVar(&o.all, "all", false, "every session on disk")
	cmd.Flags().StringVar(&o.project, "project", "", "filter by cwd/repo substring")
	cmd.Flags().StringVar(&o.harness, "harness", "", "filter by harness (pi or claude)")
	cmd.Flags().BoolVar(&o.active, "active", false, "only sessions touched in the last 2h")
	cmd.Flags().BoolVar(&o.temp, "temp", false, "include sessions run from temp dirs")
	cmd.Flags().BoolVar(&o.jsonOut, "json", false, "emit JSON instead of a table")
	cmd.Flags().StringVar(&since, "since", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&until, "until", "", "end date (YYYY-MM-DD)")
	cmd.MarkFlagsMutuallyExclusive("week", "yesterday", "all")
	return cmd
}

// parseArgs remains a small testable adapter around Cobra's parser.
func parseArgs(argv []string) (*options, error) {
	for _, arg := range argv {
		if arg == "-h" || arg == "--help" {
			return nil, errHelp
		}
	}
	o := &options{}
	cmd := newListCommand(o, func() error { return nil })
	cmd.SetArgs(normalizeWeekArg(argv))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		return nil, err
	}
	return o, nil
}

// normalizeWeekArg preserves the previous optional-argument behavior. pflag
// deliberately treats a following `-1` as another flag, while this CLI has
// always accepted `--week -1` as well as `--week=-1`.
func normalizeWeekArg(argv []string) []string {
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		if argv[i] == "--week" && i+1 < len(argv) {
			if _, err := strconv.Atoi(argv[i+1]); err == nil {
				out = append(out, "--week="+argv[i+1])
				i++
				continue
			}
		}
		out = append(out, argv[i])
	}
	return out
}

// parseDay parses "YYYY-MM-DD" strictly.
func parseDay(value string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD, got %q", value)
	}
	return dayOf(t), nil
}

func weekBounds(offset int, today time.Time) (time.Time, time.Time) {
	daysSinceMonday := (int(today.Weekday()) + 6) % 7
	monday := dayOf(today).AddDate(0, 0, -daysSinceMonday+offset*7)
	sunday := monday.AddDate(0, 0, 6)
	return monday, sunday
}

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
