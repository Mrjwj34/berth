//go:build linux

package netns

import "testing"

func TestParseProcNetListen(t *testing.T) {
	data := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1
   1: 00000000:0050 00000000:0000 01 00000000:00000000 00:00000000 00000000     0        0 2
`
	got := parseProcNetListen(data)
	if len(got) != 1 || got[0] != 8080 {
		t.Fatalf("got %v", got)
	}
}
