#!/bin/bash
# Build script for SteelClock
# Ensures fresh resources on every build
#
# Usage:
#   ./build.sh         # Full build (all widgets)
#   ./build.sh --light # Light build (excludes heavy widgets)
#   ./build.sh -l      # Same as --light

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
echo "Building SteelClock for Windows ($BUILD_VARIANT)"
echo "======================================"
echo ""

# Step 1: Cleanup old resources
echo "[1/6] Cleaning old resources..."
rm -f cmd/steelclock/*.syso
rm -f internal/tray/icon.ico
rm -f steelclock.exe steelclock-light.exe
rm -f winres/*.syso
echo "OK Cleanup complete"
echo ""

# Step 2: Check for go-winres
echo "[2/6] Checking for go-winres..."
WINRES_CMD=""

if command -v go-winres &> /dev/null; then
    WINRES_CMD="go-winres"
    echo "OK go-winres found in PATH"
elif [ -f "$HOME/go/bin/go-winres" ]; then
    WINRES_CMD="$HOME/go/bin/go-winres"
    echo "OK go-winres found in $HOME/go/bin"
else
    echo "X go-winres not found"
    echo ""
    echo "Installing go-winres..."
    if go install github.com/tc-hib/go-winres@latest; then
        WINRES_CMD="$HOME/go/bin/go-winres"
        echo "OK go-winres installed successfully"
    else
        echo "X Failed to install go-winres"
        echo ""
        echo "Please install manually:"
        echo "  go install github.com/tc-hib/go-winres@latest"
        echo ""
        exit 1
    fi
fi
echo ""

# Step 3: Generate Windows resources (.syso files)
echo "[3/6] Generating Windows resources..."
if [ ! -f "winres/winres.json" ]; then
    echo "X winres/winres.json not found"
    echo "  Skipping resource generation"
    echo ""
else
    # Generate .syso files in winres folder first
    if $WINRES_CMD make --out winres/rsrc 2>&1; then
        echo "OK Resource files generated in winres/"

        # Copy .syso files to cmd/steelclock/ for compilation
        if ls winres/*.syso 1> /dev/null 2>&1; then
            cp winres/*.syso cmd/steelclock/
            echo "OK Copied .syso files to cmd/steelclock/"
        else
            echo "!! Warning: No .syso files found in winres/"
        fi
    else
        echo "!! Warning: go-winres failed (missing icon files?)"
        echo "  Continuing without embedded resources"
    fi
fi
echo ""

# Step 4: Copy tray icon
echo "[4/6] Preparing tray icon..."
if [ -f "winres/icon.ico" ]; then
    cp winres/icon.ico internal/tray/icon.ico
    echo "OK Copied icon.ico to internal/tray/"
else
    echo "!! Warning: winres/icon.ico not found"
    echo "  Tray icon will use default"
fi
echo ""

# Step 5: Build executable
echo "[5/6] Compiling executable ($BUILD_VARIANT)..."
OUTPUT_NAME="steelclock${OUTPUT_SUFFIX}.exe"
GOOS=windows GOARCH=amd64 go build $BUILD_TAGS -ldflags="-s -w -H windowsgui" -o "$OUTPUT_NAME" ./cmd/steelclock
echo "OK Compilation successful"
echo ""

# Step 6: Optional cleanup
echo "[6/6] Cleanup intermediate files..."
rm -f winres/*.syso
echo "OK Removed intermediate .syso files from winres/"
echo ""

# Summary
echo "======================================"
echo "Build Summary ($BUILD_VARIANT)"
echo "======================================"
ls -lh "$OUTPUT_NAME"
file "$OUTPUT_NAME"

# Check if resources are embedded
if objdump -h "$OUTPUT_NAME" 2>/dev/null | grep -q "\.rsrc"; then
    echo "OK Windows resources (.rsrc) embedded"
else
    echo "!! No .rsrc section found (no icon embedded)"
fi

echo ""
echo "OK Build complete!"
echo ""
echo "Usage:"
echo "  $OUTPUT_NAME          # Run with system tray"
echo "  $OUTPUT_NAME -config path/to/config.json"
echo ""
echo "Logs: steelclock.log in the same directory as the executable"
