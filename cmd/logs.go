package cmd

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

var (
	flagApp       string
	flagLevels    []string
	flagQ         string
	flagRequestID string
	flagSince     string
	flagUntil     string
	flagLimit     int
	flagBeforeTS  string
	flagBeforeID  int64
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Query the log",
	Long: `Prints entries newest first, filtered by app, level, free text, request id, and
time. Pages backwards until --limit entries are gathered, so a query asks for
exactly what it returns.

--since accepts a relative duration ("30m", "2h") or an RFC3339 timestamp;
--until is an RFC3339 timestamp. --before-ts/--before-id resume from a cursor
(the pair --json emits as next_before). With --json the whole result is one
document.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			return errors.New("no session stored — run `journal login` first")
		}

		since, until, err := resolveTimeRange(flagSince, flagUntil)
		if err != nil {
			return err
		}

		entries, err := api.LogsPaged(ctx, client.Filter{
			App:       flagApp,
			Levels:    flagLevels,
			Q:         flagQ,
			RequestID: flagRequestID,
			Since:     since,
			Until:     until,
			BeforeTS:  flagBeforeTS,
			BeforeID:  flagBeforeID,
		}, flagLimit)
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(entries)
		}
		if len(entries) == 0 {
			ui.Step("no entries match")
			return nil
		}
		for _, entry := range entries {
			renderEntry(entry, false)
		}
		return nil
	},
}

// resolveTimeRange turns the flag strings into RFC3339Nano query bounds. Empty
// values stay empty; a relative duration on --since is resolved against now.
func resolveTimeRange(sinceRaw, untilRaw string) (string, string, error) {
	if sinceRaw == "" && untilRaw == "" {
		return "", "", nil
	}

	var since string
	if sinceRaw != "" {
		if d, err := time.ParseDuration(sinceRaw); err == nil {
			since = time.Now().Add(-d).Format(time.RFC3339Nano)
		} else {
			since = sinceRaw
		}
	}

	if sinceRaw == "" && untilRaw != "" {
		return "", "", errors.New("--since is required when --until is set")
	}
	return since, untilRaw, nil
}

func init() {
	logsCmd.Flags().StringVar(&flagApp, "app", "", "Source app, exact match")
	logsCmd.Flags().StringSliceVar(&flagLevels, "level", nil, "Levels to include (error,warn,info,debug)")
	logsCmd.Flags().StringVar(&flagQ, "q", "", "Full-text search")
	logsCmd.Flags().StringVar(&flagRequestID, "request-id", "", "Exact match on meta request_id")
	logsCmd.Flags().StringVar(&flagSince, "since", "", "Lower bound: duration (30m) or RFC3339")
	logsCmd.Flags().StringVar(&flagUntil, "until", "", "Upper bound: RFC3339 timestamp")
	logsCmd.Flags().IntVar(&flagLimit, "limit", 100, "Max entries to gather across pages")
	logsCmd.Flags().StringVar(&flagBeforeTS, "before-ts", "", "Resume cursor ts, from a prior --json next_before")
	logsCmd.Flags().Int64Var(&flagBeforeID, "before-id", 0, "Resume cursor id, from a prior --json next_before")
}
