package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/client"
	"github.com/FacileStudio/journal-cli/internal/ui"
)

var (
	flagKeyApp     string
	flagKeyPublic  bool
	flagKeyOrigins string
	flagKeyQuota   int
	flagKeyYes     bool
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API ingest keys",
	Long: `Manage secret backend and public browser ingest keys. Requires an admin session
or an admin token in JOURNAL_TOKEN.`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered API keys",
	Args:  cobra.NoArgs,
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

		keys, err := api.ListKeys(ctx)
		if err != nil {
			return err
		}

		filtered := make([]client.APIKey, 0, len(keys))
		for _, k := range keys {
			if flagKeyApp == "" || k.App == flagKeyApp {
				filtered = append(filtered, k)
			}
		}

		if flagJSON {
			return ui.JSON(filtered)
		}
		if len(filtered) == 0 {
			ui.Step("no API keys found")
			return nil
		}

		rows := make([][]string, 0, len(filtered))
		for _, k := range filtered {
			status := "active"
			if k.RevokedAt != nil {
				status = "revoked"
			}
			quota := "unlimited"
			if k.DailyQuota > 0 {
				quota = fmt.Sprintf("%d/day (%d used)", k.DailyQuota, k.UsedToday)
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", k.ID),
				k.App,
				k.Kind,
				k.Prefix,
				status,
				quota,
				relativeTime(k.CreatedAt),
			})
		}
		ui.Table([]string{"ID", "APP", "KIND", "PREFIX", "STATUS", "QUOTA", "CREATED"}, rows)
		return nil
	},
}

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API ingest key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagKeyApp == "" {
			return errors.New("--app is required")
		}

		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			return errors.New("no session stored — run `journal login` first")
		}

		kind := "secret"
		if flagKeyPublic {
			kind = "public"
		}

		var origins []string
		if flagKeyOrigins != "" {
			for _, o := range strings.Split(flagKeyOrigins, ",") {
				trimmed := strings.TrimSpace(o)
				if trimmed != "" {
					origins = append(origins, trimmed)
				}
			}
		}

		resp, err := api.CreateKey(ctx, client.CreateKeyRequest{
			App:            flagKeyApp,
			Kind:           kind,
			AllowedOrigins: origins,
			DailyQuota:     flagKeyQuota,
		})
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(resp)
		}

		ui.Success("created %s key for %s (id: %d)", resp.Key.Kind, resp.Key.App, resp.Key.ID)
		ui.Plain("%s", resp.Token)
		ui.Hint("store this token securely; it will not be shown again")
		return nil
	},
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errors.New("key id must be an integer")
		}

		ctx, stop := signalContext()
		defer stop()

		api, cfg, err := connect()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			return errors.New("no session stored — run `journal login` first")
		}

		if err := api.RevokeKey(ctx, id); err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(map[string]any{"revoked": id})
		}

		ui.Success("revoked key %d", id)
		return nil
	},
}

func init() {
	keysListCmd.Flags().StringVar(&flagKeyApp, "app", "", "Filter keys by application name")

	keysCreateCmd.Flags().StringVar(&flagKeyApp, "app", "", "Application name (required)")
	keysCreateCmd.Flags().BoolVar(&flagKeyPublic, "public", false, "Create a public browser key instead of a secret key")
	keysCreateCmd.Flags().StringVar(&flagKeyOrigins, "origins", "", "Comma-separated allowed origins (for public keys)")
	keysCreateCmd.Flags().IntVar(&flagKeyQuota, "quota", 0, "Daily event quota limit (for public keys)")

	keysRevokeCmd.Flags().BoolVarP(&flagKeyYes, "yes", "y", false, "Confirm revocation without prompting")

	keysCmd.AddCommand(keysListCmd, keysCreateCmd, keysRevokeCmd)
}
