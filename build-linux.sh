#!/bin/bash
# Build script for SteelClock on Linux
# Can be run natively on Linux or for cross-compilation

set -e

echo "======================================"
echo "Building SteelClock for Linux"
echo "======================================"
echo ""

# Determine if we're cross-compiling
NATIVE_BUILD=false
if [[ "$(uname -s)" == "Linux" ]]; then
    NATIVE_BUILD=true
fi

# Step 1: Cleanup old build
echo "[1/4] Cleaning old build..."
rm -f steelclock
rm -f internal/tray/icon.ico
echo "OK Cleanup complete"
echo ""

# Step 2: Copy tray icon (optional, for consistency)
echo "[2/4] Preparing tray icon..."
if [ -f "winres/icon.ico" ]; then
    mkdir -p internal/tray
    cp winres/icon.ico internal/tray/icon.ico
    echo "OK Copied icon.ico to internal/tray/"
else
    echo "!! Warning: winres/icon.ico not found"
    echo "   Tray icon will use default"
fi
echo ""

# Step 3: Check dependencies (native build only)
echo "[3/4] Checking dependencies..."
if [ "$NATIVE_BUILD" = true ]; then
    # Check for required GTK libraries
    MISSING_DEPS=false

    if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
        echo "!! Warning: GTK+3 not found"
        echo "   Install with: sudo apt-get install libgtk-3-dev"
        MISSING_DEPS=true
    fi

    if ! pkg-config --exists ayatana-appindicator3-0.1 2>/dev/null; then
        echo "!! Warning: libayatana-appindicator3 not found"
        echo "   Install with: sudo apt-get install libayatana-appindicator3-dev"
        MISSING_DEPS=true
    fi

    if [ "$MISSING_DEPS" = true ]; then
        echo ""
        echo "Some dependencies are missing. The build may fail."
        echo "Install all dependencies with:"
        echo "  sudo apt-get install libgtk-3-dev libayatana-appindicator3-dev"
        echo ""
    else
        echo "OK All dependencies found"
    fi
else
    echo "-- Cross-compiling, skipping dependency check"
fi
echo ""

# Step 4: Build executable
echo "[4/4] Compiling executable..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o steelclock ./cmd/steelclock
echo "OK Compilation successful"
echo ""

# Summary
echo "======================================"
echo "Build Summary"
echo "======================================"
ls -lh steelclock
file steelclock

echo ""
echo "OK Build complete!"
echo ""
echo "Usage:"
echo "  ./steelclock                    # Run (requires udev rules for direct driver)"
echo "  ./steelclock -config config.json"
echo ""
echo "For direct USB driver access, install udev rules:"
echo "  sudo cp profiles/99-steelseries.rules /etc/udev/rules.d/"
echo "  sudo udevadm control --reload-rules && sudo udevadm trigger"
echo ""
echo "Logs: steelclock.log in the same directory as the executable"
