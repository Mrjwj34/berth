package main

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/Mrjwj34/berth/internal/app"
	"github.com/Mrjwj34/berth/internal/skill"
	"github.com/spf13/cobra"
)

func cmdInit(asJSON *bool) *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Write a commented berth.yaml and install the skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				if err := a.Init(ctx, force); err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), map[string]any{"ok": true, "file": "berth.yaml"})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "wrote berth.yaml and installed .agents/skills/berth/SKILL.md. Edit ports/processes, then: berth new smoke --up")
				return nil
			})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing berth.yaml")
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
	c.Flags().StringVar(&base, "base", "", "baseline branch (default: berth.yaml base or main)")
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
				keys := make([]string, 0, len(ws.Env))
				for k := range ws.Env {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					v := ws.Env[k]
					if runtime.GOOS == "windows" {
						fmt.Fprintf(cmd.OutOrStdout(), "$env:%s = '%s'\n", k, strings.ReplaceAll(v, "'", "''"))
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "export %s='%s'\n", k, strings.ReplaceAll(v, "'", "'\"'\"'"))
					}
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
		Short: "Register the current git worktree as a berth workspace",
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
					return writeJSON(cmd.OutOrStdout(), map[string]any{"slug": ws.Slug, "path": ws.Path, "ports": ws.Ports, "listen": ws.Listen})
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
		Short: "Run a command with BERTH_* injected",
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
		Short: "Wipe BERTH_DATA_DIR and rerun setup hooks",
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
				fmt.Fprintln(cmd.OutOrStdout(), "workspace released (adopted checkouts are preserved)")
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
				for _, warning := range rep.Warnings {
					fmt.Fprintln(cmd.ErrOrStderr(), "warning:", warning)
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
	c.Flags().BoolVar(&fix, "fix", false, "install missing native dependencies; never delete workspaces")
	return c
}

func cmdSkill() *cobra.Command {
	c := &cobra.Command{Use: "skill", Short: "Skill asset commands"}
	var agents []string
	var all bool
	var scope string
	install := &cobra.Command{
		Use:   "install",
		Short: "Install the embedded skill into the selected harness directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := skill.Options{Agents: agents, All: all, Scope: scope}
			return withApp(func(ctx context.Context, a *app.App) error {
				results, err := a.SkillInstall(ctx, opts)
				if err != nil {
					return err
				}
				printInstalls(cmd.OutOrStdout(), results)
				return nil
			})
		},
	}
	install.Flags().StringArrayVar(&agents, "agent", nil, "harness to install for; repeatable or comma-separated. Default: the shared "+skill.SharedProjectPath)
	install.Flags().BoolVar(&all, "all", false, "install for every supported harness")
	install.Flags().StringVar(&scope, "scope", "project", "project (this repository) or user (your home directory)")
	c.AddCommand(install)
	return c
}

func cmdHook() *cobra.Command {
	c := &cobra.Command{Use: "hook", Short: "Optional harness adapters"}
	var agents []string
	var all bool
	var scope string
	install := &cobra.Command{
		Use:   "install [cursor|windsurf|claude|all]",
		Short: "Merge berth's worktree adapter into the selected harness configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := skill.Options{Agents: agents, All: all, Scope: scope}
			if len(args) == 1 {
				if args[0] == "all" {
					opts.All = true
				} else {
					opts.Agents = append(opts.Agents, args[0])
				}
			}
			normalized, err := opts.Normalize()
			if err != nil {
				return err
			}
			if normalized.Scope == skill.ScopeUser {
				fmt.Fprintln(cmd.OutOrStdout(), "hooks are project-scoped: a worktree is created inside a repository, so berth has nothing to write at user scope. Nothing was written; re-run without --scope user")
				return nil
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				results, err := a.HookInstall(ctx, opts)
				if err != nil {
					return err
				}
				printInstalls(cmd.OutOrStdout(), results)
				return nil
			})
		},
	}
	install.Flags().StringArrayVar(&agents, "agent", nil, "harness worktree hook to install; repeatable or comma-separated. Default: every hook berth installs (cursor, windsurf)")
	install.Flags().BoolVar(&all, "all", false, "install every hook berth installs (claude is opt-in and stays opt-in)")
	install.Flags().StringVar(&scope, "scope", "project", "hooks are project-scoped; user is refused with a message")
	c.AddCommand(install)
	return c
}

// printInstalls prints one tab-separated line per artifact, in the order the
// install visited them, so the output is deterministic and machine-readable.
func printInstalls(w io.Writer, results []app.InstallResult) {
	if len(results) == 0 {
		fmt.Fprintln(w, "nothing to install")
		return
	}
	for _, r := range results {
		status := "already installed"
		if r.Changed {
			status = "installed"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Scope, r.Agent, r.Path, status)
	}
}

func cmdAgents(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List supported harnesses and what is installed in this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				rep, err := a.Agents(ctx)
				if err != nil {
					return err
				}
				if *asJSON {
					return writeJSON(cmd.OutOrStdout(), rep)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "repo\t%s\n", rep.Repo)
				fmt.Fprintf(cmd.OutOrStdout(), "home\t%s\n", rep.Home)
				fmt.Fprintln(cmd.OutOrStdout(), "NAME\tSHARED\tPROJECT\tUSER\tHOOK\tINSTALLED\tNOTE")
				for _, row := range rep.Agents {
					hook := "-"
					if row.Hook != "" {
						hook = row.Hook + " " + row.HookFile
						if row.HookOptIn {
							hook += " (opt-in)"
						}
					}
					installed := fmt.Sprintf("project=%s user=%s", yesNo(row.InstalledProject), yesNo(row.InstalledUser))
					if row.Hook != "" {
						installed += " hook=" + yesNo(row.HookInstalled)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						row.Name, yesNo(row.ReadsShared), row.ProjectPath, row.UserPath, hook, installed, row.Note)
				}
				return nil
			})
		},
	}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
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
