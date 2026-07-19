@echo off
setlocal enabledelayedexpansion
title This Is Your Year - Start the website

REM ===========================================================================
REM  Double-click this file to start the website.
REM  The first time, it sets up everything for you. Later launches are quick.
REM  When it finishes, the site opens in your browser at http://localhost:8080
REM ===========================================================================

set "COMPOSE_FILE=%~dp0deployments\docker-compose.customer.yml"

echo.
echo   ============================================================
echo      THIS IS YOUR YEAR  -  starting the website
echo   ============================================================
echo.

REM --- 1. Make sure the engine that runs the website is installed -------------
where docker >nul 2>nul
if errorlevel 1 (
    echo   The one-time setup component ^(Docker Desktop^) is not installed yet.
    echo   Installing it now. Please click "Yes" if Windows asks for permission.
    echo.
    where winget >nul 2>nul
    if errorlevel 1 (
        echo   Automatic setup is not available on this version of Windows.
        echo   Please open this link, click "Download", run the installer, then
        echo   double-click start-website.bat again:
        echo.
        echo       https://www.docker.com/products/docker-desktop/
        echo.
        pause
        exit /b 1
    )
    winget install -e --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements
    echo.
    echo   ------------------------------------------------------------
    echo   Setup finished. Please RESTART your computer now, then
    echo   double-click start-website.bat again to launch the website.
    echo   ------------------------------------------------------------
    echo.
    pause
    exit /b 0
)

REM --- 2. Make sure that engine is actually running --------------------------
docker info >nul 2>nul
if errorlevel 1 (
    echo   Starting Docker Desktop... ^(this can take a minute the first time^)
    if exist "%ProgramFiles%\Docker\Docker\Docker Desktop.exe" (
        start "" "%ProgramFiles%\Docker\Docker\Docker Desktop.exe"
    )
    set /a tries=0
    :waitengine
    timeout /t 3 >nul
    docker info >nul 2>nul
    if not errorlevel 1 goto engineready
    set /a tries+=1
    if !tries! geq 60 (
        echo.
        echo   Docker Desktop did not finish starting.
        echo   Please open "Docker Desktop" from the Start menu, wait until it
        echo   says it is running, then double-click start-website.bat again.
        echo.
        pause
        exit /b 1
    )
    echo   ...still starting ^(!tries!^)
    goto waitengine
)
:engineready

echo.
echo   Building and starting the website.
echo   The FIRST time this can take several minutes - please leave this
echo   window open. Later launches take only a few seconds.
echo.

REM --- 3. Build (first time) and start everything ----------------------------
docker compose -f "%COMPOSE_FILE%" up -d --build
if errorlevel 1 (
    echo.
    echo   Something went wrong while starting the website.
    echo   Please make sure you are connected to the internet and try again.
    echo.
    pause
    exit /b 1
)

REM --- 4. Wait until the website is ready, then open it -----------------------
echo.
echo   Almost there - waiting for the website to be ready...
set /a tries=0
:waitsite
timeout /t 3 >nul
curl -sf http://localhost:8080/healthz >nul 2>nul
if not errorlevel 1 goto siteready
set /a tries+=1
if !tries! geq 60 (
    echo.
    echo   The website is taking longer than expected to start.
    echo   Try opening http://localhost:8080 in your browser in a minute.
    echo.
    pause
    exit /b 1
)
goto waitsite
:siteready

start "" http://localhost:8080

echo.
echo   ============================================================
echo      The website is running!
echo.
echo      It just opened in your browser. If not, go to:
echo          http://localhost:8080
echo.
echo      Create an account with "Register" to begin.
echo.
echo      To stop it later:   double-click  stop-website.bat
echo   ============================================================
echo.
pause
endlocal
