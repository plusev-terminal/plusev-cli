package cmd

import (
	"fmt"
	"net/mail"

	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

func keyCommand() *cli.Command {
	return &cli.Command{
		Name:  "key",
		Usage: "Manage your dev key.",
		Subcommands: []*cli.Command{
			keyEmailCommand(),
		},
	}
}

func keyEmailCommand() *cli.Command {
	return &cli.Command{
		Name:      "email",
		Usage:     "Request a change to the email bound to your dev key.",
		Args:      true,
		ArgsUsage: "[email]  (prompts if omitted)",
		Action: func(c *cli.Context) error {
			email := c.Args().First()

			if email == "" {
				input, err := prompt.RequiredText("New email", "you@example.com")
				if err != nil {
					return err
				}

				email = input
			}

			if _, err := mail.ParseAddress(email); err != nil {
				return fmt.Errorf("invalid email %q: %w", email, err)
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			if err := client.UpdateEmail(c.Context, email); err != nil {
				return err
			}

			output.Success("Email change requested.")

			return nil
		},
	}
}
