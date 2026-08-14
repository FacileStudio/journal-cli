package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

const (
	pollInterval = 2 * time.Second
	pageSize     = 100
)

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Follow new entries as they land",
	Long: `Polls the log every two seconds and prints entries as they arrive, until
interrupted. The same filters as ` + "`journal logs`" + ` apply, so a tail can
scope to one app, one level, or one request id.

With --json each entry is its own document, one per line — the ndjson shape a
consumer piping into jq wants.`,
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
		// A tail is "what happens from now"; an explicit --since reaches back.
		if since == "" {
			since = time.Now().Format(time.RFC3339Nano)
		}

		if !flagJSON {
			ui.Step("following %s, press ctrl-c to stop", cfg.URL)
		}

		return follow(ctx, api, client.Filter{
			App:       flagApp,
			Levels:    flagLevels,
			Q:         flagQ,
			RequestID: flagRequestID,
			Since:     since,
			Until:     until,
			Limit:     pageSize,
		})
	},
}

// follow polls until the context is cancelled, printing each entry as it
// appears in chronological order. A full page of new rows means the tail fell
// behind; it warns once rather than silently sewing a cut stream together.
func follow(ctx context.Context, api *client.Client, filter client.Filter) error {
	maxID := int64(0)
	gapWarned := false

	for {
		page, err := api.Logs(ctx, filter)
		if err != nil {
			var apiErr *client.Error
			if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
				return err
			}
			// A blip should not tear the tail down — the dashboard stays
			// quiet through a restart too.
			ui.Warn("poll failed, retrying — %s", err)
			if !sleep(ctx) {
				return ErrInterrupted
			}
			continue
		}

		// Entries arrive newest first. Collect the fresh ones, then reverse
		// to chronological so the output reads like a log.
		var fresh []client.Entry
		for _, entry := range page.Entries {
			if entry.ID > maxID {
				fresh = append(fresh, entry)
				maxID = entry.ID
			}
		}
		for i, j := 0, len(fresh)-1; i < j; i, j = i+1, j-1 {
			fresh[i], fresh[j] = fresh[j], fresh[i]
		}

		if len(fresh) == pageSize && !gapWarned {
			ui.Warn("more entries than one poll could carry — the stream was cut")
			gapWarned = true
		}

		for _, entry := range fresh {
			if flagJSON {
				renderEntryJSON(entry)
			} else {
				renderEntry(entry, true)
			}
		}

		if !sleep(ctx) {
			return ErrInterrupted
		}
	}
}

// sleep returns false when the context is cancelled before the poll interval
// elapses, so the caller exits with the interrupted status rather than another
// poll.
func sleep(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(pollInterval):
		return true
	}
}

func init() {
	tailCmd.Flags().StringVar(&flagApp, "app", "", "Source app, exact match")
	tailCmd.Flags().StringSliceVar(&flagLevels, "level", nil, "Levels to include (error,warn,info,debug)")
	tailCmd.Flags().StringVar(&flagQ, "q", "", "Full-text search")
	tailCmd.Flags().StringVar(&flagRequestID, "request-id", "", "Exact match on meta request_id")
	tailCmd.Flags().StringVar(&flagSince, "since", "", "Start from a duration (30m) or RFC3339 instead of now")
	tailCmd.Flags().StringVar(&flagUntil, "until", "", "Upper bound: RFC3339 timestamp")
}
