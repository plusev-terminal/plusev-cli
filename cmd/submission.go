package cmd

import (
	"fmt"
	"strconv"

	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/urfave/cli/v2"
)

func submissionCommand() *cli.Command {
	return &cli.Command{
		Name:  "submission",
		Usage: "List and cancel your pending submissions.",
		Subcommands: []*cli.Command{
			submissionListCommand(),
			submissionCancelCommand(),
		},
	}
}

func submissionListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List your submissions.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "status", Usage: "Filter by status (pending, approved, rejected, superseded, cancelled)"},
			&cli.StringFlag{Name: "kind", Usage: "Filter by kind (entry, release)"},
			&cli.StringFlag{Name: "plugin-id", Usage: "Filter by pluginId"},
		},
		Action: func(c *cli.Context) error {
			client, err := loadClient(c)
			if err != nil {
				return err
			}

			req := api.SubmissionListReq{
				Status:   c.String("status"),
				Kind:     c.String("kind"),
				PluginID: c.String("plugin-id"),
			}

			subs, err := client.ListSubmissions(c.Context, req)
			if err != nil {
				return err
			}

			if len(subs) == 0 {
				output.Info("No submissions.")
				return nil
			}

			rows := make([][]string, 0, len(subs))

			for _, s := range subs {
				rows = append(rows, []string{
					strconv.FormatUint(s.ID, 10),
					s.Kind,
					colorStatus(s.Status),
					s.PluginID,
					s.SubmittedAt.Format("2006-01-02 15:04"),
				})
			}

			output.Table([]string{"ID", "KIND", "STATUS", "PLUGIN", "SUBMITTED"}, rows)

			return nil
		},
	}
}

func submissionCancelCommand() *cli.Command {
	return &cli.Command{
		Name:      "cancel",
		Usage:     "Cancel a pending submission.",
		Args:      true,
		ArgsUsage: "<submissionId>",
		Action: func(c *cli.Context) error {
			idStr, err := requireArg(c, "submissionId")
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				return fmt.Errorf("<submissionId> must be numeric, got %q", idStr)
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			if err := client.CancelSubmission(c.Context, id); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Cancelled submission %d", id))

			return nil
		},
	}
}

// colorStatus renders a status word with a hint of color for scannability.
func colorStatus(status string) string {
	switch status {
	case "pending":
		return status
	case "approved":
		return "\033[32m" + status + "\033[0m"
	case "rejected":
		return "\033[31m" + status + "\033[0m"
	case "superseded", "cancelled":
		return "\033[2m" + status + "\033[0m"
	default:
		return status
	}
}
