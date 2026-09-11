//go:build linux

package netns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"golang.org/x/sys/unix"
)

func init() {
	switch os.Getenv(roleEnv) {
	case "supervise":
		os.Exit(runSupervise())
	case "inside":
		os.Exit(runInside())
	case "probe":
		if err := loUp(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

var (
	availOnce sync.Once
	availOK   bool
)

func cloneUserNet() *syscall.SysProcAttr {
	// Map the host user to uid 0 in the namespace so CAP_NET_ADMIN survives
	// exec. Ambient capabilities are dropped on many runners (GitHub Actions)
	// after exec of the Go binary, which leaves SIOCSIFFLAGS lo EPERM.
	return &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      os.Getuid(),
			Size:        1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		}},
		GidMappingsEnableSetgroups: false,
	}
}

func Available() bool {
	availOnce.Do(func() {
		self, err := os.Executable()
		if err != nil {
			return
		}
		cmd := exec.Command(self)
		cmd.Env = append(os.Environ(), roleEnv+"=probe")
		cmd.SysProcAttr = cloneUserNet()
		availOK = cmd.Run() == nil
	})
	return availOK
}

func Start(ctx context.Context, worktree string, maps []Mapping, isolate bool, bin string, args []string, env []string) error {
	if !isolate {
		return nil
	}
	if !Available() {
		return fmt.Errorf("cannot create a user+net namespace to remap listen ports. Enable unprivileged user namespaces (sysctl kernel.unprivileged_userns_clone=1) or pass the process --port $LANE_PORT_*")
	}
	if err := os.MkdirAll(fwdDir(worktree), 0o755); err != nil {
		return err
	}
	if err := writeSpec(worktree, spec{
		Worktree: worktree,
		Maps:     maps,
		Bin:      bin,
		Args:     args,
		Env:      env,
	}); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve lane binary: %w", err)
	}
	if err := os.MkdirAll(config.LaneDir(worktree), 0o755); err != nil {
		return err
	}
	logf, err := os.OpenFile(isolateLogPath(worktree), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logf.Close()

	cmd := exec.Command(self)
	cmd.Dir = worktree
	cmd.Env = append(os.Environ(), roleEnv+"=supervise", specEnv+"="+specPath(worktree))
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start port remap supervisor: %w", err)
	}
	_ = cmd.Process.Release()
	if err := waitStarted(ctx, worktree, maps); err != nil {
		_ = Stop(worktree)
		return err
	}
	return nil
}

func Stop(worktree string) error {
	sup := supervisePID(worktree)
	in := InsidePID(worktree)
	if sup > 0 {
		_ = syscall.Kill(-sup, syscall.SIGTERM)
		_ = syscall.Kill(sup, syscall.SIGTERM)
	}
	if in > 0 && in != sup {
		_ = syscall.Kill(in, syscall.SIGTERM)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if (sup <= 0 || !alive(sup)) && (in <= 0 || !alive(in)) {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if alive(sup) {
		_ = syscall.Kill(-sup, syscall.SIGKILL)
		_ = syscall.Kill(sup, syscall.SIGKILL)
	}
	if alive(in) {
		_ = syscall.Kill(in, syscall.SIGKILL)
	}
	cleanup(worktree)
	return nil
}

var (
	ErrNotRunning = fmt.Errorf("workspace network namespace is not running. Start it with lane up")
	ErrNoNsenter  = fmt.Errorf("nsenter not found. Install util-linux to run commands on the project's localhost ports, or call 127.0.0.1:$LANE_PORT_* from the host")
)

func Exec(ctx context.Context, worktree string, argv []string, env []string) error {
	pid := InsidePID(worktree)
	if pid <= 0 || !alive(pid) {
		return ErrNotRunning
	}
	bin, err := exec.LookPath("nsenter")
	if err != nil {
		return ErrNoNsenter
	}
	args := []string{"--target", strconv.Itoa(pid), "--user", "--net", "--preserve-credentials", "--"}
	args = append(args, argv...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitStarted(ctx context.Context, worktree string, maps []Mapping) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if isolateReady(worktree, maps) {
			return nil
		}
		time.Sleep(40 * time.Millisecond)
	}
	detail := ""
	if data, err := os.ReadFile(isolateLogPath(worktree)); err == nil && len(data) > 0 {
		detail = "\n" + string(data)
	}
	return fmt.Errorf("port remap supervisor did not publish host ports.%s\nInspect %s or run lane doctor", detail, isolateLogPath(worktree))
}

func isolateReady(worktree string, maps []Mapping) bool {
	if !alive(supervisePID(worktree)) || !alive(InsidePID(worktree)) {
		return false
	}
	if _, err := os.Stat(connectSock(worktree)); err != nil {
		return false
	}
	if _, err := os.Stat(controlSock(worktree)); err != nil {
		return false
	}
	return hostPublished(maps)
}

func hostPublished(maps []Mapping) bool {
	for _, m := range maps {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", m.Host), 100*time.Millisecond)
		if err != nil {
			return false
		}
		_ = c.Close()
	}
	return true
}

func Publish(worktree string, m Mapping) error {
	c, err := net.DialTimeout("unix", controlSock(worktree), time.Second)
	if err != nil {
		return fmt.Errorf("publish %s: %w. Is the workspace isolated? Run lane up", m.Name, err)
	}
	defer c.Close()
	if err := json.NewEncoder(c).Encode(m); err != nil {
		return err
	}
	var ack string
	if err := json.NewDecoder(c).Decode(&ack); err != nil {
		return fmt.Errorf("publish %s ack: %w", m.Name, err)
	}
	if ack != "ok" {
		return fmt.Errorf("publish %s: %s", m.Name, ack)
	}
	return nil
}

func cleanup(worktree string) {
	_ = os.Remove(supervisePIDPath(worktree))
	_ = os.Remove(insidePIDPath(worktree))
	_ = os.RemoveAll(fwdDir(worktree))
}

func runSupervise() int {
	s, err := loadSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: load spec: %v\n", err)
		return 1
	}
	if err := writePID(supervisePIDPath(s.Worktree), os.Getpid()); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: write supervisor pid: %v\n", err)
		return 1
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: %v\n", err)
		return 1
	}
	inside := exec.Command(self)
	inside.Dir = s.Worktree
	inside.Env = append(os.Environ(), roleEnv+"=inside", specEnv+"="+os.Getenv(specEnv))
	inside.Stdout = os.Stdout
	inside.Stderr = os.Stderr
	inside.SysProcAttr = cloneUserNet()
	if err := inside.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: spawn isolated process: %v. Enable unprivileged user namespaces (sysctl kernel.unprivileged_userns_clone=1)\n", err)
		return 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := serveControl(ctx, s.Worktree); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: control: %v\n", err)
		_ = inside.Process.Kill()
		return 1
	}
	errCh := make(chan error, 1)
	go func() { errCh <- startHostProxies(ctx, s) }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		_ = inside.Process.Signal(syscall.SIGTERM)
	}()

	waitErr := inside.Wait()
	cancel()
	select {
	case proxErr := <-errCh:
		if proxErr != nil && waitErr == nil {
			fmt.Fprintf(os.Stderr, "lane netns: host publish: %v\n", proxErr)
			return 1
		}
	default:
	}
	if waitErr != nil {
		fmt.Fprintf(os.Stderr, "lane netns: isolated process: %v\n", waitErr)
		return 1
	}
	return 0
}

