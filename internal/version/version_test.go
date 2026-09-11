package version

import (
	"testing"
)

func TestCurrent(t *testing.T) {
	info := Current("", "", "")
	if info.Version != "0.1.0-dev" {
		t.Fatalf("expected 0.1.0-dev, got %s", info.Version)
	}
	if info.Commit != "none" {
		t.Fatalf("expected none, got %s", info.Commit)
	}
	if info.Date != "unknown" {
		t.Fatalf("expected unknown, got %s", info.Date)
	}

	custom := Current("v1.0.0", "abc1234", "2026-09-11")
	if custom.Version != "v1.0.0" || custom.Commit != "abc1234" || custom.Date != "2026-09-11" {
		t.Fatalf("unexpected custom info: %+v", custom)
	}
}
