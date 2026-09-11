package netns

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

func writeListenPort(c net.Conn, port int) error {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(port))
	_, err := c.Write(b[:])
	return err
}

func readListenPort(c net.Conn) (int, error) {
	var b [2]byte
	if _, err := io.ReadFull(c, b[:]); err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint16(b[:])), nil
}

func pipe(a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(b, a)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(a, b)
		done <- struct{}{}
	}()
	<-done
}

func serve(ctx context.Context, ln net.Listener, dial func() (net.Conn, error)) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			d, err := dial()
			if err != nil {
				_ = c.Close()
				return
			}
			pipe(c, d)
		}(c)
	}
}

func listenUnix(path string) (net.Listener, error) {
	_ = os.Remove(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", path, err)
	}
	return ln, nil
}

func listenHost(port int) (net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen 127.0.0.1:%d: %w", port, err)
	}
	return ln, nil
}

func dialTCP(addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		var last error
		for i := 0; i < 40; i++ {
			c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err == nil {
				return c, nil
			}
			last = err
			time.Sleep(25 * time.Millisecond)
		}
		return nil, last
	}
}

func dialUnix(path string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		var last error
		for i := 0; i < 40; i++ {
			c, err := net.DialTimeout("unix", path, 200*time.Millisecond)
			if err == nil {
				return c, nil
			}
			last = err
			time.Sleep(25 * time.Millisecond)
		}
		return nil, last
	}
}
