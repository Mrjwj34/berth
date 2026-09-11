package ports

import (
	"context"
	"net"
	"testing"
)

func TestAllocateSkipsBoundAndReserved(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	bound := ln.Addr().(*net.TCPAddr).Port

	reserved := map[int]string{Min: "other"}
	if bound >= Min && bound <= Max {
		reserved[bound] = "bound"
	}
	got, err := Allocate(context.Background(), reserved, []string{"web", "api"})
	if err != nil {
		t.Fatal(err)
	}
	if got["web"] == got["api"] {
		t.Fatalf("ports collided: %+v", got)
	}
	if got["web"] == Min || got["api"] == Min {
		t.Fatalf("reserved port reused: %+v", got)
	}
	if bound >= Min && bound <= Max && (got["web"] == bound || got["api"] == bound) {
		t.Fatalf("bound port reused: %+v", got)
	}
	if !Free(got["web"]) || !Free(got["api"]) {
		t.Fatalf("allocated ports not free: %+v", got)
	}
}

func TestAllocateRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Allocate(ctx, nil, []string{"web"}); err == nil {
		t.Fatal("expected context error")
	}
}
