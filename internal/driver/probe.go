package driver

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Probe timing and behavior constants.
const (
	// probePollInterval is how often the spacebar is sampled while a command byte
	// is being shown.
	probePollInterval = 40 * time.Millisecond
	// probeBlank is the brief gap between attempts so each reads as a distinct flash.
	probeBlank = 300 * time.Millisecond
	// probeConfirmDwell is how long each candidate is held during the confirmation
	// pass (longer than the sweep dwell, since only a few candidates are tested).
	probeConfirmDwell = 2500 * time.Millisecond
	// probeConfirmBefore/After bound the numeric neighborhood retried during
	// confirmation, absorbing the reaction-time lag of the initial mark (the real
	// command is at or slightly before the marked one as the sweep advances).
	probeConfirmBefore = 4
	probeConfirmAfter  = 1
)

// probeCuratedCommands are command bytes already seen on SteelSeries OLED devices
// (0x93 GameDAC/old Nova, 0x61 Apex keyboards, 0x4a/0x0a newer wireless units).
// They are tried first so a match is usually found within seconds.
var probeCuratedCommands = []byte{0x93, 0x61, 0x4a, 0x0a}

// hexList formats a byte slice as space-separated 0xNN values for logs/reports.
func hexList(bs []byte) string {
	if len(bs) == 0 {
		return "(none)"
	}
	parts := make([]string, len(bs))
	for i, b := range bs {
		parts[i] = fmt.Sprintf("0x%02X", b)
	}
	return strings.Join(parts, " ")
}

// findProbeTarget picks the SteelSeries collection with the largest HID feature
// report — the OLED screen interface — returning its path and report length.
func findProbeTarget() (path string, featureLen int, err error) {
	devices, derr := EnumerateDevices()
	if derr != nil {
		return "", 0, fmt.Errorf("enumerate devices: %w", derr)
	}
	for _, d := range devices {
		if d.VID == SteelSeriesVID && d.HasCaps && d.FeatureReportLen > featureLen {
			featureLen = d.FeatureReportLen
			path = d.Path
		}
	}
	if path == "" || featureLen < 2 {
		return "", 0, fmt.Errorf("no SteelSeries interface with a usable feature report found (HID capabilities unavailable on this platform?)")
	}
	return path, featureLen, nil
}

// commandSweepOrder lists every command byte 0x00–0xFF with the curated likely
// values first, so the probe tends to hit early.
func commandSweepOrder() []int {
	seen := make(map[int]bool, 256)
	order := make([]int, 0, 256)
	for _, c := range probeCuratedCommands {
		if !seen[int(c)] {
			seen[int(c)] = true
			order = append(order, int(c))
		}
	}
	for c := 0; c <= 0xFF; c++ {
		if !seen[c] {
			order = append(order, c)
		}
	}
	return order
}

// probeWaitForMark shows the current attempt for d, polling the spacebar; it
// returns true as soon as the user presses space (the screen lit up).
func probeWaitForMark(d time.Duration) bool {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if probeKeyDown() {
			return true
		}
		time.Sleep(probePollInterval)
	}
	return false
}

// probeDrainKey waits (briefly) for the spacebar to be released, so a held press
// from the previous step does not immediately re-trigger the next one.
func probeDrainKey() {
	end := time.Now().Add(2 * time.Second)
	for time.Now().Before(end) {
		if !probeKeyDown() {
			return
		}
		time.Sleep(probePollInterval)
	}
}

// sendAllOn writes an all-pixels-on frame for a given report ID and command byte.
func sendAllOn(handle DeviceHandle, featureLen int, reportID, command byte) {
	frame := make([]byte, featureLen)
	for i := range frame {
		frame[i] = 0xFF
	}
	frame[0] = reportID
	frame[1] = command
	_ = sendFeatureReport(handle, frame)
}

// sendBlank writes an all-zero (blank) frame for a given report ID.
func sendBlank(handle DeviceHandle, featureLen int, reportID byte) {
	blank := make([]byte, featureLen)
	blank[0] = reportID
	_ = sendFeatureReport(handle, blank)
}

// confirmCommand re-tests the numeric neighborhood of a marked command byte one
// at a time, holding each longer, so the user can pinpoint the exact command
// even if their initial mark lagged by a step. Returns the confirmed byte and
// whether the user confirmed one.
func confirmCommand(handle DeviceHandle, featureLen int, reportID byte, marked int) (byte, bool) {
	lo := marked - probeConfirmBefore
	if lo < 0 {
		lo = 0
	}
	hi := marked + probeConfirmAfter
	if hi > 0xFF {
		hi = 0xFF
	}
	log.Printf("OLED probe: confirming around command 0x%02X (testing 0x%02X..0x%02X)", marked, lo, hi)
	for cmd := lo; cmd <= hi; cmd++ {
		sendBlank(handle, featureLen, reportID)
		time.Sleep(probeBlank)
		sendAllOn(handle, featureLen, reportID, byte(cmd))
		log.Printf("OLED probe: confirm command=0x%02X — press space if the screen is lit", cmd)
		if probeWaitForMark(probeConfirmDwell) {
			return byte(cmd), true
		}
		probeDrainKey()
	}
	return byte(marked), false
}

