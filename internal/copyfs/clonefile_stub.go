//go:build !darwin

package copyfs

import "errors"

func clonefile(_, _ string) error {
	return errors.New("clonefile: unsupported")
}
