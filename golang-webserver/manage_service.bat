@echo off
setlocal EnableDelayedExpansion

:: Check for Administrator Privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo This script requires Administrator privileges.
    echo Please run it as Administrator.
    pause
    exit /b 1
)

set SERVICE_NAME=ImageServer
set EXE_NAME=image_server.exe
set CONFIG_FILE=config.json

:: If no parameter is provided or if it's /? or -h or --help, show usage
if "%1"=="" goto show_usage
if "%1"=="/?" goto show_usage
if "%1"=="-h" goto show_usage
if "%1"=="--help" goto show_usage

:: Check if executable exists
if not exist "%~dp0%EXE_NAME%" (
    echo Error: %EXE_NAME% not found in script directory.
    echo Please place this script in the same directory as %EXE_NAME%
    pause
    exit /b 1
)

goto process_commands

:: Function to check service status
:check_service_status
sc query %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    for /f "tokens=4" %%i in ('sc query %SERVICE_NAME% ^| findstr STATE') do set state=%%i
    set /a state_num=state
) else (
    set state_num=0
)
goto :eof

:process_commands
if "%1"=="install" (
    echo Installing %SERVICE_NAME% service...
    "%~dp0%EXE_NAME%" install
    if !errorLevel! equ 0 (
        echo Service installed successfully.
        echo Configuring service...
        sc config %SERVICE_NAME% start= auto
        sc failure %SERVICE_NAME% reset= 86400 actions= restart/60000/restart/60000/restart/60000
        echo Service configured for automatic restart on failure.
    ) else (
        echo Failed to install service.
    )
    goto end
)

if "%1"=="remove" (
    echo Removing %SERVICE_NAME% service...
    call :check_service_status
    if !state_num! neq 0 (
        echo Stopping service first...
        net stop %SERVICE_NAME%
        timeout /t 2 /nobreak >nul
    )
    "%~dp0%EXE_NAME%" remove
    if !errorLevel! equ 0 (
        echo Service removed successfully.
    ) else (
        echo Failed to remove service.
    )
    goto end
)

if "%1"=="start" (
    echo Starting %SERVICE_NAME% service...
    net start %SERVICE_NAME%
    goto end
)

if "%1"=="stop" (
    echo Stopping %SERVICE_NAME% service...
    net stop %SERVICE_NAME%
    goto end
)

if "%1"=="restart" (
    echo Restarting %SERVICE_NAME% service...
    net stop %SERVICE_NAME%
    timeout /t 2 /nobreak >nul
    net start %SERVICE_NAME%
    goto end
)

if "%1"=="status" (
    call :check_service_status
    if !state_num! equ 0 (
        echo Service is not installed
    ) else (
        sc query %SERVICE_NAME%
    )
    goto end
)

if "%1"=="enable" (
    echo Enabling %SERVICE_NAME% service...
    sc config %SERVICE_NAME% start= auto
    call :check_service_status
    if !state_num! neq 4 (
        net start %SERVICE_NAME%
    )
    echo Service enabled and started.
    goto end
)

if "%1"=="disable" (
    echo Disabling %SERVICE_NAME% service...
    net stop %SERVICE_NAME%
    sc config %SERVICE_NAME% start= disabled
    echo Service stopped and disabled.
    goto end
)

if "%1"=="debug" (
    echo Running in debug mode...
    "%~dp0%EXE_NAME%" debug
    goto end
)

if "%1"=="config" (
    if "%2"=="" (
        if exist "%~dp0%CONFIG_FILE%" (
            type "%~dp0%CONFIG_FILE%"
        ) else (
            echo Config file not found.
        )
    ) else (
        echo Creating new config file...
        echo { > "%~dp0%CONFIG_FILE%"
        echo   "port": "%2", >> "%~dp0%CONFIG_FILE%"
        echo   "folder": "%3" >> "%~dp0%CONFIG_FILE%"
        echo } >> "%~dp0%CONFIG_FILE%"
        echo Config file created/updated.
    )
    goto end
)

:: If we get here, the command wasn't recognized
echo Unknown command: %1
echo.
goto show_usage

:show_usage
echo.
echo ImageServer Service Manager
echo =========================
echo Usage:
echo   %~n0 install        - Install the service
echo   %~n0 remove         - Remove the service
echo   %~n0 start          - Start the service
echo   %~n0 stop           - Stop the service
echo   %~n0 restart        - Restart the service
echo   %~n0 status         - Check service status
echo   %~n0 enable         - Enable and start service
echo   %~n0 disable        - Stop and disable service
echo   %~n0 debug          - Run in debug mode
echo   %~n0 config         - Show current config
echo   %~n0 config PORT FOLDER - Create/update config file
echo.
echo Examples:
echo   %~n0 config 8080 "C:\Images"
echo   %~n0 install
echo   %~n0 status
echo.
goto end

:end
endlocal
exit /b