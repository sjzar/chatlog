# Chatlog 开发指南

## 🚀 开发环境搭建

### 环境要求

#### 必需软件
- **Go**: 1.24.0 或更高版本
- **Git**: 版本控制
- **Make**: 构建工具 (可选)

#### 平台特定要求

**Windows:**
- Windows 10/11
- Microsoft C++ Build Tools (用于 CGO)
- Windows Terminal (推荐，更好的终端体验)

**macOS:**
- macOS 10.15 或更高版本
- Xcode Command Line Tools
- 临时关闭 SIP (仅在获取密钥时需要)

### 克隆项目

```bash
git clone https://github.com/sjzar/chatlog.git
cd chatlog
```

### 安装依赖

```bash
go mod download
go mod verify
```

### 构建项目

```bash
# 开发构建
go build -o chatlog main.go

# 或使用 make (如果有 Makefile)
make build

# 运行
./chatlog
```

## 📁 项目结构详解

### 目录结构

```
chatlog/
├── cmd/chatlog/              # 命令行入口
│   ├── root.go              # 根命令，TUI 入口
│   ├── cmd_server.go        # HTTP 服务命令
│   ├── cmd_key.go          # 密钥获取命令
│   ├── cmd_decrypt.go      # 解密命令
│   ├── cmd_version.go      # 版本命令
│   └── log.go              # 日志配置
│
├── internal/               # 内部包，不对外暴露
│   ├── chatlog/           # 应用核心逻辑
│   │   ├── app.go         # TUI 应用主类
│   │   ├── manager.go     # 业务管理器
│   │   ├── conf/          # 配置管理
│   │   ├── ctx/           # 应用上下文
│   │   ├── database/      # 数据库服务
│   │   ├── http/          # HTTP 服务
│   │   ├── webhook/       # Webhook 功能
│   │   └── wechat/        # 微信服务
│   │
│   ├── errors/            # 错误处理
│   │   ├── errors.go      # 通用错误
│   │   ├── http_errors.go # HTTP 错误
│   │   ├── mcp.go         # MCP 错误
│   │   └── ...           # 其他错误类型
│   │
│   ├── mcp/              # MCP 协议实现
│   │   ├── mcp.go        # MCP 服务器
│   │   ├── sse.go        # SSE 支持
│   │   ├── stdio.go      # 标准IO支持
│   │   ├── tool.go       # 工具定义
│   │   └── ...
│   │
│   ├── model/            # 数据模型
│   │   ├── message.go    # 消息模型
│   │   ├── contact.go    # 联系人模型
│   │   ├── chatroom.go   # 群聊模型
│   │   ├── session.go    # 会话模型
│   │   ├── media.go      # 媒体模型
│   │   └── wxproto/      # Protocol Buffers
│   │
│   ├── ui/               # Terminal UI 组件
│   │   ├── footer/       # 页脚组件
│   │   ├── form/         # 表单组件
│   │   ├── help/         # 帮助组件
│   │   ├── infobar/      # 信息栏组件
│   │   ├── menu/         # 菜单组件
│   │   └── style/        # 样式定义
│   │
│   ├── wechat/           # 微信集成
│   │   ├── wechat.go     # 微信管理器
│   │   ├── manager.go    # 实例管理
│   │   ├── decrypt/      # 解密功能
│   │   ├── key/          # 密钥提取
│   │   ├── model/        # 微信模型
│   │   └── process/      # 进程检测
│   │
│   └── wechatdb/         # 数据库操作
│       ├── wechatdb.go   # 数据库管理器
│       ├── repository/   # 仓库模式
│       └── datasource/   # 数据源抽象
│
├── pkg/                  # 公共包，可对外暴露
│   ├── appver/          # 应用版本检测
│   ├── config/          # 配置管理
│   ├── filecopy/        # 文件复制
│   ├── filemonitor/     # 文件监控
│   ├── util/            # 工具函数
│   └── version/         # 版本信息
│
├── docs/                # 项目文档
├── script/              # 构建脚本
├── go.mod              # Go 模块定义
├── go.sum              # 依赖校验和
├── main.go             # 程序入口
├── Makefile            # 构建配置
├── Dockerfile          # Docker 配置
└── README.md           # 项目说明
```

## 🔧 开发规范

### 代码风格

