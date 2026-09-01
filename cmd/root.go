// Package cmd implements the journal command tree.
package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/config"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

var version = "dev"

var (
	flagURL     string
	flagJSON    bool
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "journal",
	Short: "Terminal client for a Journal instance",
	Long: `Journal is the suite's centralized logging service: apps ship structured
entries over HTTP, and this is its terminal client. It queries, filters, and
follows the log without opening the dashboard, and emits JSON for anything
that pipes into a tool.

Configuration stays in the instance. This client never writes it.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// Set once the command's own body starts. Cobra validates flags and args
	// before this runs, so an error arriving with it still false is a usage
	// error rather than a failure of the work — and those exit 2, not 1.
	PersistentPreRun: func(cmd *cobra.Command, args []string) { commandStarted = true },
}

var commandStarted bool

func init() {
	rootCmd.Version = version
	// cobra's default is `<bin> version <v>`, which the installer cannot parse.
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "Journal instance URL, overriding the stored one")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Print one JSON document and nothing else")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")

	cobra.OnInitialize(func() {
		// Structured output forces colour off: a caller piping JSON into jq
		// must not receive escape codes.
		if flagNoColor || flagJSON {
			ui.DisableColor()
		}
	})
}

// ErrInterrupted marks a command stopped by a signal rather than a failure.
var ErrInterrupted = errors.New("interrupted")

// Execute runs the command tree and maps the outcome onto an exit code:
// 0 success, 1 error, 2 usage error (cobra's own), 130 on SIGINT.
func Execute() {
	rootCmd.AddCommand(
		loginCmd,
		logoutCmd,
		logsCmd,
		tailCmd,
		contextCmd,
		appsCmd,
		keysCmd,
	)

	err := rootCmd.Execute()
	switch {
	case err == nil:
		return
	case !commandStarted:
		ui.Error("%s", err)
		ui.Hint("run `journal <command> --help` for usage")
		os.Exit(2)
	case errors.Is(err, ErrInterrupted):
		// 128 + SIGINT, which is what a shell and every `while` loop expect
		// from a process the user stopped.
		os.Exit(130)
	default:
		ui.Error("%s", err)
		os.Exit(1)
	}
}

// signalContext cancels on SIGINT and SIGTERM, so a long-running follow stops
// cleanly and reports an interrupted exit instead of a failure.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// connect builds a client from the stored configuration, with --url and the
// JOURNAL_URL environment variable able to override the stored instance. A
// caller that needs the session (every read command) must check token separately.
func connect() (*client.Client, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}

	if fromEnv := serverURLFromEnv(); fromEnv != "" {
		cfg.URL = config.NormalizeURL(fromEnv)
	}
	if flagURL != "" {
		cfg.URL = config.NormalizeURL(flagURL)
	}
	// An explicit token in the environment wins over the stored one — the
	// headless/CI path, and how an agent with its own credential avoids
	// touching the config file.
	if fromEnv := os.Getenv("JOURNAL_TOKEN"); fromEnv != "" {
		cfg.Token = fromEnv
	}

	return client.New(cfg.URL, cfg.Token), cfg, nil
}

// serverURLFromEnv reads the instance URL from the environment, preferring the
// suite-wide JOURNAL_SERVER_URL that CLI-STANDARD §6.3 names over the
// JOURNAL_URL this CLI shipped with. The older spelling stays an accepted alias
// so an existing shell profile keeps working.
func serverURLFromEnv() string {
	if value := os.Getenv("JOURNAL_SERVER_URL"); value != "" {
		return value
	}
	return os.Getenv("JOURNAL_URL")
}
