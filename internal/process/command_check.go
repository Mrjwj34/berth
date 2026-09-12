package process

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Mrjwj34/berth/internal/config"
)

// WindowsCommandProblems reports process commands that cannot deliver their
// arguments intact when the native runtime starts them through process-compose
// on Windows.
//
// process-compose splits the command string on whitespace and passes the pieces
// as argv without honoring quotes: a quoted argument arrives with the quote
// characters still in it, and an argument containing a space arrives split in
// two. Both were confirmed by running a probe under process-compose 1.122.0 on
// Windows. The failure is confusing at runtime — a program receives half a path
// and reports an unrelated error — so berth explains it before starting
// anything. It is a warning rather than a refusal because a program that strips
// its own quotes, or one that never receives a space, still works.
//
// Values come from env, so a checkout path containing a space is detected even
// though the configuration itself looks fine. Container workspaces are exempt:
// their commands run inside Linux, where the shell handles quoting.
func WindowsCommandProblems(processes map[string]any, env map[string]string) []string {
	if len(processes) == 0 {
		return nil
	}
	spaced := spacedValues(env)
	names := make([]string, 0, len(processes))
	for name := range processes {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []string
	for _, name := range names {
		proc, ok := processes[name].(map[string]any)
		if !ok {
			continue
		}
		command, ok := proc["command"].(string)
		if !ok || strings.TrimSpace(command) == "" {
			continue
		}
		expanded := config.Expand(command, env)

		var reasons []string
		if strings.Contains(expanded, `"`) {
			reasons = append(reasons, "it contains a double quote, which arrives as part of the argument instead of grouping words")
		}
		for _, value := range spaced {
			if strings.Contains(expanded, value) {
				reasons = append(reasons, fmt.Sprintf("it passes %s, which contains a space and will be split into separate arguments", value))
			}
		}
		if len(reasons) > 0 {
			out = append(out, fmt.Sprintf("processes.%s: %s", name, strings.Join(reasons, "; ")))
		}
	}
	return out
}

// spacedValues returns the distinct environment values that contain whitespace,
// longest first so a value that contains another one is reported as itself.
func spacedValues(env map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range env {
		if value == "" || !strings.ContainsAny(value, " \t") || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// WindowsCommandRemedy is the fix users apply to the problems above. It is kept
// next to the detection so the message and the documentation cannot drift.
const WindowsCommandRemedy = "Windows starts these commands through process-compose, which splits the command on spaces and does not strip quotes. Pass paths relative to the workspace root (berth sets working_dir there), read BERTH_DATA_DIR from the environment inside your program, or move the checkout to a path without spaces"
