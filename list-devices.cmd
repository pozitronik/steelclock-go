@echo off
REM SteelClock device diagnostic.
REM Keep this file next to steelclock.exe, then double-click it.
REM It lists the connected SteelSeries USB devices, shows them in a popup,
REM and writes "steelclock-devices.txt" next to the program for you to attach.
"%~dp0steelclock.exe" -list-devices