#### Go 代码规范
```go
// 包声明
package chatlog

// 导入顺序：标准库、第三方库、内部包
import (
    "context"
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
    
    "github.com/sjzar/chatlog/internal/model"
)

// 常量定义
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

// 结构体定义
type Service struct {
    db     *sql.DB
    logger *zerolog.Logger
}

// 构造函数
func NewService(db *sql.DB) *Service {
    return &Service{
        db:     db,
        logger: log.With().Str("component", "service").Logger(),
    }
}

// 方法定义
func (s *Service) ProcessMessage(ctx context.Context, msg *model.Message) error {
    s.logger.Debug().
        Str("msg_id", msg.ID).
        Msg("processing message")
    
    // 具体实现
    return nil
}
```

#### 错误处理
```go
// 自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// 错误包装
func (s *Service) GetMessage(id string) (*model.Message, error) {
    if id == "" {
        return nil, &ValidationError{
            Field:   "id",
            Message: "message id cannot be empty",
        }
    }
    
    msg, err := s.db.GetMessage(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get message %s: %w", id, err)
    }
    
    return msg, nil
}
```

#### 日志记录
```go
// 使用结构化日志
log.Info().
    Str("component", "wechat").
    Str("platform", "windows").
    Int("pid", 1234).
    Msg("detected wechat process")

// 错误日志
log.Error().
    Err(err).
    Str("file", filename).
    Msg("failed to decrypt file")

// 调试日志
log.Debug().
    Interface("config", cfg).
    Msg("loaded configuration")
```

### 项目约定

#### 目录命名
- 使用小写字母和下划线
- 内部包放在 `internal/` 下
- 公共包放在 `pkg/` 下
- 平台特定代码使用构建标签

#### 文件命名
```go
// 平台特定文件
key_windows.go    // Windows 实现
key_darwin.go     // macOS 实现
key_others.go     // 其他平台默认实现

// 构建标签
//go:build windows
// +build windows

package key
```

#### 接口设计
```go
// 定义接口
type MessageRepository interface {
    GetByID(ctx context.Context, id string) (*model.Message, error)
    GetByTalker(ctx context.Context, talker string, opts *QueryOptions) ([]*model.Message, error)
    Create(ctx context.Context, msg *model.Message) error
}

// 实现接口
type sqliteMessageRepository struct {
    db *sql.DB
}

func (r *sqliteMessageRepository) GetByID(ctx context.Context, id string) (*model.Message, error) {
    // 实现
    return nil, nil
}
```

## 🧪 测试

### 单元测试

```go
// message_test.go
package model

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMessage_IsFromSelf(t *testing.T) {
    tests := []struct {
        name     string
        isSender int
        want     bool
    }{
        {"is sender", 1, true},
        {"not sender", 0, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msg := &Message{IsSender: tt.isSender}
            assert.Equal(t, tt.want, msg.IsFromSelf())
        })
    }
}

func TestMessage_GetDisplayTime(t *testing.T) {
    msg := &Message{
        CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
    }
    
    result := msg.GetDisplayTime()
    require.NotEmpty(t, result)
    assert.Contains(t, result, "2024")
}
```

### 集成测试

