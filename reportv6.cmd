@echo off
rem windows下获取ipv6地址上报，保存为cmd文件并在计划任务里登记，注意window文件编码,错误时会识别不出中文的关键字"临时"
rem 以下两个现要按需修改：YOUR-HOST-NAME和https://example.com/
rem 增加变量
set REPORT_URL=https://example.com/
set HOST_NAME=YOUR-HOST-NAME

SETLOCAL ENABLEDELAYEDEXPANSION
set firstIPv6=0
rem 中文环境里面是"临时"
for /f "tokens=5 delims= " %%A in ('netsh interface ipv6 show addresses ^| findstr "临时"') do (
    set firstIPv6=%%A
    goto sendRequest
)

:sendRequest
if "!firstIPv6!"=="0" (
goto end
)
if not exist .ipv6 (
    type nul > .ipv6
) 
set /p ipv6=<.ipv6
rem 移除可能存在的双引号
set ipv6=!ipv6:"=!

if "!firstIPv6!" neq "!ipv6!" (
    echo !firstIPv6!>.ipv6
    curl -X POST "!REPORT_URL!" -H "Content-Type: application/json" -d "{ \"host\": \"!HOST_NAME!\", \"ipv6\": \"!firstIPv6!\" }"
    rem echo curl -X POST "!REPORT_URL!" -H "Content-Type: application/json" -d "{ \"host\": \"!HOST_NAME!\", \"ipv6\": \"!firstIPv6!\" }"
    goto end
rem ) else (
rem    echo skip
)
:end
ENDLOCAL