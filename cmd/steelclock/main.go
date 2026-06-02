package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pozitronik/steelclock-go/internal/app"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/dialog"
	"github.com/pozitronik/steelclock-go/internal/driver"
)

// looksLikeAppDir returns true if dir contains steelclock.json or a profiles/ subdirectory,
// suggesting it is the application's working directory.
func looksLikeAppDir(dir string) bool {
	if info, err := os.Stat(filepath.Join(dir, "steelclock.json")); err == nil && !info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(dir, "profiles")); err == nil && info.IsDir() {
		return true
	}
	return false
}

var logFile *os.File

func main() {
	configPathFlag := flag.String("config", "", "Path to configuration file (overrides profile system)")
	listDevicesFlag := flag.Bool("list-devices", false, "Enumerate connected SteelSeries HID devices, write a report, and exit (diagnostic)")
	probeOLEDFlag := flag.Bool("probe-oled", false, "Sweep OLED command bytes on the SteelSeries screen interface so a person filming the screen can identify the working command, then exit (diagnostic)")
	flag.Parse()

	setupLogging()
	defer closeLogging()

	// Diagnostic: enumerate HID devices and exit. Does not start the app.
	if *listDevicesFlag {
		runDeviceDiagnostic()
		return
	}

	// Diagnostic: sweep OLED command bytes and exit. Does not start the app.
	if *probeOLEDFlag {
		runOLEDProbe()
		return
	}

	// Get current working directory for config search
	baseDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// If CWD doesn't look like the app directory, fall back to executable's directory.
	// This handles autostart and shortcut scenarios where CWD differs from app location.
	// We os.Chdir so that all relative paths (fonts, profiles, etc.) resolve correctly.
	if !looksLikeAppDir(baseDir) {
		if exePath, exeErr := os.Executable(); exeErr == nil {
			if altDir := filepath.Dir(exePath); looksLikeAppDir(altDir) {
				log.Printf("CWD %q has no config; changing working directory to %q", baseDir, altDir)
				if err := os.Chdir(altDir); err != nil {
					log.Printf("Warning: failed to chdir to %q: %v", altDir, err)
				}
				baseDir = altDir
			}
		}
	}

	// If explicit config path is provided, use legacy single-config mode
	if *configPathFlag != "" {
		application := app.NewApp(*configPathFlag)
		application.Run()
		return
	}

	// Use profile manager for multi-config mode
	profileMgr := config.NewProfileManager(baseDir)
	if err := profileMgr.LoadProfiles(); err != nil {
		log.Printf("Warning: Failed to load profiles: %v", err)
	}

	application := app.NewAppWithProfiles(profileMgr)
	application.Run()
}

// runDeviceDiagnostic enumerates connected HID devices, logs and saves a report,
// and shows it in a dialog. Used by the -list-devices flag to discover a device's
// USB identifiers and interfaces without requiring the user to inspect them manually.
func runDeviceDiagnostic() {
	var report string
	if devices, err := driver.EnumerateDevices(); err != nil {
		report = fmt.Sprintf("Device enumeration failed: %v", err)
	} else {
		report = driver.FormatDeviceDiagnostic(devices)
	}

	log.Print(report)

	// Save next to the executable so it is easy to find and attach.
	reportPath := "steelclock-devices.txt"
	if exePath, err := os.Executable(); err == nil {
		reportPath = filepath.Join(filepath.Dir(exePath), "steelclock-devices.txt")
	}
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		log.Printf("Failed to write device report: %v", err)
	} else {
		report += fmt.Sprintf("\nSaved to:\n%s", reportPath)
	}

	dialog.ShowMessage("SteelClock device diagnostic", report, false)
}

// oledProbeDwell is how long each command byte is held during the sweep — long
// enough to be visible to someone filming the OLED, short enough that the full
// 256-byte sweep stays a few minutes.
const oledProbeDwell = 1200 * time.Millisecond

// runOLEDProbe sweeps OLED command bytes so a person filming the screen can spot
// which command drives an undocumented SteelSeries display. Used by -probe-oled.
func runOLEDProbe() {
	dialog.ShowMessage("SteelClock OLED probe",
		"This figures out how your headset OLED is driven by trying screen commands one by one.\n\n"+
			"1. Make sure SteelSeries GG is fully closed.\n"+
			"2. Watch the headset OLED screen.\n"+
			"3. Click OK to begin.\n\n"+
			"The moment ANYTHING appears on the OLED, press the SPACEBAR. It will then re-check a\n"+
			"few nearby commands to pin it down — press SPACE again on the one that lights the screen.\n"+
			"Known commands are tried first, so it usually only takes a few seconds.\n\n"+
			"When the result popup appears, send me steelclock.log.",
		false)

	report, err := driver.RunOLEDProbe(oledProbeDwell)
	if err != nil {
		report = fmt.Sprintf("OLED probe failed: %v", err)
	}
	log.Print(report)

	dialog.ShowMessage("SteelClock OLED probe — done", report, false)
}

// setupLogging configures logging to file
func setupLogging() {
	exePath, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to get executable path: %v\n", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	logFileName := filepath.Join(exeDir, "steelclock.log")

	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to open log file: %v\n", err)
		return
	}

	multiWriter := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// closeLogging closes the log file
func closeLogging() {
	if logFile != nil {
		_ = logFile.Close()
	}
}
