//go:build linux

package hotplug

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// readTimeout bounds each receive so the reader notices Stop.
const readTimeout = 500 * time.Millisecond

// startSource listens to kernel uevents on a NETLINK_KOBJECT_UEVENT socket,
// which needs no privileges, and signals on hidraw nodes of the vendor.
func startSource(vendorID uint16, signal func()) (func(), error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return nil, fmt.Errorf("uevent socket: %w", err)
	}
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: 1}); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("uevent bind: %w", err)
	}
	tv := unix.NsecToTimeval(readTimeout.Nanoseconds())
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("uevent socket timeout: %w", err)
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 16*1024)
		for {
			select {
			case <-stop:
				return
			default:
			}
			n, _, err := unix.Recvfrom(fd, buf, 0)
			if err != nil {
				if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) || errors.Is(err, unix.ENOBUFS) {
					continue // timeout, interrupted, or events dropped under load
				}
				return
			}
			if ueventMatches(buf[:n], vendorID) {
				signal()
			}
		}
	}()

	return func() {
		close(stop)
		<-done
		_ = unix.Close(fd)
	}, nil
}
