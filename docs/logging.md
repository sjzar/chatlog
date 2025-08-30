# 日志系统使用说明

## 概述

Chatlog 使用结构化日志系统来记录应用程序的运行状态、错误信息和调试详情。基于 `zerolog` 库实现，支持多种日志级别和输出格式。

## 🎛️ 日志配置

### 环境变量配置

```bash
# 设置日志级别
export LOG_LEVEL=debug     # debug, info, warn, error
export LOG_FORMAT=json     # json, console
export LOG_OUTPUT=file     # console, file, both

# 文件输出配置
export LOG_FILE=chatlog.log
export LOG_MAX_SIZE=100    # MB
export LOG_MAX_AGE=7       # days
export LOG_MAX_BACKUPS=3   # 备份文件数量
```

### 命令行参数

```bash
# 启用调试模式
chatlog --debug

# 启动服务时开启调试日志
chatlog server --debug

# 指定日志文件
chatlog server --log-file /path/to/chatlog.log
```

### 配置文件设置

```json
{
  "log": {
    "level": "info",
    "format": "console",
    "output": "both",
    "file": {
      "path": "./logs/chatlog.log",
      "max_size": 100,
      "max_age": 7,
      "max_backups": 3
    }
  }
}
```

## 📊 日志级别

### 级别说明

| 级别 | 代码 | 说明 | 使用场景 |
|------|------|------|----------|
| DEBUG | 0 | 详细调试信息 | 开发调试、问题排查 |
| INFO | 1 | 一般信息 | 正常操作记录 |
| WARN | 2 | 警告信息 | 潜在问题提醒 |
| ERROR | 3 | 错误信息 | 错误和异常情况 |
| FATAL | 4 | 致命错误 | 程序无法继续运行 |

### 级别设置示例

```bash
# 仅显示错误和致命错误
export LOG_LEVEL=error

# 显示所有信息 (开发模式)
export LOG_LEVEL=debug

# 生产环境建议设置
export LOG_LEVEL=info
```

## 🏗️ 日志结构

### 标准日志字段

```json
{
  "level": "info",
  "time": "2024-01-01T12:00:00Z",
  "message": "HTTP server started",
  "component": "server",
  "addr": "127.0.0.1:5030",
  "pid": 12345
}
```

### 字段说明

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `level` | string | 日志级别 | "info", "error" |
| `time` | string | 时间戳 (RFC3339) | "2024-01-01T12:00:00Z" |
| `message` | string | 日志消息 | "HTTP server started" |
| `component` | string | 组件名称 | "server", "wechat", "database" |
| `error` | string | 错误信息 | "connection refused" |
| `caller` | string | 调用位置 | "main.go:123" |

## 🔍 组件分类

### 核心组件日志

#### 1. 应用启动 (component: "app")

```json
// 应用启动
{
  "level": "info",
  "component": "app",
  "message": "Chatlog starting",
  "version": "v1.0.0",
  "platform": "windows"
}

// 配置加载
{
  "level": "info", 
  "component": "app",
  "message": "Configuration loaded",
  "config_path": "/path/to/config.json"
}

// 应用关闭
{
  "level": "info",
  "component": "app", 
  "message": "Chatlog shutting down",
  "uptime": "1h30m45s"
}
```

#### 2. 微信集成 (component: "wechat")

```json
// 进程检测
{
  "level": "info",
  "component": "wechat",
  "message": "WeChat process detected",
  "pid": 12345,
  "version": "3.9.5.81",
  "platform": "windows"
}

// 密钥获取
{
  "level": "info",
  "component": "wechat",
  "message": "Data key extracted successfully", 
  "key_length": 32
}

// 解密操作
{
  "level": "info",
  "component": "wechat",
  "message": "Database decryption completed",
  "files_processed": 15,
  "duration": "2.5s"
}
```

#### 3. HTTP 服务器 (component: "server")

```json
// 服务启动
{
  "level": "info",
  "component": "server", 
  "message": "HTTP server started",
  "addr": "127.0.0.1:5030"
}

// 请求日志
{
  "level": "info",
  "component": "server",
  "message": "HTTP request",
  "method": "GET",
  "path": "/api/v1/chatlog",
  "remote_addr": "127.0.0.1:54321",
  "user_agent": "curl/7.68.0",
  "duration": "45ms",
  "status": 200
}

// API 错误
{
  "level": "error",
  "component": "server",
  "message": "API request failed",
  "method": "GET", 
  "path": "/api/v1/chatlog",
  "error": "invalid parameter: time format",
  "status": 400
}
```

#### 4. 数据库操作 (component: "database")

