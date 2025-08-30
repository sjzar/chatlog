# 故障排除指南

## 🚨 常见问题快速定位

### 问题分类

- **🔑 密钥相关**: 密钥获取失败、验证失败
- **🗄️ 数据解密**: 解密失败、数据库损坏
- **🖥️ 界面问题**: TUI 显示异常、操作失效
- **🌐 服务相关**: HTTP 服务启动失败、API 错误
- **🎵 语音处理**: 语音转换失败、播放异常
- **🖼️ 图片处理**: 图片解密失败、显示异常

## 🔑 密钥相关问题

### 问题：获取密钥失败

**症状:**
```
Error: failed to extract data key: process not found
Error: failed to get memory regions
Error: access denied
```

**可能原因:**
1. 微信进程未运行
2. 权限不足
3. 微信版本不支持
4. 安全软件阻拦

**解决方案:**

#### Windows 系统
```bash
# 1. 确认微信正在运行
tasklist | findstr WeChat

# 2. 以管理员身份运行 chatlog
# 右键 -> 以管理员身份运行

# 3. 检查防火墙和杀毒软件设置
# 临时关闭实时防护，重新尝试
```

#### macOS 系统
```bash
# 1. 确认微信正在运行
ps aux | grep WeChat

# 2. 检查 SIP 状态
csrutil status

# 3. 临时关闭 SIP (需要在恢复模式下执行)
# 重启按住 Cmd+R 进入恢复模式
csrutil disable

# 4. 安装必要工具
xcode-select --install

# 5. 获取密钥后重新启用 SIP
csrutil enable
```

### 问题：密钥验证失败

**症状:**
```
Error: data key validation failed
Error: invalid key format
```

**解决方案:**

1. **重新获取密钥**
```bash
# 确保微信进程稳定运行
chatlog key

# 或在 TUI 中重新获取密钥
```

2. **手动设置密钥**
```bash
# 如果有已知的正确密钥
chatlog server --data-key "your-key-here"
```

3. **检查密钥格式**
```bash
# 密钥应该是 64 字符的十六进制字符串
# 例如: 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

## 🗄️ 数据解密问题

### 问题：解密失败

**症状:**
```
Error: failed to decrypt database
Error: file is not a database
Error: database is locked
```

**诊断步骤:**

1. **检查数据目录**
```bash
# 列出数据目录内容
ls -la "/path/to/WeChat Files/wxid_xxx"

# 检查关键文件是否存在
ls MSG*.db
ls MediaMSG*.db
ls MicroMsg.db
```

2. **检查文件权限**
```bash
# Windows
icacls "MSG0.db"

# macOS/Linux
ls -l MSG0.db
```

3. **检查微信进程**
```bash
# 确保微信已完全关闭
# Windows
taskkill /f /im WeChat.exe

# macOS
pkill -f WeChat
```

**解决方案:**

1. **重新尝试解密**
```bash
# 完全关闭微信后重试
chatlog decrypt
```

2. **检查磁盘空间**
```bash
# 确保有足够的磁盘空间存储解密后的文件
df -h  # macOS/Linux
dir    # Windows
```

3. **使用命令行指定参数**
```bash
chatlog decrypt --data-dir "/path/to/wechat/data" --data-key "your-key"
```

### 问题：数据库损坏

**症状:**
```
Error: database disk image is malformed
Error: no such table
```

**解决方案:**

1. **检查原始数据库**
```bash
# 使用 SQLite 命令行工具检查
sqlite3 MSG0.db ".schema"
sqlite3 MSG0.db "PRAGMA integrity_check;"
```

2. **重新复制数据库文件**
```bash
# 从微信数据目录重新复制
# 确保微信已关闭
```

3. **尝试修复数据库**
```bash
# 使用 SQLite 修复
sqlite3 damaged.db ".recover" | sqlite3 recovered.db
```

## 🖥️ 界面问题

### 问题：TUI 显示异常

**症状:**
- 界面乱码
- 花屏
- 按键无响应
- 中文显示异常

**解决方案:**

#### Windows 系统
```cmd
# 1. 使用 Windows Terminal (推荐)
# 从 Microsoft Store 安装 Windows Terminal

# 2. 设置控制台字体
# 右键控制台标题栏 -> 属性 -> 字体 -> 选择支持中文的字体

# 3. 设置代码页
chcp 65001

# 4. 设置环境变量
set TERM=xterm-256color
```

#### macOS/Linux 系统
```bash
# 1. 检查终端设置
echo $TERM
echo $LANG

# 2. 设置正确的环境变量
export TERM=xterm-256color
export LANG=zh_CN.UTF-8

