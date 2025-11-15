#!/bin/bash
# Build script for SteelClock

set -e

echo "Building SteelClock for Windows..."
echo ""

# Check if go-winres is available and generate Windows resources if configured
if command -v go-winres &> /dev/null; then
    if [ -f "winres/winres.json" ]; then
        echo "Generating Windows resources with go-winres..."
        if go-winres make --out cmd/steelclock/rsrc 2>&1; then
            echo "✓ Windows resources generated"
            echo ""
        else
            echo "Warning: go-winres failed (missing icon files?), continuing without resources"
            echo ""
        fi
    else
        echo "Note: winres/winres.json not found, skipping resource generation"
        echo ""
    fi
else
    # Try to find go-winres in common locations
    if [ -f "$HOME/go/bin/go-winres" ]; then
        if [ -f "winres/winres.json" ]; then
            echo "Generating Windows resources with go-winres..."
            if "$HOME/go/bin/go-winres" make --out cmd/steelclock/rsrc 2>&1; then
                echo "✓ Windows resources generated"
                echo ""
            else
                echo "Warning: go-winres failed (missing icon files?), continuing without resources"
                echo ""
            fi
        fi
    else
        echo "Note: go-winres not found, skipping resource generation"
        echo "      Install with: go install github.com/tc-hib/go-winres@latest"
        echo ""
    fi
fi

# Cross-compile for Windows from Linux/WSL (GUI mode, no console)
echo "Compiling executable..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o steelclock.exe ./cmd/steelclock

echo ""
echo "✓ Build complete: steelclock.exe (GUI mode)"
ls -lh steelclock.exe
file steelclock.exe

echo ""
echo "Run with -console flag to enable console mode"
echo "Example: steelclock.exe -console"
