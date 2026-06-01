package driver

import (
	"fmt"
	"sort"
	"strings"
)

// lookupKnownDevice returns the KnownDevice entry for a SteelSeries PID, if any.
func lookupKnownDevice(pid uint16) (KnownDevice, bool) {
	for _, k := range KnownDevices {
		if k.VID == SteelSeriesVID && k.PID == pid {
			return k, true
		}
	}
	return KnownDevice{}, false
}

// FormatDeviceDiagnostic produces a human-readable report of the connected HID
// interfaces, highlighting SteelSeries (VID 0x1038) ones and cross-referencing
// their PIDs against the known-device table. It is used by the -list-devices
// diagnostic to discover which device/interfaces a machine exposes (e.g. whether
// a Nova Pro's OLED interface is present and whether its PID is recognized).
func FormatDeviceDiagnostic(devices []DeviceInfo) string {
	var b strings.Builder
	b.WriteString("SteelClock device diagnostic\n")
	b.WriteString("============================\n\n")

	var steel, other []DeviceInfo
	for _, d := range devices {
		if d.VID == SteelSeriesVID {
			steel = append(steel, d)
		} else {
			other = append(other, d)
		}
	}

	if len(steel) == 0 {
		b.WriteString("No SteelSeries (VID 0x1038) HID interfaces found.\n")
		b.WriteString("The device may be powered off, in a different mode, or exposed under another USB interface.\n\n")
	} else {
		fmt.Fprintf(&b, "SteelSeries HID interfaces (VID 0x1038): %d\n", len(steel))

		sort.Slice(steel, func(i, j int) bool {
			if steel[i].PID != steel[j].PID {
				return steel[i].PID < steel[j].PID
			}
			return steel[i].Interface < steel[j].Interface
		})
		for _, d := range steel {
			iface := d.Interface
			if iface == "" {
				iface = "(single)"
			}
			fmt.Fprintf(&b, "  PID 0x%04x  %-9s  %s\n", d.PID, iface, d.Path)
		}
		b.WriteString("\n")

		// Cross-reference each unique PID against the known-device table.
		seen := make(map[uint16]bool)
		for _, d := range steel {
			if seen[d.PID] {
				continue
			}
			seen[d.PID] = true

			if known, ok := lookupKnownDevice(d.PID); ok {
				proto := "Apex (default)"
				if known.NewProtocol != nil {
					proto = known.NewProtocol().DeviceFamily()
				}
				fmt.Fprintf(&b, "  PID 0x%04x is KNOWN: %q (%dx%d, protocol: %s)\n",
					d.PID, known.Name, known.DisplaySize.Width, known.DisplaySize.Height, proto)
			} else {
				fmt.Fprintf(&b, "  PID 0x%04x is NOT in SteelClock's known-device table\n", d.PID)
			}
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "(%d other non-SteelSeries HID interface(s) also present)\n", len(other))
	return b.String()
}
