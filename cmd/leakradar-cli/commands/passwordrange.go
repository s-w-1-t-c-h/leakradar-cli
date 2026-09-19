package commands

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var prOpts struct {
	limit      int
	suffixOnly bool
}

var passwordRangeCmd = &cobra.Command{
	Use:   "password-range <sha1-prefix>",
	Short: "K-anonymity SHA-1 prefix lookup for leaked password hashes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.PasswordRange(cmdContext(), args[0], prOpts.limit, prOpts.suffixOnly)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.PasswordRange(w, resp, false) })
		rec().Record(record.TargetPath("password-range", args[0], "result.json"), "password-range", args[0], redactedCommandLine(), resp, text)
		return output.PasswordRange(os.Stdout, resp, jsonOut())
	},
}

func init() {
	passwordRangeCmd.Flags().IntVar(&prOpts.limit, "limit", 1000, "max hashes to return (max 10000)")
	passwordRangeCmd.Flags().BoolVar(&prOpts.suffixOnly, "suffix-only", false, "return hash suffixes only")
	RootCmd.AddCommand(passwordRangeCmd)
}
