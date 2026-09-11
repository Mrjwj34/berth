//go:build windows

package remap

import (
	"context"
	"fmt"
)

func Supervise(context.Context, []string) error {
	return fmt.Errorf("%w", errUnsupported)
}

func Spawn(context.Context, string, string, []string, []string) error {
	return fmt.Errorf("%w", errUnsupported)
}

func Stop(string) error { return nil }
