package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/config"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the stored session",
	Long: `Revokes the session token on the server and forgets it locally, keeping the
instance URL so the next login needs nothing.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		cfg, err := config.Load()
		unreadable := err != nil
		if unreadable {
			ui.Warn("the stored configuration could not be read — clearing it anyway")
			cfg = config.Config{URL: config.DefaultURL}
		}
		if cfg.Token == "" && !unreadable {
			ui.Warn("no session stored")
			return nil
		}

		if cfg.Token != "" {
			api := client.New(cfg.URL, cfg.Token)
			if err := api.Logout(ctx); err != nil {
				var apiErr *client.Error
				if !errors.As(err, &apiErr) || !apiErr.Unauthenticated() {
					ui.Warn("the server could not revoke the session — %s", err)
				}
			}
		}
		if err := config.Clear(); err != nil {
			return err
		}
		ui.Success("signed out of %s", cfg.URL)
		return nil
	},
}
