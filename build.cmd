@echo off
REM ---------------------------------------------------------------------------
REM build.cmd - Windows build entry point for TinyProxyAPI
REM
REM Why this wrapper exists:
REM   PowerShell's RemoteSigned execution policy refuses to load build.ps1 when
REM   the file carries a Low mandatory integrity label. That label is applied by
REM   sandboxed tooling to every newly created file under a workspace, and
REM   Windows starts a process at the LOWER of the user level and the file level,
REM   so a plain ".\build.ps1" fails with "not digitally signed".
REM   Batch files are not subject to PowerShell's execution policy, so this
REM   wrapper invokes the script with the policy bypassed on your behalf and you
REM   never have to type the flag yourself.
REM
REM Requirements:
REM   Run this from a normal (Medium integrity) PowerShell window. The
REM   "icacls /setintegritylevel M" step at the end of build.ps1 needs Medium
REM   integrity to succeed. Inside a Low integrity sandbox that step warns, and
REM   the produced executable cannot create its tray icon.
REM ---------------------------------------------------------------------------

setlocal

REM cd to this script's own directory so "go build" resolves go.mod no matter
REM which directory the caller invoked us from. %~dp0 always ends in a
REM backslash, so no separator is needed before build.ps1.
cd /d "%~dp0"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build.ps1" %*
set "RC=%ERRORLEVEL%"

if not "%RC%"=="0" echo [build.cmd] build failed with exit code %RC%

endlocal & exit /b %RC%
