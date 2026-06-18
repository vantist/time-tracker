package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/workitem"
)

func init() {
	rootCmd.AddCommand(workCmd)
	workCmd.Flags().Bool("clear", false, "Clear the current work item")
}

var workCmd = &cobra.Command{
	Use:   "work [label]",
	Short: "Set or display the current work item",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		clear, _ := cmd.Flags().GetBool("clear")

		if clear {
			if err := workitem.Clear(cwd); err != nil {
				return err
			}
			fmt.Println("Work item cleared.")
			return nil
		}

		if len(args) == 1 {
			if err := workitem.Set(args[0], cwd); err != nil {
				return err
			}
			fmt.Printf("Work item set: %s\n", args[0])
			return nil
		}

		// Show current
		label, err := workitem.Get(cwd)
		if err != nil {
			return err
		}
		if label == "" {
			fmt.Println("No work item set.")
		} else {
			fmt.Printf("Current work item: %s\n", label)
		}
		return nil
	},
}