# 3. 使用支持中文的终端
# 推荐使用 iTerm2 (macOS) 或 GNOME Terminal (Linux)
```

### 问题：操作无响应

**症状:**
- 按键不起作用
- 界面冻结
- 无法退出程序

**解决方案:**

1. **强制退出**
```bash
# 使用 Ctrl+C 强制退出
# 如果无效，在另一个终端中:
pkill -f chatlog  # macOS/Linux
taskkill /f /im chatlog.exe  # Windows
```

2. **检查终端兼容性**
```bash
# 尝试在不同终端中运行
# Windows: cmd, PowerShell, Windows Terminal
# macOS: Terminal.app, iTerm2
# Linux: gnome-terminal, konsole
```

## 🌐 服务相关问题

### 问题：HTTP 服务启动失败

**症状:**
```
Error: bind: address already in use
Error: listen tcp 127.0.0.1:5030: bind: permission denied
```

**解决方案:**

1. **检查端口占用**
```bash
# Windows
netstat -ano | findstr :5030

# macOS/Linux
lsof -i :5030
netstat -tlnp | grep :5030
```

2. **终止占用进程**
```bash
# Windows
taskkill /f /pid <PID>

# macOS/Linux
kill -9 <PID>
```

3. **使用其他端口**
```bash
chatlog server --addr "127.0.0.1:8080"
```

4. **检查防火墙设置**
```bash
# Windows 防火墙可能阻止端口绑定
# 添加防火墙例外规则
```

### 问题：API 请求失败

**症状:**
```
Error: connection refused
Error: timeout
HTTP 500 Internal Server Error
```

**诊断步骤:**

1. **检查服务状态**
```bash
curl http://127.0.0.1:5030/health
```

2. **检查日志输出**
```bash
# 在启动服务时查看详细日志
chatlog server --debug
```

3. **测试基本接口**
```bash
# 测试联系人接口
curl "http://127.0.0.1:5030/api/v1/contact?limit=1"

# 测试聊天记录接口
curl "http://127.0.0.1:5030/api/v1/chatlog?limit=1"
```

**解决方案:**

1. **重启服务**
```bash
# 停止当前服务
# Ctrl+C 或在 TUI 中停止

# 重新启动
chatlog server
```

2. **检查数据库连接**
```bash
# 确保数据已正确解密
ls -la work_directory/
```

3. **更新配置**
```bash
# 检查配置文件
cat ~/.config/chatlog/config.json
```

## 🎵 语音处理问题

### 问题：语音转换失败

**症状:**
```
Error: failed to convert SILK to MP3
Error: no such table: Media
Error: voice data is empty
```

**解决方案:**

1. **检查语音数据库**
```bash
# 检查 MediaMSG 数据库是否存在
ls MediaMSG*.db

# 检查表结构
sqlite3 MediaMSG0.db ".schema Media"
```

2. **启用详细日志**
```bash
# 查看详细的语音处理日志
export LOG_LEVEL=debug
chatlog server

# 或使用命令行参数
chatlog server --debug
```

3. **测试语音 API**
```bash
# 获取语音消息 ID (从聊天记录中)
curl "http://127.0.0.1:5030/api/v1/chatlog?msg_type=34&limit=1"

# 尝试获取语音文件
curl "http://127.0.0.1:5030/voice/message_id" -o test.mp3
```

### 问题：语音播放异常

**症状:**
- 下载的 MP3 文件无法播放
- 音质异常
- 文件大小为 0

**解决方案:**

1. **检查源数据**
```bash
# 查看原始 SILK 数据大小
sqlite3 MediaMSG0.db "SELECT length(Buf) FROM Media WHERE Reserved0 = 'message_id'"
```

2. **验证转换过程**
```bash
# 检查转换过程中的日志输出
# 查找 "SILK解码成功" 和 "MP3编码成功" 消息
```

3. **手动测试转换**
```bash
# 如果有 SILK 文件，可以尝试手动转换
# (需要相关的转换工具)
```

## 🖼️ 图片处理问题

### 问题：图片解密失败

**症状:**
```
Error: failed to decrypt image
Error: invalid image format
Error: image key not found
```

**解决方案:**

1. **检查图片密钥**
```bash
# 确保已获取图片密钥
# 在 TUI 中查看密钥信息
```

2. **测试图片 API**
```bash
# 从消息中获取图片 ID
curl "http://127.0.0.1:5030/api/v1/chatlog?msg_type=3&limit=1"

