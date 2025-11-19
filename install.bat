@echo off
setlocal enabledelayedexpansion

:: Wordsmith Install Script for Windows
:: Builds locally if in git repo, otherwise downloads from GitHub releases

set "REPO=abrayall/wordsmith"
set "INSTALL_DIR=%USERPROFILE%\bin"

:: Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "ARCH=amd64"
) else if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    set "ARCH=amd64"
)

:: Check if we're in the wordsmith repo
if exist "go.mod" (
    findstr /c:"wordsmith" go.mod >nul 2>&1
    if !errorlevel! equ 0 (
        if exist "build.bat" (
            goto :build_local
        )
    )
)

:: Not in repo, download from GitHub
goto :download_release

:build_local
echo Building from source...
call build.bat

:: Find the built binary
set "BINARY="
for %%f in (build\wordsmith-*-windows-%ARCH%.exe) do set "BINARY=%%f"

if not defined BINARY (
    echo [91mError: No binary found for windows-%ARCH%[0m
    exit /b 1
)

goto :install

:download_release
echo [38;5;69mFetching latest release...[0m

:: Create temp directory
set "TMP_DIR=%TEMP%\wordsmith-install-%RANDOM%"
mkdir "%TMP_DIR%" 2>nul

:: Get latest release tag using PowerShell (minimal dependency, built into Windows)
for /f "tokens=*" %%i in ('powershell -NoProfile -Command "(Invoke-RestMethod -Uri 'https://api.github.com/repos/%REPO%/releases/latest').tag_name"') do set "LATEST=%%i"

if not defined LATEST (
    echo [91mError: Failed to fetch latest release[0m
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

echo [38;5;69mLatest version: [97m%LATEST%[0m
echo.

:: Remove 'v' prefix for filename
set "VERSION=%LATEST:~1%"
set "FILENAME=wordsmith-%VERSION%-windows-%ARCH%.exe"
set "URL=https://github.com/%REPO%/releases/download/%LATEST%/%FILENAME%"
set "BINARY=%TMP_DIR%\wordsmith.exe"

echo [38;5;69mDownloading %FILENAME%...[0m

:: Download using PowerShell
powershell -NoProfile -Command "try { Invoke-WebRequest -Uri '%URL%' -OutFile '%BINARY%' -UseBasicParsing } catch { exit 1 }"

if %errorlevel% neq 0 (
    echo [91mError: Failed to download from %URL%[0m
    rmdir /s /q "%TMP_DIR%" 2>nul
    exit /b 1
)

goto :install

:install
echo [38;5;69mInstalling to %INSTALL_DIR%...[0m

:: Create install directory if it doesn't exist
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Copy binary
copy /y "%BINARY%" "%INSTALL_DIR%\wordsmith.exe" >nul

if %errorlevel% neq 0 (
    echo Error: Failed to install to %INSTALL_DIR%
    exit /b 1
)

:: Check if install dir is in PATH
echo %PATH% | findstr /i /c:"%INSTALL_DIR%" >nul
if %errorlevel% neq 0 (
    echo.
    echo NOTE: %INSTALL_DIR% is not in your PATH.
    echo Add it to your PATH to run wordsmith from anywhere:
    echo   setx PATH "%%PATH%%;%INSTALL_DIR%"
    echo.
)

:: Cleanup temp files if downloaded
if defined TMP_DIR (
    if exist "%TMP_DIR%" rmdir /s /q "%TMP_DIR%" 2>nul
)

echo.
echo [92mSuccessfully installed wordsmith to %INSTALL_DIR%\wordsmith.exe[0m
echo.
echo Run 'wordsmith --help' to get started
echo.

endlocal
