#!/bin/bash
# Build script for SteelClock
# Ensures fresh resources on every build

set -e

echo "======================================"
echo "Building SteelClock for Windows"
echo "======================================"
echo ""

# Step 1: Cleanup old resources
echo "[1/6] Cleaning old resources..."
rm -f cmd/steelclock/*.syso
rm -f internal/tray/icon.ico
rm -f steelclock.exe
rm -f winres/*.syso
echo "✓ Cleanup complete"
echo ""

# Step 2: Check for go-winres
echo "[2/6] Checking for go-winres..."
WINRES_CMD=""

if command -v go-winres &> /dev/null; then
    WINRES_CMD="go-winres"
    echo "✓ go-winres found in PATH"
elif [ -f "$HOME/go/bin/go-winres" ]; then
    WINRES_CMD="$HOME/go/bin/go-winres"
    echo "✓ go-winres found in $HOME/go/bin"
else
    echo "✗ go-winres not found"
    echo ""
    echo "Installing go-winres..."
    if go install github.com/tc-hib/go-winres@latest; then
        WINRES_CMD="$HOME/go/bin/go-winres"
        echo "✓ go-winres installed successfully"
    else
        echo "✗ Failed to install go-winres"
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
    echo "✗ winres/winres.json not found"
    echo "  Skipping resource generation"
    echo ""
else
    # Generate .syso files in winres folder first
    if $WINRES_CMD make --out winres/rsrc 2>&1; then
        echo "✓ Resource files generated in winres/"

        # Copy .syso files to cmd/steelclock/ for compilation
        if ls winres/*.syso 1> /dev/null 2>&1; then
            cp winres/*.syso cmd/steelclock/
            echo "✓ Copied .syso files to cmd/steelclock/"
        else
            echo "⚠ Warning: No .syso files found in winres/"
        fi
    else
        echo "⚠ Warning: go-winres failed (missing icon files?)"
        echo "  Continuing without embedded resources"
    fi
fi
echo ""

# Step 4: Copy tray icon
echo "[4/6] Preparing tray icon..."
if [ -f "winres/icon.ico" ]; then
    cp winres/icon.ico internal/tray/icon.ico
    echo "✓ Copied icon.ico to internal/tray/"
else
    echo "⚠ Warning: winres/icon.ico not found"
    echo "  Tray icon will use default"
fi
echo ""

# Step 5: Build executable
echo "[5/6] Compiling executable..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o steelclock.exe ./cmd/steelclock
echo "✓ Compilation successful"
echo ""

# Step 6: Optional cleanup
echo "[6/6] Cleanup intermediate files..."
rm -f winres/*.syso
echo "✓ Removed intermediate .syso files from winres/"
echo ""

# Summary
echo "======================================"
echo "Build Summary"
echo "======================================"
ls -lh steelclock.exe
file steelclock.exe

# Check if resources are embedded
if objdump -h steelclock.exe 2>/dev/null | grep -q "\.rsrc"; then
    echo "✓ Windows resources (.rsrc) embedded"
else
    echo "⚠ No .rsrc section found (no icon embedded)"
fi

echo ""
echo "✓ Build complete!"
echo ""
echo "Usage:"
echo "  steelclock.exe          # Run with system tray"
echo "  steelclock.exe -config path/to/config.json"
echo ""
echo "Logs: steelclock.log in the same directory as the executable"
