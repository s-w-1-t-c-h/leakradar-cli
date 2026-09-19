package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
)

// printTaskResult prints an async unlock-task response and, when it carries
// a task_id, hints at how to poll it.
func printTaskResult(task api.AsyncTaskResult) error {
	if err := output.JSON(os.Stdout, task); err != nil {
		return err
	}
	if id := task.TaskID(); id != "" {
		fmt.Fprintf(os.Stderr, "poll with: leakradar-cli task status %s\n", output.TerminalSafe(id))
	}
	return nil
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Check background task status (e.g. an async unlock queued with --async)",
}

var taskStatusCmd = &cobra.Command{
	Use:   "status <task-id>",
	Short: "Poll a background task until you check it (does not block/loop)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		status, err := c.GetTaskStatus(cmdContext(), args[0])
		if err != nil {
			return err
		}
		if jsonOut() {
			return output.JSON(os.Stdout, status)
		}
		fmt.Printf("task_id=%s running=%v completed=%v", output.TerminalSafe(status.TaskID), status.Running, status.Completed)
		if status.Total != nil {
			fmt.Printf(" total=%d", *status.Total)
		}
		if status.Updated != nil {
			fmt.Printf(" updated=%d", *status.Updated)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskStatusCmd)
	RootCmd.AddCommand(taskCmd)
}
