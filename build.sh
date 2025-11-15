#!/bin/bash
# Build script for SteelClock

set -e

echo "Building SteelClock for Windows..."

# Cross-compile for Windows from Linux/WSL
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o steelclock.exe ./cmd/steelclock

echo "Build complete: steelclock.exe"
ls -lh steelclock.exe
file steelclock.exe
