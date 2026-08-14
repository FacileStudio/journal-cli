package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/config"
	"github.com/FacileStudio/journal-cli/internal/loopback"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

var (
	flagEmail     string
	flagPassword  string
	flagNoBrowser bool
)

var loginCmd = &cobra.Command{
	Use:   "login [url]",
	Short: "Authenticate with a Journal instance",
	Long: `Stores the instance URL and the session token it returns, so later commands
need neither. The URL defaults to the one already stored.

By default the login goes through single sign-on when the instance offers it:
a browser opens, the flow redirects back to a one-time code on this machine,
and the code is traded for a session token. An instance without SSO takes an
address and a password on the command line instead.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		// Loaded, not built: the stored document is read and only the two
		// fields this command owns are replaced, so a key added later survives
		// a login instead of being reset to its zero value.
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			cfg.URL = config.NormalizeURL(args[0])
		}
		if flagURL != "" {
			cfg.URL = config.NormalizeURL(flagURL)
		}
		if fromEnv := os.Getenv("JOURNAL_URL"); fromEnv != "" && flagURL == "" && len(args) == 0 {
			cfg.URL = config.NormalizeURL(fromEnv)
		}
		if cfg.URL == "" {
			cfg.URL = config.DefaultURL
		}

		api := client.New(cfg.URL, "")

		// Ask the instance what it accepts before prompting for anything. An
		// SSO-only instance will refuse a password, so asking for one first
		// would be a dead end dressed up as a choice.
		auth, err := api.AuthConfig(ctx)
		if err != nil {
			return err
		}

		var token string
		if auth.OIDCEnabled {
			token, err = loginViaSSO(ctx, api, cfg.URL)
		} else {
			ui.Warn("this instance has no single sign-on configured")
			token, err = loginViaPassword(ctx, api)
		}
		if err != nil {
			return err
		}

		cfg.Token = token
		if err := config.Save(cfg); err != nil {
			return err
		}
		ui.Success("signed in to %s", cfg.URL)
		return nil
	},
}

// loginViaSSO runs the porte CLI flow: a listener on loopback, a browser, a
// redirect back with a one-time code, and an exchange for the session token.
func loginViaSSO(ctx context.Context, api *client.Client, base string) (string, error) {
	listener, err := loopback.Listen()
	if err != nil {
		return "", err
	}
	state, err := loopback.RandomState()
	if err != nil {
		return "", err
	}

	loginURL := listener.LoginURL(base, state)
	if flagNoBrowser || !loopback.OpenBrowser(loginURL) {
		ui.Step("open this URL to sign in")
		ui.Plain("%s", loginURL)
	} else {
		ui.Step("opening your browser to sign in")
		ui.Hint("if nothing opened, copy the URL printed with --no-browser")
	}

	ui.Step("waiting for the login to complete")
	code, err := listener.WaitForCode(state)
	if err != nil {
		return "", err
	}

	token, err := api.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("the login code was refused — %w", err)
	}
	return token, nil
}

// loginViaPassword asks for an address and a password and exchanges them for a
// session token.
func loginViaPassword(ctx context.Context, api *client.Client) (string, error) {
	email := flagEmail
	if email == "" {
		print("Email: ")
		line, err := readLine(os.Stdin)
		if err != nil {
			return "", err
		}
		email = strings.TrimSpace(line)
	}
	if email == "" {
		return "", errors.New("an email address is required")
	}

	password := flagPassword
	if password == "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			print("Password: ")
			entered, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return "", err
			}
			password = string(entered)
		} else {
			// Nothing to read a secret from: an explicit --password is the
			// only way in from a pipe, and refusing beats echoing one into
			// the terminal.
			return "", errors.New("a password is required when stdin is not a terminal — pass --password")
		}
	}
	if password == "" {
		return "", errors.New("a password is required")
	}

	token, err := api.Login(ctx, email, password)
	if err != nil {
		return "", fmt.Errorf("login failed — %w", err)
	}
	return token, nil
}

func readLine(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}

func init() {
	loginCmd.Flags().StringVar(&flagEmail, "email", "", "Email address, skipping the prompt")
	loginCmd.Flags().StringVar(&flagPassword, "password", "", "Password, skipping the prompt")
	loginCmd.Flags().BoolVar(&flagNoBrowser, "no-browser", false, "Print the login URL instead of opening a browser")
}
