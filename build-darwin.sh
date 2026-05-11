#!/bin/bash
# Build script for SteelClock on macOS (Apple Silicon)
# Can be run natively on macOS or for cross-compilation
#
# Usage:
#   ./build-darwin.sh         # Full build (all widgets)
#   ./build-darwin.sh --light # Light build (excludes heavy widgets)
#   ./build-darwin.sh -l      # Same as --light

set -e

# Parse arguments
BUILD_VARIANT="full"
BUILD_TAGS=""
OUTPUT_SUFFIX=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --light|-l)
            BUILD_VARIANT="light"
            BUILD_TAGS="-tags light"
            OUTPUT_SUFFIX="-light"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--light|-l]"
            exit 1
            ;;
    esac
done

echo "======================================"
echo "Building SteelClock for macOS ($BUILD_VARIANT)"
echo "======================================"
echo ""

# Determine if we're building natively on macOS
NATIVE_BUILD=false
if [[ "$(uname -s)" == "Darwin" ]]; then
    NATIVE_BUILD=true
fi

# Detect architecture
if [ "$NATIVE_BUILD" = true ]; then
    ARCH=$(uname -m)
    if [ "$ARCH" = "arm64" ]; then
        GOARCH="arm64"
    else
        GOARCH="amd64"
    fi
else
    # Default to arm64 for cross-compilation (Apple Silicon)
    GOARCH="arm64"
fi

# Step 1: Cleanup old build
echo "[1/3] Cleaning old build..."
rm -f steelclock steelclock-light
rm -f internal/tray/icon.ico
echo "OK Cleanup complete"
echo ""

# Step 2: Copy tray icon
echo "[2/3] Preparing tray icon..."
if [ -f "winres/icon.ico" ]; then
    mkdir -p internal/tray
    cp winres/icon.ico internal/tray/icon.ico
    echo "OK Copied icon.ico to internal/tray/"
else
    echo "!! Warning: winres/icon.ico not found"
    echo "   Tray icon will use default"
fi
echo ""

# Step 3: Build executable
echo "[3/3] Compiling executable ($BUILD_VARIANT, GOARCH=$GOARCH)..."
OUTPUT_NAME="steelclock${OUTPUT_SUFFIX}"
GOOS=darwin GOARCH="$GOARCH" go build $BUILD_TAGS -ldflags="-s -w" -o "$OUTPUT_NAME" ./cmd/steelclock
echo "OK Compilation successful"
echo ""

# Summary
echo "======================================"
echo "Build Summary ($BUILD_VARIANT)"
echo "======================================"
ls -lh "$OUTPUT_NAME"
file "$OUTPUT_NAME"

echo ""
echo "OK Build complete!"
echo ""
echo "Usage:"
echo "  ./$OUTPUT_NAME                    # Run"
echo "  ./$OUTPUT_NAME -config config.json"
echo ""
echo "Note: macOS may require granting USB/HID access permissions."
echo "  If the app is blocked by Gatekeeper, run:"
echo "  xattr -d com.apple.quarantine ./$OUTPUT_NAME"
echo ""
echo "Logs: steelclock.log in the same directory as the executable"