// RunOLEDProbe reverse-engineers an undocumented SteelSeries OLED interactively,
// using the user's eye + spacebar as the "did it light up?" sensor (no USB
// capture or video needed). It:
//  1. finds the screen interface (largest feature report) and opens it;
//  2. discovers which HID feature report IDs the interface accepts at that length
//     (a valid report ID + matching size is accepted; others error) — avoiding
//     fragile report-descriptor parsing;
//  3. for the accepted report ID, sweeps command bytes (known values first)
//     showing an all-pixels-on frame and waiting for the user to press space when
//     the screen reacts, then runs a short confirmation pass to pinpoint the byte.
//
// SteelSeries GG must be closed so it does not also drive the screen.
func RunOLEDProbe(dwell time.Duration) (string, error) {
	var b strings.Builder

	path, featureLen, err := findProbeTarget()
	if err != nil {
		return "", err
	}
	log.Printf("OLED probe: target %s (feature report %d bytes)", path, featureLen)
	fmt.Fprintf(&b, "Probe target: %s\nFeature report length: %d bytes\n\n", path, featureLen)

	handle, err := openDevice(path)
	if err != nil {
		return "", fmt.Errorf("open device: %w", err)
	}
	defer func() { _ = closeDevice(handle) }()

	// Phase 1 — discover which report IDs accept a full-size feature report.
	// An all-zero report is benign (a blank frame for the screen report).
	var reportIDs []byte
	for rid := 0; rid <= 0xFF; rid++ {
		zero := make([]byte, featureLen)
		zero[0] = byte(rid)
		if sendFeatureReport(handle, zero) == nil {
			reportIDs = append(reportIDs, byte(rid))
		}
	}
	log.Printf("OLED probe: accepted feature report IDs at length %d: %s", featureLen, hexList(reportIDs))
	fmt.Fprintf(&b, "Accepted feature report IDs (length %d): %s\n", featureLen, hexList(reportIDs))
	if len(reportIDs) == 0 {
		fmt.Fprintf(&b, "\nNo report ID accepted a full-size feature report, so the sweep cannot run.\n"+
			"This means the screen report uses a different length than the reported maximum.\n")
		log.Print("OLED probe: no accepted report IDs; aborting sweep")
		return b.String(), nil
	}

	// Phase 2 — interactive command-byte sweep. The user presses space when the
	// OLED shows anything; the app already knows which command byte is active.
	order := commandSweepOrder()
	log.Printf("OLED probe: starting interactive sweep (%v per step). Press SPACE when the screen lights up.", dwell)
	fmt.Fprintf(&b, "\nInteractive sweep started (%v per step). Watching for SPACE.\n", dwell)

	for _, rid := range reportIDs {
		markedCmd := -1
		for _, cmd := range order {
			sendAllOn(handle, featureLen, rid, byte(cmd))
			log.Printf("OLED probe: reportID=0x%02X command=0x%02X (all pixels on)", rid, cmd)
			if probeWaitForMark(dwell) {
				markedCmd = cmd
				log.Printf("OLED probe: user marked reportID=0x%02X command=0x%02X", rid, cmd)
				break
			}
			sendBlank(handle, featureLen, rid)
			time.Sleep(probeBlank)
		}

		if markedCmd < 0 {
			continue // nothing marked for this report ID; try the next one
		}

		probeDrainKey()
		cmd, confirmed := confirmCommand(handle, featureLen, rid, markedCmd)
		sendBlank(handle, featureLen, rid)
		if confirmed {
			log.Printf("OLED probe: CONFIRMED reportID=0x%02X command=0x%02X", rid, cmd)
			fmt.Fprintf(&b, "\nFOUND: report ID 0x%02X, command byte 0x%02X.\nThe screen lit up for this combination.\n", rid, cmd)
		} else {
			log.Printf("OLED probe: marked reportID=0x%02X near command=0x%02X (unconfirmed)", rid, markedCmd)
			fmt.Fprintf(&b, "\nMarked report ID 0x%02X around command 0x%02X, but the confirmation pass was inconclusive.\n"+
				"Re-run and press space a touch earlier, or note what you saw.\n", rid, markedCmd)
		}
		return b.String(), nil
	}

	log.Print("OLED probe: sweep complete with no mark")
	b.WriteString("\nSweep finished and nothing was marked — the screen never lit up.\n" +
		"That suggests the frame needs a different framing (e.g. a header) or report ID. Tell me and I'll adjust.\n")
	return b.String(), nil
}
