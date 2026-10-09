package hotplug

import (
	"fmt"
	"strings"
)

// interfaceLinkMatches reports whether a Windows device interface symbolic link
// (e.g. `\\?\HID#VID_1038&PID_1612&MI_01#...`) belongs to the vendor.
func interfaceLinkMatches(link string, vendorID uint16) bool {
	return strings.Contains(strings.ToUpper(link), fmt.Sprintf("VID_%04X", vendorID))
}

// ueventMatches reports whether a Linux kernel uevent message announces a
// hidraw node of the vendor being added or removed. The message is
// "ACTION@DEVPATH" followed by NUL-separated KEY=VALUE fields; a hidraw
// DEVPATH contains the parent HID device as "BUS:VID:PID.N", e.g.
// ".../0003:1038:1612.0005/hidraw/hidraw3".
func ueventMatches(msg []byte, vendorID uint16) bool {
	var action, subsystem, devpath string
	for _, field := range strings.Split(string(msg), "\x00") {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "ACTION":
			action = value
		case "SUBSYSTEM":
			subsystem = value
		case "DEVPATH":
			devpath = value
		}
	}
	if subsystem != "hidraw" || (action != "add" && action != "remove") {
		return false
	}
	return strings.Contains(strings.ToUpper(devpath), fmt.Sprintf(":%04X:", vendorID))
}
