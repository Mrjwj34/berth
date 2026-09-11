//go:build linux

package remap

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/Mrjwj34/lane/internal/ports"
	"golang.org/x/sys/unix"
)

type seccompData struct {
	Nr                 int32
	Arch               uint32
	InstructionPointer uint64
	Args               [6]uint64
}

type seccompNotif struct {
	ID    uint64
	Pid   uint32
	Flags uint32
	Data  seccompData
}

type seccompNotifResp struct {
	ID    uint64
	Val   int64
	Error int32
	Flags uint32
}

func recvListener(comm int) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))
	n, oobn, _, _, err := unix.Recvmsg(comm, buf, oob, 0)
	if err != nil {
		return -1, err
	}
	if n == 1 && buf[0] == 'x' {
		return -1, fmt.Errorf("child could not install seccomp")
	}
	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, err
	}
	for _, msg := range msgs {
		fds, err := unix.ParseUnixRights(&msg)
		if err != nil || len(fds) == 0 {
			continue
		}
		return fds[0], nil
	}
	return -1, fmt.Errorf("no seccomp listener fd from remap wrapper")
}

func handleLoop(listener int, tablePath string) {
	var req seccompNotif
	var resp seccompNotifResp
	for {
		req = seccompNotif{}
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(listener), unix.SECCOMP_IOCTL_NOTIF_RECV, uintptr(unsafe.Pointer(&req)))
		if errno == unix.EINTR || errno == unix.ENOENT {
			continue
		}
		if errno != 0 {
			return
		}
		resp = seccompNotifResp{ID: req.ID}
		handleNotif(tablePath, &req, &resp)
		_, _, sendErr := unix.Syscall(unix.SYS_IOCTL, uintptr(listener), unix.SECCOMP_IOCTL_NOTIF_SEND, uintptr(unsafe.Pointer(&resp)))
		if sendErr != 0 && sendErr != unix.ENOENT {
			return
		}
	}
}

func handleNotif(tablePath string, req *seccompNotif, resp *seccompNotifResp) {
	switch req.Data.Nr {
	case int32(unix.SYS_BIND):
		fail(resp, doBind(tablePath, req))
	case int32(unix.SYS_CONNECT):
		fail(resp, doConnect(tablePath, req))
	case int32(unix.SYS_GETSOCKNAME):
		fail(resp, doName(tablePath, req, false))
	case int32(unix.SYS_GETPEERNAME):
		fail(resp, doName(tablePath, req, true))
	default:
		resp.Error = -int32(unix.ENOSYS)
	}
}

func fail(resp *seccompNotifResp, err error) {
	if err == nil {
		resp.Error = 0
		resp.Val = 0
		return
	}
	if errno, ok := err.(unix.Errno); ok {
		resp.Error = -int32(errno)
		resp.Val = 0
		return
	}
	if errno, ok := err.(syscall.Errno); ok {
		resp.Error = -int32(errno)
		resp.Val = 0
		return
	}
	resp.Error = -int32(unix.EIO)
}

func doBind(tablePath string, req *seccompNotif) error {
	fd, addr, err := stealSock(req)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	if sa, ok := rewriteBind(tablePath, addr); ok {
		return unix.Bind(fd, sa)
	}
	sa, err := toSockaddr(addr)
	if err != nil {
		return unix.EINVAL
	}
	return unix.Bind(fd, sa)
}

func doConnect(tablePath string, req *seccompNotif) error {
	fd, addr, err := stealSock(req)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	if sa, ok := rewriteConnect(tablePath, addr); ok {
		return unix.Connect(fd, sa)
	}
	sa, err := toSockaddr(addr)
	if err != nil {
		return unix.EINVAL
	}
	return unix.Connect(fd, sa)
}

func doName(tablePath string, req *seccompNotif, peer bool) error {
	fd, err := stealFD(int(req.Pid), int(req.Data.Args[0]))
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	var sa unix.Sockaddr
	if peer {
		sa, err = unix.Getpeername(fd)
	} else {
		sa, err = unix.Getsockname(fd)
	}
	if err != nil {
		return err
	}
	raw, err := fromSockaddr(sa)
	if err != nil {
		return err
	}
	if host := portOf(raw); host > 0 {
		if listen, ok := reverseListen(tablePath, host); ok && listen != host {
			setPort(raw, listen)
		}
	}
	ptr := uintptr(req.Data.Args[1])
	lenPtr := uintptr(req.Data.Args[2])
	if err := processWrite(int(req.Pid), ptr, raw); err != nil {
		return err
	}
	var namelen uint32 = uint32(len(raw))
	return processWrite(int(req.Pid), lenPtr, uint32Bytes(namelen))
}

func stealSock(req *seccompNotif) (int, []byte, error) {
	fd, err := stealFD(int(req.Pid), int(req.Data.Args[0]))
	if err != nil {
		return 0, nil, err
	}
	n := int(req.Data.Args[2])
	if n <= 0 || n > 256 {
		_ = unix.Close(fd)
		return 0, nil, unix.EINVAL
	}
	buf, err := processRead(int(req.Pid), uintptr(req.Data.Args[1]), n)
	if err != nil {
		_ = unix.Close(fd)
		return 0, nil, err
	}
	return fd, buf, nil
}

