@echo off
setlocal enabledelayedexpansion

echo ======================================
echo Wordsmith Build
echo ======================================
echo.

:: Get script directory
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

:: Build directory
set "BUILD_DIR=%SCRIPT_DIR%build"

:: Clean previous build
echo Cleaning previous build...
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%"

:: Get version from latest git tag
echo Reading version from git tags...
for /f "tokens=*" %%i in ('git describe --tags --match "v*.*.*" 2^>nul') do set "GIT_DESCRIBE=%%i"
if "%GIT_DESCRIBE%"=="" set "GIT_DESCRIBE=v0.1.0"

:: Parse version (simplified parsing for batch)
set "VERSION=%GIT_DESCRIBE:~1%"

:: Check for uncommitted changes
git status --porcelain > nul 2>&1
for /f %%i in ('git status --porcelain 2^>nul') do (
    for /f "tokens=1-4 delims=/ " %%a in ('date /t') do set "DATESTAMP=%%c%%a%%b"
    for /f "tokens=1-2 delims=: " %%a in ('time /t') do set "TIMESTAMP=%%a%%b"
    set "VERSION=%VERSION%-!DATESTAMP!!TIMESTAMP!"
    echo Detected uncommitted changes, appending timestamp
    goto :done_dirty
)
:done_dirty

echo Building version: %VERSION%
echo.

:: Build for multiple platforms
echo Building darwin/amd64...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-X wordsmith/cmd.Version=%VERSION%" -o "%BUILD_DIR%\wordsmith-%VERSION%-darwin-amd64" .
echo Created: wordsmith-%VERSION%-darwin-amd64

echo Building darwin/arm64...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags "-X wordsmith/cmd.Version=%VERSION%" -o "%BUILD_DIR%\wordsmith-%VERSION%-darwin-arm64" .
echo Created: wordsmith-%VERSION%-darwin-arm64

echo Building linux/amd64...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-X wordsmith/cmd.Version=%VERSION%" -o "%BUILD_DIR%\wordsmith-%VERSION%-linux-amd64" .
echo Created: wordsmith-%VERSION%-linux-amd64

echo Building linux/arm64...
set GOOS=linux
set GOARCH=arm64
go build -ldflags "-X wordsmith/cmd.Version=%VERSION%" -o "%BUILD_DIR%\wordsmith-%VERSION%-linux-arm64" .
echo Created: wordsmith-%VERSION%-linux-arm64

echo Building windows/amd64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X wordsmith/cmd.Version=%VERSION%" -o "%BUILD_DIR%\wordsmith-%VERSION%-windows-amd64.exe" .
echo Created: wordsmith-%VERSION%-windows-amd64.exe

echo.
echo ======================================
echo Build Complete!
echo ======================================
echo.
echo Artifacts created in build\:
dir /b "%BUILD_DIR%"
echo.

endlocal
