# Chatlog Scripts

这个目录包含了 Chatlog 项目的构建和部署脚本。

## 📁 脚本说明

### 构建脚本
- **`build.bat`** - Windows 构建脚本，自动检查环境并构建项目
- **`build.sh`** - Linux/macOS 构建脚本，跨平台构建支持
- **`package.sh`** - 项目打包脚本

### 运行脚本  
- **`run.bat`** - 快速运行脚本，使用 `go run` 直接启动
- **`start-server.bat`** - 服务器启动脚本，使用预构建的可执行文件

### 工具脚本
- **`check-env.bat`** - 环境检查脚本，验证 Go 和 C 编译器是否正确安装

## 🚀 使用方法

### Windows 用户

```cmd
# 1. 检查环境
script\check-env.bat

# 2. 构建项目
script\build.bat

# 3. 启动服务器
script\start-server.bat
```

### Linux/macOS 用户

```bash
# 1. 构建项目
./script/build.sh

# 2. 启动服务器
./bin/chatlog server [参数...]
```

## 📋 注意事项

- 所有 Windows 脚本需要在已安装 C 编译器的环境中运行
- 如遇到 CGO 相关错误，请先运行 `check-env.bat` 检查环境
- 构建脚本会自动创建 `bin/` 目录并输出可执行文件

## 📖 详细文档

更多构建相关信息请参考：[构建文档](../docs/README-BUILD.md)