package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"leakradar-cli/internal/config"
	"leakradar-cli/internal/output"
)

var authSetKeyFlag string

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage the stored LeakRadar API key",
}

var authSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Store an API key (OS keychain, or a 0600 config file if no keychain is available)",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := authSetKeyFlag
		if key == "" {
			if info, _ := os.Stdin.Stat(); info != nil && (info.Mode()&os.ModeCharDevice) == 0 {
				// stdin is piped, e.g. `echo $KEY | leakradar-cli auth set`
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					key = strings.TrimSpace(scanner.Text())
				}
			} else {
				fmt.Fprint(os.Stderr, "LeakRadar API key: ")
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				if err != nil {
					return fmt.Errorf("reading key: %w", err)
				}
				key = strings.TrimSpace(string(b))
			}
		}
		if key == "" {
			return fmt.Errorf("no key provided")
		}

		inKeyring, err := config.SetAPIKey(key)
		if err != nil {
			return err
		}
		if inKeyring {
			fmt.Fprintln(os.Stderr, "API key stored in the OS keychain.")
		} else {
			fmt.Fprintln(os.Stderr, "OS keychain unavailable; stored in config file with restricted permissions instead.")
		}
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Validate the stored API key against the LeakRadar API",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(os.Stderr, "key source: %s\n", output.TerminalSafe(config.Source()))
		c, err := newClient()
		if err != nil {
			return err
		}
		p, err := c.GetProfile(cmdContext())
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "key is valid.")
		return printProfile(p)
	},
}

var authClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove the stored API key from the keychain and config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearAPIKey(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "API key removed.")
		return nil
	},
}

func init() {
	authSetCmd.Flags().StringVar(&authSetKeyFlag, "key", "", "API key value (omit to be prompted, or pipe it via stdin)")
	authCmd.AddCommand(authSetCmd, authStatusCmd, authClearCmd)
	RootCmd.AddCommand(authCmd)
}
