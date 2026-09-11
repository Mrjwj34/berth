//go:build linux

package netns

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const tcpListen = "0A"

// Discover returns TCP ports in LISTEN state inside the workspace netns.
func Discover(worktree string) ([]int, error) {
	pid := InsidePID(worktree)
	if pid <= 0 || !alive(pid) {
		return nil, ErrNotRunning
	}
	seen := map[int]struct{}{}
	for _, table := range []string{"tcp", "tcp6"} {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/net/%s", pid, table))
		if err != nil {
			continue
		}
		for _, port := range parseProcNetListen(string(data)) {
			seen[port] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for port := range seen {
		out = append(out, port)
	}
	sort.Ints(out)
	return out, nil
}

func parseProcNetListen(data string) []int {
	var out []int
	sc := bufio.NewScanner(strings.NewReader(data))
	if sc.Scan() {
		// header
	}
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 || !strings.EqualFold(fields[3], tcpListen) {
			continue
		}
		_, portHex, ok := strings.Cut(fields[1], ":")
		if !ok {
			continue
		}
		port, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil || port == 0 {
			continue
		}
		out = append(out, int(port))
	}
	return out
}
