@echo off
setlocal
set ARCH=amd64
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set ARCH=arm64
set "BIN=%~dp0pk-windows-%ARCH%.exe"
if not exist "%BIN%" goto fallback
"%BIN%" %*
exit /b %ERRORLEVEL%
:fallback
rem No binary beside the shim: use the pk on PATH, told where this plugin
rem is through PK_PLUGIN_ROOT. A PATH search for pk.exe never matches
rem this pk.cmd, so the shim cannot run itself.
set "FOUND="
for %%p in (pk.exe) do set "FOUND=%%~$PATH:p"
if not defined FOUND goto missing
set "PK_PLUGIN_ROOT=%~dp0.."
"%FOUND%" %*
exit /b %ERRORLEVEL%
:missing
echo pk: %BIN% not found; run 'make bin-local' in the plankit repo, reinstall the plugin, or install pk (go install github.com/markwharton/plankit/cmd/pk@latest) 1>&2
exit /b 3
