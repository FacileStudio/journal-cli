package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

// renderEntry prints one log entry as a code line.
func renderEntry(entry client.Entry, clock bool) {
	at := entry.CreatedAt
	if clock {
		at = clockTime(entry.CreatedAt)
	} else {
		at = relativeTime(entry.CreatedAt)
	}
	fmt.Fprintf(
		ui.Stdout(),
		"%s %s %s %s\n",
		ui.Dim(at),
		ui.Dim(entry.App),
		ui.Level(entry.Level),
		entry.Message,
	)
}

// renderEntryWithMarker prints one log entry with a leading glyph marking the
// anchor — the entry a context query was asked about.
func renderEntryWithMarker(entry client.Entry, marker string) {
	at := relativeTime(entry.CreatedAt)
	fmt.Fprintf(
		ui.Stdout(),
		"%s %s %s %s %s\n",
		marker,
		ui.Dim(at),
		ui.Dim(entry.App),
		ui.Level(entry.Level),
		entry.Message,
	)
}

// renderEntryJSON prints one entry as a JSON document on its own line — the
// ndjson shape a consumer piping into jq wants from a tail.
func renderEntryJSON(entry client.Entry) {
	encoded, _ := json.Marshal(entry)
	ui.Plain("%s", string(encoded))
}

// relativeTime renders a timestamp as an age, which is what a log reader
// actually wants, falling back to the clock past a day.
func relativeTime(raw string) string {
	at, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}

	elapsed := time.Since(at)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	}
	return at.Local().Format("2 Jan 15:04")
}

// clockTime renders a timestamp as a wall clock, for the live feed where every
// line is seconds old and "just now" would say nothing.
func clockTime(raw string) string {
	at, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return at.Local().Format("15:04:05")
}
