@echo off
echo Building KnowsWell...

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo go mod tidy failed
    pause
    exit /b 1
)

go build -ldflags="-H windowsgui -s -w" -o .\Compiled\KnowsWell.exe .
if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful: KnowsWell.exe
    for %%I in (KnowsWell.exe) do echo Size: %%~zI bytes
) else (
    echo.
    echo Build FAILED
    pause
)
