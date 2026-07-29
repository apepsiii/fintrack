@echo off
echo =============================================
echo   FinTrack - Financial Tracker Server
echo =============================================
echo.

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed.
    echo Please install Go 1.21 or higher from https://go.dev/dl/
    pause
    exit /b 1
)

REM Display Go version
for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
echo [OK] Go version: %GO_VERSION%
echo.

REM Download dependencies if needed
if not exist "go.sum" (
    echo [INFO] Downloading dependencies...
    go mod tidy
    echo.
)

REM Run the application
echo [INFO] Starting server at http://localhost:8080
echo.
echo Available pages:
echo   - Login:      http://localhost:8080/login
echo   - Dashboard:  http://localhost:8080/
echo   - Statistics: http://localhost:8080/stats
echo   - Targets:    http://localhost:8080/targets
echo   - Profile:    http://localhost:8080/profile
echo.
echo Press Ctrl+C to stop the server
echo =============================================
echo.

go run main.go
