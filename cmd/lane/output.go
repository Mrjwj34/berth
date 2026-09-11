package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Mrjwj34/lane/internal/app"
)

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printWorkspace(w io.Writer, ws *app.WorkspaceView, asJSON bool) error {
	if asJSON {
		return writeJSON(w, ws)
	}
	fmt.Fprintf(w, "slug\t%s\n", ws.Slug)
	fmt.Fprintf(w, "path\t%s\n", ws.Path)
	fmt.Fprintf(w, "branch\t%s\n", ws.Branch)
	if len(ws.Ports) > 0 {
		fmt.Fprintf(w, "ports\t%s\n", formatPorts(ws.Ports, ws.Listen))
	}
	fmt.Fprintf(w, "running\t%v\n", ws.Running)
	return nil
}

func formatPorts(ports, listen map[string]int) string {
	parts := make([]string, 0, len(ports))
	for name, port := range ports {
		if l := listen[name]; l > 0 && l != port {
			parts = append(parts, fmt.Sprintf("%s=%d->%d", name, port, l))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%d", name, port))
		}
	}
	return strings.Join(parts, " ")
}

func printOverview(w io.Writer, list []app.WorkspaceView, asJSON bool) error {
	if asJSON {
		return writeJSON(w, map[string]any{"workspaces": list})
	}
	if len(list) == 0 {
		fmt.Fprintln(w, "no workspaces. Run lane new <slug> or lane adopt")
		return nil
	}
	fmt.Fprintf(w, "%-16s %-20s %-8s %-24s %s\n", "SLUG", "BRANCH", "STATUS", "PORTS", "PATH")
	for _, ws := range list {
		status := "stopped"
		if ws.Running {
			status = "up"
		}
		if ws.Dirty {
			status += "*"
		}
		fmt.Fprintf(w, "%-16s %-20s %-8s %-24s %s\n", ws.Slug, ws.Branch, status, formatPorts(ws.Ports, ws.Listen), ws.Path)
	}
	return nil
}