func runInside() int {
	s, err := loadSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: load spec: %v\n", err)
		return 1
	}
	if err := loUp(); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: bring up lo: %v\n", err)
		return 1
	}
	if err := writePID(insidePIDPath(s.Worktree), os.Getpid()); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: write netns pid: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := startInsideProxies(ctx, s); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: inside publish: %v\n", err)
		return 1
	}
	cmd := exec.Command(s.Bin, s.Args...)
	cmd.Dir = s.Worktree
	cmd.Env = s.Env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lane netns: %v\n", err)
		return 1
	}
	return 0
}

func startHostProxies(ctx context.Context, s spec) error {
	for _, m := range s.Maps {
		if err := startHostProxy(ctx, s.Worktree, m); err != nil {
			return err
		}
	}
	<-ctx.Done()
	return nil
}

func startHostProxy(ctx context.Context, worktree string, m Mapping) error {
	ln, err := listenHost(m.Host)
	if err != nil {
		return err
	}
	go serve(ctx, ln, func() (net.Conn, error) {
		c, err := dialUnix(connectSock(worktree))()
		if err != nil {
			return nil, err
		}
		if err := writeListenPort(c, m.Listen); err != nil {
			_ = c.Close()
			return nil, err
		}
		return c, nil
	})
	return nil
}

func serveControl(ctx context.Context, worktree string) error {
	ln, err := listenUnix(controlSock(worktree))
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handleControl(ctx, worktree, c)
		}
	}()
	return nil
}

func handleControl(ctx context.Context, worktree string, c net.Conn) {
	defer c.Close()
	var m Mapping
	if err := json.NewDecoder(c).Decode(&m); err != nil {
		_ = json.NewEncoder(c).Encode(err.Error())
		return
	}
	if m.Listen <= 0 || m.Host <= 0 {
		_ = json.NewEncoder(c).Encode("host and listen ports required")
		return
	}
	if err := startHostProxy(ctx, worktree, m); err != nil {
		_ = json.NewEncoder(c).Encode(err.Error())
		return
	}
	_ = json.NewEncoder(c).Encode("ok")
}

func startInsideProxies(ctx context.Context, s spec) error {
	ln, err := listenUnix(connectSock(s.Worktree))
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				port, err := readListenPort(c)
				if err != nil {
					_ = c.Close()
					return
				}
				d, err := dialTCP(fmt.Sprintf("127.0.0.1:%d", port))()
				if err != nil {
					_ = c.Close()
					return
				}
				pipe(c, d)
			}(c)
		}
	}()
	return nil
}

func loUp() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	ifr, err := unix.NewIfreq("lo")
	if err != nil {
		return err
	}
	if err := unix.IoctlIfreq(fd, unix.SIOCGIFFLAGS, ifr); err != nil {
		return fmt.Errorf("SIOCGIFFLAGS lo: %w", err)
	}
	ifr.SetUint16(ifr.Uint16() | unix.IFF_UP | unix.IFF_RUNNING)
	if err := unix.IoctlIfreq(fd, unix.SIOCSIFFLAGS, ifr); err != nil {
		return fmt.Errorf("SIOCSIFFLAGS lo: %w", err)
	}
	return nil
}
