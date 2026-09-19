package commands

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show account info, plan, and point/quota balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		p, err := c.GetProfile(cmdContext())
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.Profile(w, p, false) })
		rec().Record(record.TimestampPath("profile", ""), "profile", "", redactedCommandLine(), p, text)
		return printProfile(p)
	},
}

func printProfile(p *api.Profile) error {
	return output.Profile(os.Stdout, p, jsonOut())
}

func init() {
	RootCmd.AddCommand(profileCmd)
}
