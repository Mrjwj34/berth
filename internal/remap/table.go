package remap

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mrjwj34/lane/internal/ports"
)

type table struct {
	Maps     []Mapping
	Reserved map[int]struct{}
}

func loadTable(path string) (table, error) {
	out := table{Reserved: map[int]struct{}{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return table{}, err
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#reserved ") {
			p, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "#reserved ")))
			if err == nil && p > 0 {
				out.Reserved[p] = struct{}{}
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		fromS, toS, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		from, err1 := strconv.Atoi(strings.TrimSpace(fromS))
		to, err2 := strconv.Atoi(strings.TrimSpace(toS))
		if err1 != nil || err2 != nil || from <= 0 || to <= 0 {
			continue
		}
		out.Maps = append(out.Maps, Mapping{Listen: from, Host: to, Name: strconv.Itoa(from)})
	}
	return out, sc.Err()
}

func writeTable(path string, t table) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# lane remap listen=host\n")
	seenRes := map[int]struct{}{}
	for p := range t.Reserved {
		if p <= 0 {
			continue
		}
		if _, ok := seenRes[p]; ok {
			continue
		}
		seenRes[p] = struct{}{}
		fmt.Fprintf(&b, "#reserved %d\n", p)
	}
	seen := map[int]struct{}{}
	for _, m := range t.Maps {
		if m.Listen <= 0 || m.Host <= 0 {
			continue
		}
		if _, ok := seen[m.Listen]; ok {
			continue
		}
		seen[m.Listen] = struct{}{}
		fmt.Fprintf(&b, "%d=%d\n", m.Listen, m.Host)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func seedTable(path string, maps []Mapping, reserved []int) error {
	return withTableLock(path, func() error {
		t, err := loadTable(path)
		if err != nil {
			return err
		}
		if t.Reserved == nil {
			t.Reserved = map[int]struct{}{}
		}
		for _, p := range reserved {
			if p > 0 {
				t.Reserved[p] = struct{}{}
			}
		}
		byListen := map[int]Mapping{}
		for _, m := range t.Maps {
			byListen[m.Listen] = m
		}
		for _, m := range maps {
			if m.Listen <= 0 || m.Host <= 0 {
				continue
			}
			byListen[m.Listen] = m
			t.Reserved[m.Host] = struct{}{}
		}
		t.Maps = t.Maps[:0]
		for _, m := range byListen {
			t.Maps = append(t.Maps, m)
		}
		return writeTable(path, t)
	})
}

func lookupHost(path string, listen int) (int, bool) {
	t, err := loadTable(path)
	if err != nil {
		return 0, false
	}
	for _, m := range t.Maps {
		if m.Listen == listen {
			return m.Host, true
		}
	}
	return 0, false
}

func reverseListen(path string, host int) (int, bool) {
	t, err := loadTable(path)
	if err != nil {
		return 0, false
	}
	for _, m := range t.Maps {
		if m.Host == host {
			return m.Listen, true
		}
	}
	return 0, false
}

func allocListen(path string, listen int) (int, error) {
	if listen <= 0 || (listen >= ports.Min && listen <= ports.Max) {
		return listen, nil
	}
	var host int
	err := withTableLock(path, func() error {
		t, err := loadTable(path)
		if err != nil {
			return err
		}
		for _, m := range t.Maps {
			if m.Listen == listen {
				host = m.Host
				return nil
			}
		}
		taken := map[int]struct{}{}
		for p := range t.Reserved {
			taken[p] = struct{}{}
		}
		for _, m := range t.Maps {
			taken[m.Host] = struct{}{}
			taken[m.Listen] = struct{}{}
		}
		start := ports.Min + (listen % 1000)
		p, err := nextFree(start, taken)
		if err != nil {
			return err
		}
		t.Maps = append(t.Maps, Mapping{Listen: listen, Host: p, Name: strconv.Itoa(listen)})
		if err := writeTable(path, t); err != nil {
			return err
		}
		host = p
		return nil
	})
	return host, err
}

func nextFree(start int, taken map[int]struct{}) (int, error) {
	if start < ports.Min {
		start = ports.Min
	}
	for port := start; port <= ports.Max; port++ {
		if _, used := taken[port]; used {
			continue
		}
		if ports.Free(port) {
			return port, nil
		}
	}
	for port := ports.Min; port < start; port++ {
		if _, used := taken[port]; used {
			continue
		}
		if ports.Free(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free TCP port in %d-%d for listen remap. Free a port or run lane gc", ports.Min, ports.Max)
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, string(os.PathSeparator)); i >= 0 {
		return path[:i]
	}
	return "."
}

// Mappings returns listen→host rows from the workspace remap table.
func Mappings(worktree string) []Mapping {
	t, err := loadTable(TablePath(worktree))
	if err != nil {
		return nil
	}
	return t.Maps
}
