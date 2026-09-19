package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version/Commit are set by main() from build-time ldflags (goreleaser).
var (
	Version = "dev"
	Commit  = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the leakradar-cli CLI version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("leakradar-cli %s (%s)\n", Version, Commit)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
