@echo off
setlocal
title This Is Your Year - Remove the website

REM ===========================================================================
REM  Double-click this file to completely remove the website from this computer.
REM  This DELETES all accounts and data created in the app. It cannot be undone.
REM  (Docker Desktop itself is left installed - see the note at the end.)
REM ===========================================================================

set "COMPOSE_FILE=%~dp0deployments\docker-compose.customer.yml"

echo.
echo   ============================================================
echo      REMOVE the "This Is Your Year" website
echo.
echo      This will delete the website AND all data you created
echo      in it (accounts, topics, goals). This cannot be undone.
echo   ============================================================
echo.
set /p CONFIRM=  Type YES and press Enter to remove everything:

if /I not "%CONFIRM%"=="YES" (
    echo.
    echo   Cancelled. Nothing was removed.
    echo.
    pause
    exit /b 0
)

echo.
echo   Removing the website and its data...
echo.

REM -v removes the database (all data); --rmi local removes the built image.
docker compose -f "%COMPOSE_FILE%" down -v --rmi local

echo.
echo   ============================================================
echo      Done. The website and its data have been removed.
echo.
echo      Docker Desktop is still installed (other apps may use it).
echo      If you also want to remove it, go to:
echo         Windows Settings  ^>  Apps  ^>  installed apps  ^>
echo         Docker Desktop  ^>  Uninstall
echo.
echo      You can also delete this whole folder now.
echo   ============================================================
echo.
pause
endlocal
