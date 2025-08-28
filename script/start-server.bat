@echo off
setlocal enabledelayedexpansion

echo ===========================================
echo   Chatlog 服务器启动脚本
echo ===========================================

:: 检查是否存在已构建的可执行文件
if exist "bin\chatlog.exe" (
    echo ✅ 找到已构建的 chatlog.exe
    echo 🚀 启动服务器...
    echo.
    
    :: 使用已构建的可执行文件运行
    bin\chatlog.exe server --addr "100.119.132.40:5030" --data-dir "D:\MyFolders\WindowsDocuments\WeChat Files\wxid_8erobdogc9u022" --work-dir "C:\Users\zlx\Documents\chatlog\wxid_8erobdogc9u022" --platform windows --version 3 --data-key "5e13299164a246de8fa36e25c6778ad08623dc9d3e46466999e4da3f8bbbfb5f" --auto-decrypt
    
) else if exist "chatlog.exe" (
    echo ✅ 找到 chatlog.exe
    echo 🚀 启动服务器...
    echo.
    
    :: 使用根目录的可执行文件运行
    chatlog.exe server --addr "100.119.132.40:5030" --data-dir "D:\MyFolders\WindowsDocuments\WeChat Files\wxid_8erobdogc9u022" --work-dir "C:\Users\zlx\Documents\chatlog\wxid_8erobdogc9u022" --platform windows --version 3 --data-key "5e13299164a246de8fa36e25c6778ad08623dc9d3e46466999e4da3f8bbbfb5f" --auto-decrypt
    
) else (
    echo ❌ 找不到 chatlog.exe
    echo.
    echo 请先运行以下任一命令构建程序:
    echo   1. build.bat    (推荐 - 完整构建)
    echo   2. make build   (如果安装了 make)
    echo   3. 手动构建:
    echo      set CGO_ENABLED=1
    echo      go build -o chatlog.exe main.go
    echo.
    pause
    exit /b 1
)

if errorlevel 1 (
    echo.
    echo ❌ 服务器启动失败!
    echo.
    echo 可能的原因:
    echo   1. 数据目录不存在或无法访问
    echo   2. 工作目录无法创建
    echo   3. 端口被占用
    echo   4. 数据密钥不正确
    echo.
    pause
)