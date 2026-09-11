package ports

import (
	"context"
	"fmt"
	"net"
)

const (
	Min = 20000
	Max = 39999
)

func Allocate(ctx context.Context, reserved map[int]string, names []string) (map[string]int, error) {
	out := make(map[string]int, len(names))
	taken := make(map[int]struct{}, len(reserved)+len(names))
	for p := range reserved {
		taken[p] = struct{}{}
	}
	next := Min
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		port, err := nextFree(ctx, next, taken)
		if err != nil {
			return nil, fmt.Errorf("allocate port %q: %w. Free a port in %d-%d or stop unused workspaces with lane down / lane gc", name, err, Min, Max)
		}
		out[name] = port
		taken[port] = struct{}{}
		next = port + 1
	}
	return out, nil
}

func nextFree(ctx context.Context, start int, taken map[int]struct{}) (int, error) {
	if start < Min {
		start = Min
	}
	for port := start; port <= Max; port++ {
		if _, used := taken[port]; used {
			continue
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if Free(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free TCP port in %d-%d", Min, Max)
}

func Free(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	// Closing is not always instant on some stacks; a short pause is unnecessary
	// because we only need the probe result.
	return true
}
