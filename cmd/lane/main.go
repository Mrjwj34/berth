package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Mrjwj34/lane/internal/version"
)

var (
	appVersion = "0.1.0-dev"
	commit     = "none"
	date       = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "lane - Project-agnostic, harness-agnostic workspace runner for AI agents and developers\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  lane [command] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  init       Scaffold lane.yaml in current repository\n")
		fmt.Fprintf(os.Stderr, "  new        Create and initialize a new isolated workspace\n")
		fmt.Fprintf(os.Stderr, "  attach     Attach or switch to an existing workspace\n")
		fmt.Fprintf(os.Stderr, "  ls         List all workspaces and their status\n")
		fmt.Fprintf(os.Stderr, "  status     Show process status for the current workspace\n")
		fmt.Fprintf(os.Stderr, "  ports      Show port allocation for the current workspace\n")
		fmt.Fprintf(os.Stderr, "  up         Start workspace processes\n")
		fmt.Fprintf(os.Stderr, "  down       Stop workspace processes\n")
		fmt.Fprintf(os.Stderr, "  logs       View logs of a workspace process\n")
		fmt.Fprintf(os.Stderr, "  run        Execute a command in workspace environment\n")
		fmt.Fprintf(os.Stderr, "  reset      Reset workspace data and rerun setup\n")
		fmt.Fprintf(os.Stderr, "  done       Clean up and remove a finished workspace\n")
		fmt.Fprintf(os.Stderr, "  gc         Run garbage collection for orphaned/idle workspaces\n")
		fmt.Fprintf(os.Stderr, "  doctor     Inspect environment health and verify prerequisites\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	info := version.Current(appVersion, commit, date)

	if *showVersion {
		fmt.Printf("lane %s (commit: %s, date: %s)\n", info.Version, info.Commit, info.Date)
		return
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "version":
		fmt.Printf("lane %s (commit: %s, date: %s)\n", info.Version, info.Commit, info.Date)
	case "help":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "lane: '%s' is not implemented yet in initial skeleton. Run 'lane help' for available commands.\n", cmd)
		os.Exit(1)
	}
}
