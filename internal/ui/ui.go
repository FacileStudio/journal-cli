package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	faint  = color.New(color.Faint)
)

// DisableColor implements --no-color. NO_COLOR and TTY detection are already
// handled by fatih/color at init time.
func DisableColor() {
	color.NoColor = true
}

func Step(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "%s %s\n", cyan.Sprint("▸"), fmt.Sprintf(format, a...))
}

func Success(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "%s %s\n", green.Sprint("✓"), fmt.Sprintf(format, a...))
}

func Warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", yellow.Sprint("!"), fmt.Sprintf(format, a...))
}

func Error(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", red.Sprint("✗"), fmt.Sprintf(format, a...))
}

func Hint(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "  %s\n", faint.Sprint(fmt.Sprintf(format, a...)))
}

func Dim(s string) string {
	return faint.Sprint(s)
}

// Everything above is the shared implementation of CLI-STANDARD §4.1 and §4.4,
// copied verbatim. Below it are the helpers this CLI needs on top: the domain
// glyphs of §4.1 and a table writer, both of which emit data rather than status.

// Level colors a log level by severity. Levels are data, but severity is
// information a reader should not have to hunt for.
func Level(level string) string {
	switch strings.ToLower(level) {
	case "error", "fatal", "panic":
		return red.Sprint(level)
	case "warn", "warning":
		return yellow.Sprint(level)
	case "info":
		return green.Sprint(level)
	default: // debug, trace, and anything a future level adds
		return faint.Sprint(level)
	}
}

// Table writes aligned columns to stdout. The header is dimmed rather than
// bold: it is scaffolding, and the data is what the eye should land on.
func Table(header []string, rows [][]string) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, faint.Sprint(strings.Join(header, "\t")))
	for _, row := range rows {
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}
	writer.Flush()
}

// JSON writes one document to stdout and nothing else, per CLI-STANDARD §5.
func JSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// Plain writes a bare line of data to stdout, for piped output.
func Plain(format string, a ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}

// Stdout is the data sink, kept behind one name so a future --output flag has
// exactly one place to redirect.
func Stdout() *os.File { return os.Stdout }