```json
// 数据库连接
{
  "level": "info",
  "component": "database",
  "message": "Database connected",
  "db_path": "/path/to/MSG0.db",
  "connection_count": 5
}

// 查询执行
{
  "level": "debug",
  "component": "database", 
  "message": "SQL query executed",
  "query": "SELECT * FROM MSG WHERE StrTalker = ? LIMIT 100",
  "duration": "12ms",
  "rows_affected": 95
}

// 数据库错误
{
  "level": "error",
  "component": "database",
  "message": "Database query failed", 
  "query": "SELECT * FROM MSG",
  "error": "no such table: MSG",
  "db_path": "/path/to/database.db"
}
```

#### 5. MCP 协议 (component: "mcp")

```json
// MCP 服务启动
{
  "level": "info",
  "component": "mcp",
  "message": "MCP server initialized",
  "transport": "sse",
  "endpoint": "/sse"
}

// 工具调用
{
  "level": "info", 
  "component": "mcp",
  "message": "Tool called",
  "tool_name": "get_chatlog",
  "session_id": "sess_123",
  "parameters": {
    "talker": "wxid_xxx",
    "limit": 100
  }
}

// SSE 连接
{
  "level": "info",
  "component": "mcp",
  "message": "SSE client connected", 
  "client_id": "client_456",
  "remote_addr": "127.0.0.1:54321"
}
```

### 功能模块日志

#### 1. 语音处理 (component: "voice")

```json
// 语音转换开始
{
  "level": "info",
  "component": "voice",
  "message": "Starting voice conversion",
  "voice_id": "voice_123",
  "input_format": "silk",
  "output_format": "mp3"
}

// SILK 解码
{
  "level": "debug",
  "component": "voice", 
  "message": "SILK decoding completed",
  "input_size": 8192,
  "output_size": 49152,
  "duration": "15ms"
}

// MP3 编码
{
  "level": "debug",
  "component": "voice",
  "message": "MP3 encoding completed",
  "input_size": 49152, 
  "output_size": 3072,
  "bitrate": 128,
  "duration": "8ms"
}

// 语音处理错误
{
  "level": "error",
  "component": "voice",
  "message": "Voice conversion failed",
  "voice_id": "voice_123", 
  "error": "invalid SILK format",
  "input_size": 0
}
```

#### 2. 图片处理 (component: "image")

```json
// 图片解密
{
  "level": "info",
  "component": "image",
  "message": "Image decryption completed",
  "image_id": "img_456", 
  "original_size": 65536,
  "decrypted_size": 65536,
  "format": "jpg"
}

// 解密失败
{
  "level": "error",
  "component": "image",
  "message": "Image decryption failed",
  "image_id": "img_456",
  "error": "invalid encryption key",
  "file_path": "/path/to/image.dat"
}
```

#### 3. 文件监控 (component: "monitor")

```json
// 文件变化检测
{
  "level": "info",
  "component": "monitor", 
  "message": "File change detected",
  "event": "create",
  "file_path": "/path/to/MSG1.db",
  "file_size": 1048576
}

// 自动解密触发
{
  "level": "info",
  "component": "monitor",
  "message": "Auto-decrypt triggered",
  "trigger_file": "MSG1.db",
  "files_to_process": 3
}
```

## 🎯 日志过滤和搜索

### 按组件过滤

```bash
# 只查看服务器相关日志
grep '"component":"server"' chatlog.log

# 只查看错误日志
grep '"level":"error"' chatlog.log

# 查看特定时间范围
grep '2024-01-01T1[0-5]:' chatlog.log
```

### 使用 jq 处理 JSON 日志

```bash
# 安装 jq
# macOS: brew install jq
# Ubuntu: sudo apt install jq

# 查看所有错误消息
cat chatlog.log | jq 'select(.level=="error") | .message'

# 统计各组件的日志数量
cat chatlog.log | jq -r '.component' | sort | uniq -c

# 查看 HTTP 请求统计
cat chatlog.log | jq 'select(.component=="server" and .method) | {method, path, status, duration}'
```

### 实时日志监控

```bash
# 实时查看日志
tail -f chatlog.log

# 实时查看并格式化 JSON 日志
tail -f chatlog.log | jq '.'

# 实时过滤错误日志
tail -f chatlog.log | jq 'select(.level=="error")'
```

## 🐛 调试模式

### 启用详细日志

```bash
# 全局调试模式
export LOG_LEVEL=debug
chatlog server

# 临时调试模式  
chatlog server --debug

# 组件特定调试
export LOG_COMPONENTS="wechat,database,voice"
export LOG_LEVEL=debug
chatlog server
```

### 调试日志示例

```json
// 函数调用跟踪
{
  "level": "debug",
  "component": "wechat", 
  "message": "Entering function: extractDataKey",
  "caller": "key/extractor.go:45",
  "process_id": 12345
}

// 变量状态
{
  "level": "debug",
  "component": "database",
  "message": "Query parameters", 
  "talker": "wxid_xxx",
  "time_range": "2024-01-01~2024-01-31",
  "limit": 100,
  "offset": 0
}

// 性能指标
{
  "level": "debug", 
  "component": "voice",
  "message": "Performance metrics",
  "operation": "silk_decode",
  "duration": "15ms",
  "memory_used": "2.5MB",
  "cpu_time": "12ms"
}
```

