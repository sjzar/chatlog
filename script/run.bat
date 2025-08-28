@echo off
setlocal enabledelayedexpansion

echo ===========================================
echo   Chatlog 快速运行脚本
echo ===========================================

:: 设置 CGO 环境
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

echo ✅ 环境设置:
echo    CGO_ENABLED=1
echo    GOOS=windows
echo    GOARCH=amd64

:: 检查 Go 和编译器
where go >nul 2>&1
if errorlevel 1 (
    echo ❌ 错误: 找不到 Go 编译器
    pause
    exit /b 1
)

where gcc >nul 2>&1 || where cl >nul 2>&1
if errorlevel 1 (
    echo ❌ 错误: 找不到 C 编译器 (gcc 或 cl.exe)
    echo    请运行 build.bat 或安装 TDM-GCC/MinGW-w64
    pause
    exit /b 1
)

echo.
echo 🚀 启动 Chatlog 服务器...
echo    注意: 首次运行可能需要较长时间进行编译

:: 运行服务器
go run main.go server --addr "100.119.132.40:5030" --data-dir "D:\MyFolders\WindowsDocuments\WeChat Files\wxid_8erobdogc9u022" --work-dir "C:\Users\zlx\Documents\chatlog\wxid_8erobdogc9u022" --platform windows --version 3 --data-key "5e13299164a246de8fa36e25c6778ad08623dc9d3e46466999e4da3f8bbbfb5f" --auto-decrypt

if errorlevel 1 (
    echo.
    echo ❌ 服务器启动失败!
    echo    建议: 先运行 build.bat 构建可执行文件，再使用构建好的文件运行
    pause
)