func stealFD(pid, target int) (int, error) {
	pidfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return 0, err
	}
	defer unix.Close(pidfd)
	return unix.PidfdGetfd(pidfd, target, 0)
}

func rewriteBind(tablePath string, addr []byte) (unix.Sockaddr, bool) {
	fam := familyOf(addr)
	if fam != unix.AF_INET && fam != unix.AF_INET6 {
		return nil, false
	}
	want := portOf(addr)
	if want <= 0 || (want >= ports.Min && want <= ports.Max) {
		return nil, false
	}
	host, err := allocListen(tablePath, want)
	if err != nil || host <= 0 {
		return nil, false
	}
	setPort(addr, host)
	sa, err := toSockaddr(addr)
	if err != nil {
		return nil, false
	}
	return sa, true
}

func rewriteConnect(tablePath string, addr []byte) (unix.Sockaddr, bool) {
	if !addrLoopback(addr) {
		return nil, false
	}
	dest := portOf(addr)
	if dest <= 0 {
		return nil, false
	}
	host, ok := lookupHost(tablePath, dest)
	if !ok || host == dest {
		return nil, false
	}
	setPort(addr, host)
	sa, err := toSockaddr(addr)
	if err != nil {
		return nil, false
	}
	return sa, true
}

func familyOf(b []byte) uint16 {
	if len(b) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(b[:2])
}

func portOf(b []byte) int {
	fam := familyOf(b)
	if (fam != unix.AF_INET && fam != unix.AF_INET6) || len(b) < 4 {
		return -1
	}
	return int(binary.BigEndian.Uint16(b[2:4]))
}

func setPort(b []byte, port int) {
	if len(b) < 4 {
		return
	}
	binary.BigEndian.PutUint16(b[2:4], uint16(port))
}

func addrLoopback(b []byte) bool {
	switch familyOf(b) {
	case unix.AF_INET:
		if len(b) < 8 {
			return false
		}
		return b[4] == 127
	case unix.AF_INET6:
		if len(b) < 24 {
			return false
		}
		zero := true
		for i := 8; i < 23; i++ {
			if b[i] != 0 {
				zero = false
				break
			}
		}
		if zero && b[23] == 1 {
			return true
		}
		v4 := true
		for i := 8; i < 18; i++ {
			if b[i] != 0 {
				v4 = false
				break
			}
		}
		if v4 && b[18] == 0xff && b[19] == 0xff && b[20] == 127 {
			return true
		}
	}
	return false
}

func toSockaddr(b []byte) (unix.Sockaddr, error) {
	switch familyOf(b) {
	case unix.AF_INET:
		if len(b) < 8 {
			return nil, unix.EINVAL
		}
		var a unix.SockaddrInet4
		a.Port = portOf(b)
		copy(a.Addr[:], b[4:8])
		return &a, nil
	case unix.AF_INET6:
		if len(b) < 24 {
			return nil, unix.EINVAL
		}
		var a unix.SockaddrInet6
		a.Port = portOf(b)
		copy(a.Addr[:], b[8:24])
		if len(b) >= 28 {
			a.ZoneId = binary.LittleEndian.Uint32(b[24:28])
		}
		return &a, nil
	case unix.AF_UNIX:
		var a unix.SockaddrUnix
		if len(b) > 2 {
			path := b[2:]
			if i := indexByte(path, 0); i >= 0 {
				path = path[:i]
			}
			a.Name = string(path)
		}
		return &a, nil
	default:
		return nil, unix.EAFNOSUPPORT
	}
}

func fromSockaddr(sa unix.Sockaddr) ([]byte, error) {
	switch a := sa.(type) {
	case *unix.SockaddrInet4:
		b := make([]byte, 16)
		binary.LittleEndian.PutUint16(b[0:2], unix.AF_INET)
		binary.BigEndian.PutUint16(b[2:4], uint16(a.Port))
		copy(b[4:8], a.Addr[:])
		return b[:16], nil
	case *unix.SockaddrInet6:
		b := make([]byte, 28)
		binary.LittleEndian.PutUint16(b[0:2], unix.AF_INET6)
		binary.BigEndian.PutUint16(b[2:4], uint16(a.Port))
		copy(b[8:24], a.Addr[:])
		binary.LittleEndian.PutUint32(b[24:28], a.ZoneId)
		return b, nil
	case *unix.SockaddrUnix:
		b := make([]byte, 2+len(a.Name)+1)
		binary.LittleEndian.PutUint16(b[0:2], unix.AF_UNIX)
		copy(b[2:], a.Name)
		return b, nil
	default:
		return nil, unix.EAFNOSUPPORT
	}
}

func processRead(pid int, addr uintptr, n int) ([]byte, error) {
	buf := make([]byte, n)
	local := []unix.Iovec{{Base: &buf[0], Len: uint64(len(buf))}}
	remote := []unix.RemoteIovec{{Base: addr, Len: n}}
	_, err := unix.ProcessVMReadv(pid, local, remote, 0)
	return buf, err
}

func processWrite(pid int, addr uintptr, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	local := []unix.Iovec{{Base: &data[0], Len: uint64(len(data))}}
	remote := []unix.RemoteIovec{{Base: addr, Len: len(data)}}
	_, err := unix.ProcessVMWritev(pid, local, remote, 0)
	return err
}

func uint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}
