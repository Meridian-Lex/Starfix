package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "starfix",
		Short: "Post-compaction context restoration for Lex",
	}

	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Hook subcommands",
	}

	hookCmd.AddCommand(
		&cobra.Command{
			Use:   "precompact",
			Short: "Handle PreCompact hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "precompact: not implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "sessionstart",
			Short: "Handle SessionStart hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "sessionstart: not implemented")
				return nil
			},
		},
		&cobra.Command{
			Use:   "userpromptsubmit",
			Short: "Handle UserPromptSubmit hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(os.Stderr, "userpromptsubmit: not implemented")
				return nil
			},
		},
	)

	watchCmd := &cobra.Command{
		Use:   "watch-reply [session_id]",
		Short: "Watch for Telegram reply and execute timeout action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "watch-reply: not implemented")
			return nil
		},
	}

	root.AddCommand(hookCmd, watchCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
