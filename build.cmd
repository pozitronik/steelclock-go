@echo off
setlocal enabledelayedexpansion

REM Build script for SteelClock
REM Usage:
REM   build.cmd         - Full build (all widgets)
REM   build.cmd light   - Light build (excludes heavy widgets)

set BUILD_VARIANT=full
set BUILD_TAGS=
set OUTPUT_SUFFIX=

if /i "%~1"=="light" (
    set BUILD_VARIANT=light
    set BUILD_TAGS=-tags light
    set OUTPUT_SUFFIX=-light
)

echo ======================================
echo Building SteelClock for Windows (%BUILD_VARIANT%)
echo ======================================
echo.

REM Step 1: Cleanup old resources
echo [1/6] Cleaning old resources...
if exist "cmd\steelclock\*.syso" del /q "cmd\steelclock\*.syso" 2>nul
if exist "internal\tray\icon.ico" del /q "internal\tray\icon.ico" 2>nul
if exist "steelclock.exe" del /q "steelclock.exe" 2>nul
if exist "steelclock-light.exe" del /q "steelclock-light.exe" 2>nul
if exist "winres\*.syso" del /q "winres\*.syso" 2>nul
echo [OK] Cleanup complete
echo.

REM Step 2: Check for go-winres
echo [2/6] Checking for go-winres...
set WINRES_CMD=
set WINRES_FOUND=0

where go-winres.exe >nul 2>&1
if %errorlevel% equ 0 (
    set WINRES_CMD=go-winres.exe
    set WINRES_FOUND=1
    echo [OK] go-winres found in PATH
) else (
    if exist "%USERPROFILE%\go\bin\go-winres.exe" (
        set WINRES_CMD=%USERPROFILE%\go\bin\go-winres.exe
        set WINRES_FOUND=1
        echo [OK] go-winres found in %%USERPROFILE%%\go\bin
    ) else (
        echo [!] go-winres not found
        echo.
        echo Installing go-winres...
        go install github.com/tc-hib/go-winres@latest
        if %errorlevel% equ 0 (
            set WINRES_CMD=%USERPROFILE%\go\bin\go-winres.exe
            set WINRES_FOUND=1
            echo [OK] go-winres installed successfully
        ) else (
            echo [X] Failed to install go-winres
            echo.
            echo Please install manually:
            echo   go install github.com/tc-hib/go-winres@latest
            echo.
            exit /b 1
        )
    )
)
echo.

REM Step 3: Generate Windows resources (.syso files)
echo [3/6] Generating Windows resources...
if not exist "winres\winres.json" (
    echo [!] winres\winres.json not found
    echo     Skipping resource generation
    echo.
) else (
    REM Generate .syso files in winres folder first
    !WINRES_CMD! make --out winres\rsrc >nul 2>&1
    if %errorlevel% equ 0 (
        echo [OK] Resource files generated in winres\

        REM Copy .syso files to cmd\steelclock\ for compilation
        if exist "winres\*.syso" (
            copy /y "winres\*.syso" "cmd\steelclock\" >nul
            echo [OK] Copied .syso files to cmd\steelclock\
        ) else (
            echo [!] Warning: No .syso files found in winres\
        )
    ) else (
        echo [!] Warning: go-winres failed (missing icon files?)
        echo     Continuing without embedded resources
    )
)
echo.

REM Step 4: Copy tray icon
echo [4/6] Preparing tray icon...
if exist "winres\icon.ico" (
    copy /y "winres\icon.ico" "internal\tray\icon.ico" >nul
    echo [OK] Copied icon.ico to internal\tray\
) else (
    echo [!] Warning: winres\icon.ico not found
    echo     Tray icon will use default
)
echo.

REM Step 5: Build executable
echo [5/6] Compiling executable (%BUILD_VARIANT%)...
set GOOS=windows
set GOARCH=amd64
set OUTPUT_NAME=steelclock%OUTPUT_SUFFIX%.exe
go build %BUILD_TAGS% -ldflags="-s -w -H windowsgui" -o %OUTPUT_NAME% ./cmd/steelclock
if %errorlevel% neq 0 (
    echo.
    echo [X] Compilation failed!
    exit /b %errorlevel%
)
echo [OK] Compilation successful
echo.

REM Step 6: Cleanup intermediate files
echo [6/6] Cleanup intermediate files...
if exist "winres\*.syso" del /q "winres\*.syso" 2>nul
echo [OK] Removed intermediate .syso files from winres\
echo.

REM Summary
echo ======================================
echo Build Summary (%BUILD_VARIANT%)
echo ======================================
dir %OUTPUT_NAME% | find "%OUTPUT_NAME%"

REM Check if resources are embedded (using PowerShell as objdump may not be available)
powershell -Command "if ((Get-Content -Path '%OUTPUT_NAME%' -Encoding Byte -ReadCount 0 | ForEach-Object { [System.Text.Encoding]::ASCII.GetString($_) }) -match '\.rsrc') { Write-Host '[OK] Windows resources (.rsrc) embedded' -ForegroundColor Green } else { Write-Host '[!] No .rsrc section found (no icon embedded)' -ForegroundColor Yellow }" 2>nul

echo.
echo [OK] Build complete!
echo.
echo Usage:
echo   %OUTPUT_NAME%          # Run with system tray
echo   %OUTPUT_NAME% -config path\to\config.json
echo.
echo Logs: steelclock.log in the same directory as the executable
