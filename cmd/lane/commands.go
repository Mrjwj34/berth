package main

import (
	"context"
	"fmt"

	"github.com/Mrjwj34/lane/internal/app"
	"github.com/spf13/cobra"
)

func cmdInit(asJSON *bool) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Write a commented lane.yaml and install the skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				if err := a.Init(ctx, force); err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), map[string]any{"ok": true, "file": "lane.yaml"})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "wrote lane.yaml and installed skill. Edit ports/processes, then: lane new smoke --up")
				return nil
			})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing lane.yaml")
	return c
}

func cmdNew(asJSON *bool) *cobra.Command {
	var up bool
	var base string
	var printPath bool
	c := &cobra.Command{
		Use:   "new <slug>",
		Short: "Create an isolated worktree workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				ws, err := a.New(ctx, args[0], base, up)
				if err != nil {
					return err
				}
				if printPath {
					fmt.Fprintln(cmd.OutOrStdout(), ws.Path)
					return nil
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), ws)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", ws.Slug)
				fmt.Fprintf(cmd.OutOrStdout(), "path\t%s\n", ws.Path)
				fmt.Fprintf(cmd.OutOrStdout(), "branch\t%s\n", ws.Branch)
				if len(ws.Ports) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "ports\t%s\n", formatPorts(ws.Ports))
				}
				fmt.Fprintln(cmd.OutOrStdout(), ws.Path)
				return nil
			})
		},
	}
	c.Flags().BoolVar(&up, "up", false, "start processes after setup")
	c.Flags().StringVar(&base, "base", "", "baseline branch (default: lane.yaml base or main)")
	c.Flags().BoolVar(&printPath, "print-path", false, "print only the absolute worktree path")
	return c
}

func cmdAttach(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "attach [slug]",
		Short: "Print path and environment for a workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				ws, err := a.Attach(ctx, slug)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), ws)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "path\t%s\n", ws.Path)
				fmt.Fprintf(cmd.OutOrStdout(), "branch\t%s\n", ws.Branch)
				for k, v := range ws.Env {
					fmt.Fprintf(cmd.OutOrStdout(), "export %s=%q\n", k, v)
				}
				return nil
			})
		},
	}
}

func cmdAdopt(asJSON *bool) *cobra.Command {
	var setup bool
	c := &cobra.Command{
		Use:   "adopt",
		Short: "Register the current git worktree as a lane workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				ws, err := a.Adopt(ctx, setup)
				if err != nil {
					return err
				}
				return printWorkspace(cmd.OutOrStdout(), ws, *asJSON)
			})
		},
	}
	c.Flags().BoolVar(&setup, "setup", false, "run include/copy_dirs/setup hooks")
	return c
}

func cmdLS(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				list, err := a.List(ctx)
				if err != nil {
					return err
				}
				return printOverview(cmd.OutOrStdout(), list, *asJSON)
			})
		},
	}
}

func cmdStatus(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "status [slug]",
		Short: "Show process and port status for a workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				ws, err := a.Status(ctx, slug)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), ws)
				}
				_ = printWorkspace(cmd.OutOrStdout(), ws, false)
				for _, p := range ws.Processes {
					fmt.Fprintf(cmd.OutOrStdout(), "proc\t%s\t%s\tready=%s\tpid=%d\n", p.Name, p.Status, p.Ready, p.PID)
				}
				return nil
			})
		},
	}
}

func cmdPorts(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "ports [slug]",
		Short: "Show allocated ports",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				ws, err := a.Status(ctx, slug)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), map[string]any{"slug": ws.Slug, "path": ws.Path, "ports": ws.Ports})
				}
				for name, port := range ws.Ports {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\n", name, port)
				}
				return nil
			})
		},
	}
}

func cmdUp() *cobra.Command {
	return &cobra.Command{
		Use:   "up [slug]",
		Short: "Start workspace processes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				return a.Up(ctx, slug)
			})
		},
	}
}

func cmdDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down [slug]",
		Short: "Stop workspace processes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				return a.Down(ctx, slug)
			})
		},
	}
}

func cmdLogs() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <proc> [slug]",
		Short: "Show logs for a managed process",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			slug := ""
			if len(args) == 2 {
				slug = args[1]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				return a.Logs(ctx, slug, name)
			})
		},
	}
}

func cmdRun() *cobra.Command {
	return &cobra.Command{
		Use:   "run [--] <cmd> [args...]",
		Short: "Run a command with LANE_* injected",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				return a.Run(ctx, "", args)
			})
		},
	}
}

func cmdReset() *cobra.Command {
	return &cobra.Command{
		Use:   "reset [slug]",
		Short: "Wipe LANE_DATA_DIR and rerun setup hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				return a.Reset(ctx, slug)
			})
		},
	}
}

func cmdDone() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "done [slug]",
		Short: "Tear down a workspace after work is preserved",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				if err := a.Done(ctx, slug, force); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "removed")
				return nil
			})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "discard dirty or unpublished work (requires explicit user authorization)")
	return c
}

func cmdGC(asJSON *bool) *cobra.Command {
	var dry bool
	c := &cobra.Command{
		Use:   "gc",
		Short: "Reclaim vanished, idle, merged, or excess workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				rep, err := a.GC(ctx, dry)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), rep)
				}
				if len(rep.Actions) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "nothing to collect")
					return nil
				}
				for _, act := range rep.Actions {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", act.Kind, act.Slug, act.Reason)
				}
				return nil
			})
		},
	}
	c.Flags().BoolVar(&dry, "dry-run", false, "print actions without applying them")
	return c
}

func cmdDoctor(asJSON *bool) *cobra.Command {
	var fix bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose git, process-compose, and state health",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				rep, err := a.Doctor(ctx, fix)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), rep)
				}
				for _, c := range rep.Checks {
					mark := "ok"
					if !c.OK {
						mark = "FAIL"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", c.Name, mark, c.Detail)
				}
				if !rep.OK {
					return fmt.Errorf("doctor found problems. Re-run with --fix or install the missing tool")
				}
				return nil
			})
		},
	}
	c.Flags().BoolVar(&fix, "fix", false, "download process-compose and reclaim vanished workspaces")
	return c
}

func cmdSkill() *cobra.Command {
	c := &cobra.Command{Use: "skill", Short: "Skill asset commands"}
	c.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install the embedded SKILL.md into agent skill directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				if err := a.SkillInstall(ctx); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "installed skill into .agents/skills/lane, .claude/skills/lane, .cursor/skills/lane")
				return nil
			})
		},
	})
	return c
}

func cmdHook() *cobra.Command {
	c := &cobra.Command{Use: "hook", Short: "Optional harness adapters"}
	c.AddCommand(&cobra.Command{
		Use:   "install [cursor|claude|all]",
		Short: "Write Cursor worktrees.json and/or Claude Code hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			which := "all"
			if len(args) == 1 {
				which = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				if err := a.HookInstall(ctx, which); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "installed %s hooks\n", which)
				return nil
			})
		},
	})
	return c
}

func cmdOpen(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "open [port-name]",
		Short: "Open a workspace HTTP port in the browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				url, err := a.Open(ctx, "", name)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), map[string]string{"url": url})
				}
				fmt.Fprintln(cmd.OutOrStdout(), url)
				return nil
			})
		},
	}
}
