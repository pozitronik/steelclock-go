@echo off
setlocal

echo Building SteelClock for Windows...
echo.

REM Check if go-winres is available and generate Windows resources if configured
where go-winres.exe >nul 2>&1
if %errorlevel% equ 0 (
    if exist "winres\winres.json" (
        echo Generating Windows resources with go-winres...
        go-winres.exe make --out cmd\steelclock\rsrc
        if %errorlevel% equ 0 (
            echo [OK] Windows resources generated
            echo.
        ) else (
            echo Warning: go-winres failed, continuing without resources
            echo.
        )
    ) else (
        echo Note: winres\winres.json not found, skipping resource generation
        echo.
    )
) else (
    REM Try to find go-winres in user's Go bin directory
    if exist "%USERPROFILE%\go\bin\go-winres.exe" (
        if exist "winres\winres.json" (
            echo Generating Windows resources with go-winres...
            "%USERPROFILE%\go\bin\go-winres.exe" make --out cmd\steelclock\rsrc
            if %errorlevel% equ 0 (
                echo [OK] Windows resources generated
                echo.
            ) else (
                echo Warning: go-winres failed, continuing without resources
                echo.
            )
        )
    ) else (
        echo Note: go-winres not found, skipping resource generation
        echo       Install with: go install github.com/tc-hib/go-winres@latest
        echo.
    )
)

set GOOS=windows
set GOARCH=amd64

REM Build GUI version (default, no console window)
echo Compiling executable...
go build -ldflags="-s -w -H windowsgui" -o steelclock.exe ./cmd/steelclock

if %errorlevel% neq 0 (
    echo.
    echo Build failed!
    exit /b %errorlevel%
)

echo.
echo [OK] Build successful: steelclock.exe (GUI mode)
echo.
echo Run with -console flag to enable console mode
echo Example: steelclock.exe -console
