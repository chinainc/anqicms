@echo off
chcp 936 >nul
echo 正在停止 AnqiCMS ...
taskkill /f /im anqicms.exe >nul 2>&1
if errorlevel 1 (
    echo 没有找到正在运行的 anqicms.exe
) else (
    echo [OK] 已停止
)
pause