```go
// integration_test.go
//go:build integration
// +build integration

package chatlog

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
    suite.Suite
    manager *Manager
}

func (s *IntegrationTestSuite) SetupSuite() {
    // 设置测试环境
    s.manager = NewManager()
}

func (s *IntegrationTestSuite) TestDecryptDatabase() {
    ctx := context.Background()
    err := s.manager.DecryptDBFiles()
    s.Require().NoError(err)
}

func TestIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(IntegrationTestSuite))
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行单元测试
go test ./internal/model

# 运行集成测试
go test -tags=integration ./...

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🔄 开发流程

### 分支策略

```bash
# 主要分支
main        # 主分支，稳定版本
develop     # 开发分支
feature/*   # 功能分支
bugfix/*    # 修复分支
release/*   # 发布分支
```

### 提交规范

```bash
# 提交消息格式
<type>(<scope>): <subject>

<body>

<footer>

# 类型
feat:     新功能
fix:      修复bug
docs:     文档更新
style:    代码格式调整
refactor: 重构
test:     测试相关
chore:    构建/工具链更新

# 示例
feat(mcp): add voice message support

Add support for voice message processing in MCP protocol:
- Implement SILK to MP3 conversion
- Add voice data streaming
- Update API documentation

Closes #123
```

### 开发工作流

1. **创建功能分支**
```bash
git checkout -b feature/voice-enhancement
```

2. **开发和测试**
```bash
# 开发代码
go build
go test ./...

# 提交更改
git add .
git commit -m "feat(voice): enhance voice processing"
```

3. **推送和创建 PR**
```bash
git push origin feature/voice-enhancement
# 在 GitHub 创建 Pull Request
```

4. **代码审查和合并**
```bash
# 审查通过后合并到 develop
git checkout develop
git merge feature/voice-enhancement
```

## 🔧 调试技巧

### 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug main.go

# 在代码中设置断点
(dlv) break main.main
(dlv) break internal/wechat/manager.go:123

# 运行程序
(dlv) continue

# 检查变量
(dlv) print variable_name
(dlv) locals
```

### VS Code 调试配置

```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Chatlog",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/main.go",
            "args": ["server", "--debug"],
            "env": {
                "LOG_LEVEL": "debug"
            }
        }
    ]
}
```

### 日志调试

```go
// 添加调试日志
log.Debug().
    Str("function", "ProcessMessage").
    Interface("message", msg).
    Msg("processing message")

// 使用条件日志
if log.Debug().Enabled() {
    log.Debug().
        Str("expensive_operation", expensiveDebugInfo()).
        Msg("debug info")
}
```

## 🚀 部署

### 本地构建

```bash
# 构建当前平台
go build -o chatlog main.go

# 交叉编译
GOOS=windows GOARCH=amd64 go build -o chatlog.exe main.go
GOOS=darwin GOARCH=amd64 go build -o chatlog-darwin main.go
GOOS=linux GOARCH=amd64 go build -o chatlog-linux main.go
```

### Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o chatlog main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/chatlog .
CMD ["./chatlog", "server"]
```

```bash
# 构建镜像
docker build -t chatlog .

# 运行容器
docker run -p 5030:5030 -v /path/to/data:/data chatlog
```

## 📊 性能分析

### CPU Profiling

```go
// 在代码中添加 pprof
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```bash
# 收集 CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 收集内存 profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

### 基准测试

```go
// benchmark_test.go
func BenchmarkMessageProcess(b *testing.B) {
    msg := &model.Message{
        Content: "test message",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        processMessage(msg)
    }
}
```

```bash
# 运行基准测试
go test -bench=. -benchmem
```

## 🤝 贡献指南

### 贡献流程

1. **Fork 项目**
2. **创建功能分支**
3. **编写代码和测试**
4. **提交 Pull Request**
5. **代码审查**
6. **合并代码**

### PR 清单

- [ ] 代码通过所有测试
- [ ] 添加了必要的单元测试
- [ ] 更新了相关文档
- [ ] 遵循代码规范
- [ ] 提交消息符合规范
- [ ] 没有合并冲突

### 代码审查要点

1. **功能正确性**: 代码是否实现了预期功能
2. **安全性**: 是否存在安全漏洞
3. **性能**: 是否有性能问题
4. **可维护性**: 代码是否易于理解和维护
5. **测试覆盖**: 是否有充分的测试覆盖

## 📚 学习资源

### Go 语言
- [Go 官方文档](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 项目相关
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [TView Terminal UI](https://github.com/rivo/tview)
- [Cobra CLI](https://cobra.dev/)
- [Zerolog Logging](https://github.com/rs/zerolog)

### 工具和资源
- [Go Tools](https://golang.org/cmd/)
- [Delve Debugger](https://github.com/go-delve/delve)
- [Go Modules](https://golang.org/ref/mod)

## ❓ 常见问题

### Q: 如何添加新的微信版本支持？

A: 参考现有版本实现，主要步骤：
1. 在 `internal/model/` 添加版本特定的数据模型
2. 在 `internal/wechatdb/datasource/` 添加数据源实现
3. 在版本检测逻辑中添加新版本识别
4. 更新相关测试

### Q: 如何添加新的 API 接口？

A: 在 `internal/chatlog/http/` 中：
1. 在 `route.go` 中添加路由定义
2. 在 `service.go` 中添加处理函数
3. 更新 API 文档
4. 添加相应测试

### Q: 如何处理跨平台兼容性？

A: 使用构建标签和条件编译：
```go
//go:build windows
// +build windows

package platform

// Windows 特定实现
```

### Q: 如何优化数据库查询性能？

A: 主要方法：
1. 添加适当的数据库索引
2. 使用分页查询避免大结果集
3. 实现查询结果缓存
4. 优化 SQL 语句结构