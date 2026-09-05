@echo off
setlocal
set ARCH=amd64
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set ARCH=arm64
"%~dp0pk-windows-%ARCH%.exe" %*
