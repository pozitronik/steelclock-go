@echo off
setlocal

echo Building SteelClock for Windows...

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o steelclock.exe ./cmd/steelclock

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)

echo Build successful: steelclock.exe
