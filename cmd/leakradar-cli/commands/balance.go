package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show remaining unlock credits (points) and live plan quota usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		p, err := c.GetProfile(cmdContext())
		if err != nil {
			return err
		}

		view := output.BalanceView{
			Points:          p.SubscriptionPoints,
			ExtraPoints:     p.ExtraPoints,
			TotalPoints:     p.SubscriptionPoints + p.ExtraPoints,
			PlanName:        p.Plan.PlanName,
			SubscriptionEnd: p.SubscriptionEndDate,
		}
		if usage, err := c.GetUsage(cmdContext()); err != nil {
			if flagVerbose {
				fmt.Fprintf(os.Stderr, "warning: could not fetch /profile/usage: %s\n", output.TerminalSafe(err.Error()))
			}
		} else {
			view.Usage = usage
		}

		text := renderText(func(w io.Writer) error { return output.Balance(w, view, false) })
		rec().Record(record.TimestampPath("balance", ""), "balance", "", redactedCommandLine(), view, text)
		return output.Balance(os.Stdout, view, jsonOut())
	},
}

func init() {
	RootCmd.AddCommand(balanceCmd)
}
