package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pozitronik/steelclock-go/internal/app"
	"github.com/pozitronik/steelclock-go/internal/config"
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
	flag.Parse()

	setupLogging()
	defer closeLogging()

	// Get current working directory for config search
	baseDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// If CWD doesn't look like the app directory, fall back to executable's directory.
	// This handles autostart and shortcut scenarios where CWD differs from app location.
	if !looksLikeAppDir(baseDir) {
		if exePath, exeErr := os.Executable(); exeErr == nil {
			if altDir := filepath.Dir(exePath); looksLikeAppDir(altDir) {
				log.Printf("CWD %q has no config; using executable directory %q instead", baseDir, altDir)
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