# 尝试访问图片
curl "http://127.0.0.1:5030/image/image_id" -o test.jpg
```

3. **检查图片文件**
```bash
# 查看原始图片文件
ls -la "Data/Thumb" "Data/Dat"
```

## 🔧 系统级问题

### 问题：权限不足

**症状:**
```
Error: permission denied
Error: access is denied
Error: operation not permitted
```

**解决方案:**

#### Windows
```cmd
# 1. 以管理员身份运行
# 右键程序图标 -> 以管理员身份运行

# 2. 检查用户权限
whoami /priv

# 3. 临时关闭 UAC (不推荐)
```

#### macOS
```bash
# 1. 使用 sudo (谨慎使用)
sudo chatlog key

# 2. 检查文件权限
ls -la /path/to/file

# 3. 修改权限
chmod 644 /path/to/file
```

### 问题：内存不足

**症状:**
```
Error: cannot allocate memory
Error: out of memory
```

**解决方案:**

1. **检查可用内存**
```bash
# Windows
wmic OS get TotalVisibleMemorySize,FreePhysicalMemory

# macOS/Linux
free -h
vm_stat  # macOS
```

2. **优化内存使用**
```bash
# 分批处理数据
chatlog decrypt --batch-size 100

# 限制并发数
chatlog server --max-connections 50
```

3. **增加虚拟内存**
```bash
# Windows: 控制面板 -> 系统 -> 高级系统设置 -> 虚拟内存
# Linux: 增加 swap 分区
```

## 📊 性能问题

### 问题：响应速度慢

**症状:**
- API 请求超时
- 数据库查询缓慢
- 界面操作延迟

**诊断工具:**

1. **性能分析**
```bash
# 启用性能分析
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap
```

2. **数据库分析**
```bash
# SQLite 查询计划
sqlite3 database.db "EXPLAIN QUERY PLAN SELECT ..."

# 检查索引使用情况
sqlite3 database.db "PRAGMA table_info(MSG)"
```

**优化建议:**

1. **数据库优化**
```bash
# 为常用查询添加索引
sqlite3 database.db "CREATE INDEX IF NOT EXISTS idx_msg_talker ON MSG(StrTalker)"
sqlite3 database.db "CREATE INDEX IF NOT EXISTS idx_msg_time ON MSG(CreateTime)"
```

2. **查询优化**
```bash
# 使用分页查询
curl "http://127.0.0.1:5030/api/v1/chatlog?limit=50&offset=0"

# 缩小时间范围
curl "http://127.0.0.1:5030/api/v1/chatlog?time=2024-01-01&limit=100"
```

## 🔍 调试技巧

### 收集调试信息

```bash
# 1. 启用详细日志
export LOG_LEVEL=debug
chatlog server 2>&1 | tee chatlog.log

# 2. 收集系统信息
chatlog version
go version
echo $GOOS $GOARCH

# 3. 检查配置文件
cat ~/.config/chatlog/config.json

# 4. 检查数据库连接
sqlite3 database.db "SELECT count(*) FROM sqlite_master WHERE type='table'"
```

### 常用诊断命令

```bash
# 检查进程
ps aux | grep chatlog      # Unix-like
tasklist | findstr chatlog # Windows

# 检查网络连接
netstat -tlnp | grep 5030  # Linux
netstat -an | findstr 5030 # Windows
lsof -i :5030             # macOS

# 检查磁盘空间
df -h                     # Unix-like
dir                       # Windows

# 检查内存使用
free -h                   # Linux
vm_stat                   # macOS
wmic OS get FreePhysicalMemory # Windows
```

## 📞 获取帮助

### 提交问题前的准备

1. **收集基本信息**
   - 操作系统版本
   - Chatlog 版本
   - 微信版本
   - 错误日志

2. **重现步骤**
   - 详细的操作步骤
   - 期望的结果
   - 实际的结果

3. **相关文件**
   - 配置文件
   - 日志文件
   - 错误截图

### 联系方式

- **GitHub Issues**: [提交问题](https://github.com/sjzar/chatlog/issues)
- **GitHub Discussions**: [技术讨论](https://github.com/sjzar/chatlog/discussions)
- **文档建议**: 直接编辑相关文档文件并提交 PR

### 问题模板

```markdown
**问题描述**
简要描述遇到的问题。

**环境信息**
- 操作系统: [例如 Windows 11, macOS 14.0, Ubuntu 22.04]
- Chatlog 版本: [例如 v1.0.0]
- 微信版本: [例如 3.9.5.81]
- Go 版本: [例如 go1.24.0]

**重现步骤**
1. 启动程序
2. 执行操作 '...'
3. 发生错误

**期望行为**
期望程序应该...

**实际行为**
程序实际...

**错误日志**
```
[粘贴错误日志]
```

**附加信息**
其他可能有助于解决问题的信息。
```