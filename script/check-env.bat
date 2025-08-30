@echo off
echo ===========================================
echo   Chatlog 环境检查工具
echo ===========================================

echo 🔍 检查开发环境...
echo.

:: 检查 Go
where go >nul 2>&1
if errorlevel 1 (
    echo ❌ Go: 未安装
    echo    下载地址: https://golang.org/dl/
    set "GO_OK=0"
) else (
    echo ✅ Go: 已安装
    go version
    set "GO_OK=1"
)

echo.

:: 检查 GCC
where gcc >nul 2>&1
if errorlevel 1 (
    echo ❌ GCC: 未安装
    set "GCC_OK=0"
    
    :: 检查其他编译器
    where cl >nul 2>&1
    if errorlevel 1 (
        echo ❌ Visual Studio 编译器: 未安装
        echo    建议安装 TDM-GCC: https://jmeubank.github.io/tdm-gcc/
        set "CC_OK=0"
    ) else (
        echo ✅ Visual Studio 编译器: 已安装
        set "CC_OK=1"
    )
) else (
    echo ✅ GCC: 已安装
    gcc --version | findstr gcc
    set "GCC_OK=1"
    set "CC_OK=1"
)

echo.

:: 检查 CGO 环境变量
if "%CGO_ENABLED%"=="1" (
    echo ✅ CGO_ENABLED: 已启用 (1)
) else if "%CGO_ENABLED%"=="0" (
    echo ⚠️  CGO_ENABLED: 已禁用 (0) - 需要设置为 1
) else (
    echo ℹ️  CGO_ENABLED: 未设置 (默认启用)
)

echo.

:: 检查项目依赖
if exist go.mod (
    echo ✅ go.mod: 存在
    echo    检查主要依赖...
    findstr "go-sqlite3" go.mod >nul 2>&1
    if errorlevel 1 (
        echo ❌ go-sqlite3: 未找到
    ) else (
        echo ✅ go-sqlite3: 已添加
    )
) else (
    echo ❌ go.mod: 不存在
    echo    请确保在项目根目录运行此脚本
)

echo.

:: 总结和建议
echo ===========================================
echo   环境检查总结
echo ===========================================

if "%GO_OK%"=="1" (
    if "%CC_OK%"=="1" (
        echo ✅ 环境检查通过! 可以构建项目
        echo.
        echo 🚀 推荐步骤:
        echo    1. 运行 build.bat 构建程序
        echo    2. 运行 start-server.bat 启动服务器
    ) else (
        echo ❌ 缺少 C 编译器
        echo.
        echo 📋 修复步骤:
        echo    1. 安装 TDM-GCC: https://jmeubank.github.io/tdm-gcc/
        echo    2. 或安装 MinGW-w64: https://www.mingw-w64.org/downloads/
        echo    3. 重新启动命令提示符
        echo    4. 运行 build.bat
    )
) else (
    echo ❌ 缺少 Go 编译器
    echo.
    echo 📋 修复步骤:
    echo    1. 安装 Go: https://golang.org/dl/
    echo    2. 安装 C 编译器 (TDM-GCC 或 MinGW-w64)
    echo    3. 重新启动命令提示符
    echo    4. 运行 build.bat
)

echo.
pause