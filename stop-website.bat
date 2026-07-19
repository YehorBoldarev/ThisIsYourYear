@echo off
setlocal
title This Is Your Year - Stop the website

REM ===========================================================================
REM  Double-click this file to stop the website.
REM  Your account and data are kept - just run start-website.bat to use it again.
REM ===========================================================================

set "COMPOSE_FILE=%~dp0deployments\docker-compose.customer.yml"

echo.
echo   Stopping the website...
echo.

docker compose -f "%COMPOSE_FILE%" down

echo.
echo   ============================================================
echo      The website is stopped. Your data has been kept.
echo      Double-click start-website.bat whenever you want it back.
echo   ============================================================
echo.
pause
endlocal
