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
// appears in chronological order.
func follow(ctx context.Context, api *client.Client, filter client.Filter) error {
	maxID := int64(0)
	gapWarned := false

	for {
		page, err := api.Logs(ctx, filter)
		if err != nil {
			if err = retryPoll(ctx, err); err != nil {
				return err
			}
			continue
		}

		fresh, nextMax := collectFresh(page.Entries, maxID)
		maxID = nextMax
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

// retryPoll decides whether a poll failure should tear the tail down or wait
// and retry. An unauthenticated error ends the follow; anything else is a blip
// the tail survives (the dashboard stays quiet through a restart too). It
// returns nil after the wait, or the error that should end the follow.
func retryPoll(ctx context.Context, err error) error {
	var apiErr *client.Error
	if errors.As(err, &apiErr) && apiErr.Unauthenticated() {
		return err
	}
	ui.Warn("poll failed, retrying — %s", err)
	if !sleep(ctx) {
		return ErrInterrupted
	}
	return nil
}

// collectFresh filters a polled page down to the entries newer than the last
// one printed, in chronological order. The page arrives newest first; the
// result is reversed so the tail reads like a log. It returns the fresh rows
// and the new high-water mark.
func collectFresh(entries []client.Entry, maxID int64) ([]client.Entry, int64) {
	var fresh []client.Entry
	for _, entry := range entries {
		if entry.ID > maxID {
			fresh = append(fresh, entry)
			maxID = entry.ID
		}
	}
	for i, j := 0, len(fresh)-1; i < j; i, j = i+1, j-1 {
		fresh[i], fresh[j] = fresh[j], fresh[i]
	}
	return fresh, maxID
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
