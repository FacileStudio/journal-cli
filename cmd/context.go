package cmd

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FacileStudio/journal-cli/internal/ui"
)

var (
	flagBefore int
	flagAfter  int
)

var contextCmd = &cobra.Command{
	Use:   "context <id>",
	Short: "Read the stream around one entry",
	Long: `Prints the entries surrounding one entry, ignoring every filter — the
"what led to this" view. Newest first, with the anchor marked. Each side
defaults to 50 and is capped at 200.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signalContext()
		defer stop()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errors.New("the entry id must be a number")
		}

		api, cfg, err := connect()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			return errors.New("no session stored — run `journal login` first")
		}

		page, err := api.Context(ctx, id, flagBefore, flagAfter)
		if err != nil {
			return err
		}

		if flagJSON {
			return ui.JSON(page)
		}
		for _, entry := range page.Entries {
			marker := " "
			if entry.ID == id {
				marker = "»"
			}
			renderEntryWithMarker(entry, marker)
		}
		return nil
	},
}

func init() {
	contextCmd.Flags().IntVar(&flagBefore, "before", 50, "Entries before the anchor, capped at 200")
	contextCmd.Flags().IntVar(&flagAfter, "after", 50, "Entries after the anchor, capped at 200")
}
