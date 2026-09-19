package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var (
	emailSearchFlag  string
	emailOnlyFlag    bool
	usernameOnlyFlag bool
)

var emailCmd = &cobra.Command{
	Use:   "email <email-or-username>",
	Short: "Search leaks by email address or username",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if emailOnlyFlag && usernameOnlyFlag {
			return fmt.Errorf("--is-email and --is-username are mutually exclusive")
		}
		var isEmail *bool
		if emailOnlyFlag {
			v := true
			isEmail = &v
		} else if usernameOnlyFlag {
			v := false
			isEmail = &v
		}

		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchEmail(cmdContext(), api.EmailSearchRequest{
			Email:   args[0],
			Search:  emailSearchFlag,
			IsEmail: isEmail,
		})
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.Leaks(w, resp, false) })
		rec().Record(record.EmailPath(args[0], "search.json"), "email-search", args[0], redactedCommandLine(), resp, text)
		return output.Leaks(os.Stdout, resp, jsonOut())
	},
}

func init() {
	emailCmd.Flags().StringVar(&emailSearchFlag, "search", "", "free-text filter on URL or username")
	emailCmd.Flags().BoolVar(&emailOnlyFlag, "is-email", false, "restrict to email matches only")
	emailCmd.Flags().BoolVar(&usernameOnlyFlag, "is-username", false, "restrict to username matches only")
	RootCmd.AddCommand(emailCmd)
}
