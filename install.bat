@echo off
setlocal

echo.

:: Get script directory
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

:: Build first
call build.bat

:: Find the built binary
for %%f in (build\wordsmith-*-windows-amd64.exe) do set "BINARY=%%f"

if "%BINARY%"=="" (
    echo No binary found for Windows
    exit /b 1
)

:: Install to user's local bin directory
set "INSTALL_DIR=%USERPROFILE%\bin"

if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

copy /Y "%BINARY%" "%INSTALL_DIR%\wordsmith.exe" > nul

echo.
echo Installed wordsmith to %INSTALL_DIR%\wordsmith.exe
echo.

:: Check if directory is in PATH
echo %PATH% | find /i "%INSTALL_DIR%" > nul
if errorlevel 1 (
    echo NOTE: %INSTALL_DIR% is not in your PATH.
    echo Add it with:
    echo   setx PATH "%PATH%;%INSTALL_DIR%"
    echo.
)

echo Run 'wordsmith --help' from anywhere to get started
echo.

endlocal
