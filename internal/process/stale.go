package process

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Recovering a supervisor that exited on its own: a project that completed, or a
// process that failed during startup, leaves its control files behind, because
// only a successful conversation with the supervisor ever removed them. Treating
// that as an unresponsive supervisor blocks every destructive operation forever
// and leaves hand-deleting files as the only recovery.

const (
	probeTimeout = 500 * time.Millisecond
	// A supervisor that was just spawned may still be binding its endpoint, so a
	// single refusal is not proof that it is gone. The gap is only ever paid on
	// the recovery path, never on a healthy workspace.
	probeGap = 150 * time.Millisecond
)

// PIDFile records the supervisor berth started. On Windows a detached
// process-compose cannot be identified through its socket, so the recorded PID
// keeps the state unknown while it is alive; Up removes the file once the
// supervisor answers, after which the endpoint is the witness. On Unix
// process-compose daemonizes itself and berth never learns its PID, which is why
// absence of this file is not evidence either way.
func PIDFile(worktree string) string {
	return filepath.Join(filepath.Dir(Socket(worktree)), "pc.pid")
}

func writePCPID(worktree string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(PIDFile(worktree)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(PIDFile(worktree), []byte(strconv.Itoa(pid)), 0o600)
}

func readPCPID(worktree string) int {
	data, err := os.ReadFile(PIDFile(worktree))
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

// supervisorGone reports whether the control files provably outlived their
// supervisor: it performs the check the unknown-state error asks a human to
// perform, with two witnesses a human does not have, that no supervisor process
// berth started is still alive and that nothing answers on the control endpoint.
// It is deliberately hard to satisfy: a live recorded process, an endpoint that
// accepts connections, and an endpoint that cannot even be addressed all keep
// the process state unknown.
func supervisorGone(ctx context.Context, worktree string) bool {
	if pid := readPCPID(worktree); pid > 0 && processAlive(pid) {
		return false
	}
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(probeGap):
			}
		}
		err := probeControl(ctx, worktree)
		if err == nil || undecided(err) {
			// Something is listening. A supervisor that accepts connections but
			// does not answer is exactly the state that must stay unknown.
			return false
		}
	}
	return ctx.Err() == nil
}

// probeControl connects to the supervisor endpoint without sending a request:
// a usable endpoint that refuses connections proves no supervisor serves this
// workspace.
func probeControl(ctx context.Context, worktree string) error {
	network, address := controlEndpoint(worktree)
	if address == "" {
		return errNoEndpoint
	}
	child, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(child, network, address)
	if err != nil {
		return err
	}
	return conn.Close()
}

// controlEndpoint names the address the supervisor was recorded on, or nothing
// when the recorded files name no address at all.
func controlEndpoint(worktree string) (string, string) {
	if runtime.GOOS == "windows" {
		port := readPCPort(worktree)
		if port < 1 || port > 65535 {
			return "", ""
		}
		return "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	socket := Socket(worktree)
	st, err := os.Lstat(socket)
	if err != nil || st.Mode()&os.ModeSocket == 0 {
		return "", ""
	}
	return "unix", socket
}

// errNoEndpoint means the control files name no usable endpoint, so the probe
// proves nothing about the supervisor. A torn or foreign control file is not
// evidence that a supervisor exited.
var errNoEndpoint = errors.New("no usable control endpoint recorded")

// undecided reports probe outcomes that do not prove the endpoint is dead: an
// address that cannot be derived, or a probe that never completed. Both keep the
// process state unknown, which is the conservative answer.
func undecided(err error) bool {
	var nerr net.Error
	return errors.Is(err, errNoEndpoint) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) || (errors.As(err, &nerr) && nerr.Timeout())
}

// reapStaleSupervisor releases the wake-up artifacts of a supervisor that is
// gone. The API token is kept, exactly as Down keeps it: it is a credential, not
// a liveness witness. Generated configuration, data and the checkout are
// untouched.
func reapStaleSupervisor(worktree string) error {
	for _, path := range []string{Socket(worktree), PortFile(worktree), PIDFile(worktree)} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}

func warnReclaimedSupervisor(worktree, endpoint string) {
	fmt.Fprintf(os.Stderr, "berth: warning: the supervisor for %s exited on its own; reclaimed its stale control files (%s no longer answers) and the workspace counts as stopped\n", worktree, endpoint)
}
