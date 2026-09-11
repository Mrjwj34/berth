//go:build !linux

package copyfs

import (
	"errors"
	"os"
)

func reflink(_, _ string, _ os.FileMode) error { return errors.New("reflink unavailable") }
