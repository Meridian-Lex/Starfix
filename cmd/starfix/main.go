package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/meridian-lex/starfix/internal/config"
	"github.com/meridian-lex/starfix/internal/hook"
	"github.com/meridian-lex/starfix/internal/state"
)

func main() {
	root := &cobra.Command{
		Use:   "starfix",
		Short: "Post-compaction context restoration for Lex",
	}

	hookCmd := &cobra.Command{Use: "hook", Short: "Hook subcommands"}

	hookCmd.AddCommand(
		&cobra.Command{
			Use:   "precompact",
			Short: "Handle PreCompact hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					hook.HandlePreCompact(input, cfg, state.DefaultBaseDir())
					return ""
				})
			},
		},
		&cobra.Command{
			Use:   "sessionstart",
			Short: "Handle SessionStart hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					return hook.HandleSessionStart(input, cfg, state.DefaultBaseDir())
				})
			},
		},
		&cobra.Command{
			Use:   "userpromptsubmit",
			Short: "Handle UserPromptSubmit hook event",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(func(input hook.Input, cfg *config.Config) string {
					return hook.HandleUserPromptSubmit(input, cfg, state.DefaultBaseDir())
				})
			},
		},
	)

	watchCmd := &cobra.Command{
		Use:   "watch-reply [session_id]",
		Short: "Watch for Telegram reply and execute timeout action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				cfg = &config.Config{}
			}
			hook.RunWatchReply(args[0], cfg, state.DefaultBaseDir())
			return nil
		},
	}

	root.AddCommand(hookCmd, watchCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runHook(fn func(hook.Input, *config.Config) string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	input, err := hook.ReadInput(data)
	if err != nil {
		return fmt.Errorf("parse hook input: %w", err)
	}

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		cfg = &config.Config{}
	}

	if output := fn(input, cfg); output != "" {
		fmt.Print(output)
	}
	return nil
}
