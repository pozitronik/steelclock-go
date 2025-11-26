package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pozitronik/steelclock-go/internal/app"
)

var logFile *os.File

func main() {
	configPathFlag := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	setupLogging()
	defer closeLogging()

	application := app.NewApp(*configPathFlag)
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
