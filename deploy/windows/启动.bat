@echo off
chcp 936 >nul
cd /d %~dp0
title AnqiCMS

rem 同一个端口只能起一个实例：重复双击时先停掉旧进程
taskkill /f /im anqicms.exe >nul 2>&1

if exist data\installed.lock (
    echo 正在启动 AnqiCMS ...
    echo.
    echo   前台 : http://127.0.0.1:8001/
    echo   后台 : http://127.0.0.1:8001/system/
    echo   关闭本窗口即停止服务
    echo.
    start "" http://127.0.0.1:8001/
) else (
    echo ============================================================
    echo   首次运行：启动安装向导
    echo ============================================================
    echo   浏览器会自动打开安装页面（约 2 秒）。
    echo   若没有自动打开，请手动访问： http://127.0.0.1:8001/install
    echo.
    echo   在页面里填写：数据库地址/端口/库名/账号/密码、
    echo   后台用户名、后台密码、网站地址，点「确认初始化」。
    echo   勾选「演示数据」可以先带上一批示例内容。
    echo ============================================================
    echo.
    start "" http://127.0.0.1:8001/install
)

anqicms.exe
echo.
echo 服务已停止。
pause
