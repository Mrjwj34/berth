//go:build !linux

package netns

import (
	"context"
	"fmt"
	"runtime"
)

var (
	ErrNotRunning = fmt.Errorf("workspace network namespace is not running. Start it with lane up")
	ErrNoNsenter  = fmt.Errorf("workspace network namespaces are only available on Linux")
)

func Available() bool { return false }

func Start(context.Context, string, []Mapping, bool, string, []string, []string) error {
	return fmt.Errorf("isolate: net remaps hardcoded process ports via a Linux network namespace. This platform (%s) cannot run two processes that bind the same port. Use Linux, run a single workspace, or pass the process a unique --port $LANE_PORT_*", runtime.GOOS)
}

func Stop(string) error { return nil }

func Exec(context.Context, string, []string, []string) error {
	return ErrNoNsenter
}

func Discover(string) ([]int, error) { return nil, nil }

func Publish(string, Mapping) error {
	return fmt.Errorf("isolate: net is only available on Linux")
}
