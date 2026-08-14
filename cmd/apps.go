package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/ui"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List what the instance is seeing",
	Long: `Prints the distinct apps that have logged, most recently active first, with a
count and a last-seen. A good first probe: it says both what ships here and
what has gone quiet.`,
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

		apps, err := api.Apps(ctx)
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(apps)
		}
		if len(apps) == 0 {
			ui.Step("no apps have logged yet")
			return nil
		}
		rows := make([][]string, 0, len(apps))
		for _, app := range apps {
			last := app.LastSeen
			if rel := relativeTime(app.LastSeen); rel != last {
				last = rel
			}
			rows = append(rows, []string{app.Name, fmt.Sprintf("%d", app.Count), last})
		}
		ui.Table([]string{"APP", "EVENTS", "LAST SEEN"}, rows)
		return nil
	},
}
