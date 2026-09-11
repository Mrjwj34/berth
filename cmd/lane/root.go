package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Mrjwj34/lane/internal/app"
	"github.com/Mrjwj34/lane/internal/version"
	"github.com/spf13/cobra"
)

func newRoot(info version.Info) *cobra.Command {
	var asJSON bool
	root := &cobra.Command{
		Use:           "lane",
		Short:         "Project-agnostic, harness-agnostic workspace runner for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				list, err := a.List(ctx)
				if err != nil {
					return err
				}
				return printOverview(cmd.OutOrStdout(), list, asJSON)
			})
		},
	}
	root.PersistentFlags().BoolVar(&asJSON, "json", false, "stable JSON output")
	root.Version = fmt.Sprintf("%s (commit: %s, date: %s)", info.Version, info.Commit, info.Date)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	root.AddCommand(
		cmdInit(&asJSON),
		cmdNew(&asJSON),
		cmdAttach(&asJSON),
		cmdAdopt(&asJSON),
		cmdLS(&asJSON),
		cmdStatus(&asJSON),
		cmdPorts(&asJSON),
		cmdUp(),
		cmdDown(),
		cmdLogs(),
		cmdRun(),
		cmdReset(),
		cmdDone(),
		cmdGC(&asJSON),
		cmdDoctor(&asJSON),
		cmdSkill(),
		cmdHook(),
		cmdOpen(&asJSON),
		cmdRemapSupervise(),
		cmdVersion(info),
	)
	return root
}

func withApp(fn func(context.Context, *app.App) error) error {
	ctx := context.Background()
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	return fn(ctx, a)
}

func cmdVersion(info version.Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "lane %s (commit: %s, date: %s)\n", info.Version, info.Commit, info.Date)
		},
	}
}
