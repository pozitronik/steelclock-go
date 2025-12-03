@echo off
setlocal enabledelayedexpansion

echo ======================================
echo Building SteelClock for Linux
echo (Cross-compiling from Windows)
echo ======================================
echo.

REM Step 1: Cleanup old build
echo [1/3] Cleaning old build...
if exist "steelclock" del /q "steelclock" 2>nul
echo [OK] Cleanup complete
echo.

REM Step 2: Copy tray icon (optional, for consistency)
echo [2/3] Preparing tray icon...
if exist "winres\icon.ico" (
    if not exist "internal\tray" mkdir "internal\tray"
    copy /y "winres\icon.ico" "internal\tray\icon.ico" >nul
    echo [OK] Copied icon.ico to internal\tray\
) else (
    echo [!] Warning: winres\icon.ico not found
    echo     Tray icon will use default
)
echo.

REM Step 3: Build executable
echo [3/3] Compiling executable...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o steelclock ./cmd/steelclock
if %errorlevel% neq 0 (
    echo.
    echo [X] Compilation failed!
    echo.
    echo Note: Cross-compiling to Linux from Windows requires CGO_ENABLED=0
    echo The system tray functionality may be limited without CGO.
    exit /b %errorlevel%
)
echo [OK] Compilation successful
echo.

REM Summary
echo ======================================
echo Build Summary
echo ======================================
dir steelclock | find "steelclock"

echo.
echo [OK] Build complete!
echo.
echo Note: This is a cross-compiled binary.
echo For full functionality (system tray), build natively on Linux.
echo.
echo Usage on Linux:
echo   ./steelclock                    # Run (requires udev rules for direct driver)
echo   ./steelclock -config config.json
echo.
echo For direct USB driver access, install udev rules:
echo   sudo cp profiles/99-steelseries.rules /etc/udev/rules.d/
echo   sudo udevadm control --reload-rules ^&^& sudo udevadm trigger
echo.
echo Logs: steelclock.log in the same directory as the executable