## 📈 日志分析

### 性能分析

```bash
# HTTP 请求耗时分析
cat chatlog.log | jq 'select(.component=="server" and .duration) | {path, duration}' | \
grep -o '"duration":"[^"]*"' | cut -d'"' -f4 | sort -n

# 数据库查询性能
cat chatlog.log | jq 'select(.component=="database" and .duration) | {query, duration}'

# 语音转换耗时统计
cat chatlog.log | jq 'select(.component=="voice" and .message=="Voice conversion completed") | .duration'
```

### 错误统计

```bash
# 错误类型统计
cat chatlog.log | jq 'select(.level=="error") | .component' | sort | uniq -c

# API 错误统计
cat chatlog.log | jq 'select(.level=="error" and .component=="server") | {path, error, status}' 

# 数据库错误分析
cat chatlog.log | jq 'select(.level=="error" and .component=="database") | {error, query}'
```

### 使用量分析

```bash
# API 调用频率
cat chatlog.log | jq 'select(.component=="server" and .method) | .path' | sort | uniq -c | sort -nr

# 用户活跃度 (基于 IP)
cat chatlog.log | jq 'select(.component=="server") | .remote_addr' | cut -d':' -f1 | sort | uniq -c

# 功能使用统计
cat chatlog.log | jq 'select(.component=="mcp") | .tool_name' | sort | uniq -c
```

## 🔧 日志配置最佳实践

### 生产环境配置

```json
{
  "log": {
    "level": "info",
    "format": "json",
    "output": "file", 
    "file": {
      "path": "/var/log/chatlog/chatlog.log",
      "max_size": 100,
      "max_age": 30,
      "max_backups": 10,
      "compress": true
    }
  }
}
```

### 开发环境配置

```json
{
  "log": {
    "level": "debug",
    "format": "console",
    "output": "console",
    "show_caller": true,
    "show_stack_trace": true
  }
}
```

### 容器化部署配置

```dockerfile
# Dockerfile 中的日志配置
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json
ENV LOG_OUTPUT=console

# 将日志输出到 stdout/stderr 供容器运行时收集
```

## 🚨 日志轮转和清理

### 自动轮转配置

```bash
# 使用 logrotate (Linux)
cat > /etc/logrotate.d/chatlog << EOF
/var/log/chatlog/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 chatlog chatlog
    postrotate
        killall -USR1 chatlog || true
    endscript
}
EOF
```

### 手动清理脚本

```bash
#!/bin/bash
# cleanup-logs.sh

LOG_DIR="/var/log/chatlog"
MAX_AGE=30

# 删除超过 30 天的日志文件
find $LOG_DIR -name "*.log*" -mtime +$MAX_AGE -delete

# 压缩超过 7 天的日志文件
find $LOG_DIR -name "*.log" -mtime +7 ! -name "*.gz" -exec gzip {} \;

echo "Log cleanup completed"
```

## 🔍 故障排查指南

### 常见日志错误

#### 1. 数据库连接失败

```json
{
  "level": "error",
  "component": "database", 
  "message": "Failed to connect to database",
  "error": "unable to open database file",
  "db_path": "/path/to/MSG0.db"
}
```

**排查步骤:**
1. 检查文件是否存在和权限
2. 确认微信进程是否关闭
3. 验证解密密钥是否正确

#### 2. HTTP 服务启动失败

```json
{
  "level": "error",
  "component": "server",
  "message": "Failed to start HTTP server", 
  "error": "bind: address already in use",
  "addr": "127.0.0.1:5030"
}
```

**排查步骤:**
1. 检查端口是否被占用: `netstat -ano | findstr :5030`
2. 更换端口或终止占用进程
3. 检查防火墙设置

#### 3. 密钥获取失败

```json
{
  "level": "error",
  "component": "wechat",
  "message": "Failed to extract data key",
  "error": "access denied", 
  "pid": 12345
}
```

**排查步骤:**
1. 确认以管理员权限运行
2. 检查安全软件是否阻拦
3. 验证微信版本是否支持

## 💡 日志优化建议

### 性能优化

1. **异步写入**: 使用缓冲区异步写入日志
2. **级别控制**: 生产环境避免使用 debug 级别
3. **字段精简**: 只记录必要的字段信息
4. **批量写入**: 批量写入多条日志记录

### 存储优化

1. **压缩存储**: 启用日志文件压缩
2. **定期清理**: 自动删除过期日志文件
3. **分级存储**: 热数据和冷数据分别存储
4. **外部收集**: 使用 ELK、Fluentd 等工具收集日志

### 安全考虑

1. **敏感信息**: 避免在日志中记录密码、密钥等敏感信息
2. **访问控制**: 限制日志文件的访问权限
3. **数据脱敏**: 对用户相关信息进行脱敏处理
4. **审计跟踪**: 记录关键操作的审计信息