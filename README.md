<div align="center">

# Chatlog

![chatlog](https://socialify.git.ci/sjzar/chatlog/image?font=Rokkitt&name=1&pattern=Diagonal+Stripes&theme=Auto)

_聊天记录工具，帮助大家轻松使用自己的聊天数据_

[![Go Report Card](https://goreportcard.com/badge/github.com/sjzar/chatlog)](https://goreportcard.com/report/github.com/sjzar/chatlog)
[![GoDoc](https://godoc.org/github.com/sjzar/chatlog?status.svg)](https://godoc.org/github.com/sjzar/chatlog)
[![GitHub release](https://img.shields.io/github/release/sjzar/chatlog.svg)](https://github.com/sjzar/chatlog/releases)
[![GitHub license](https://img.shields.io/github/license/sjzar/chatlog.svg)](https://github.com/sjzar/chatlog/blob/main/LICENSE)

</div>

![chatlog](https://github.com/user-attachments/assets/746717b8-9b39-4a45-97f3-f0ae8fc5a344)

## Feature

- 从本地数据库文件获取聊天数据
- 支持 Windows / macOS 系统
- 支持微信 3.x / 4.0 版本
- 提供 Terminal UI 界面 & 命令行工具
- 提供 HTTP API 服务，支持查询聊天记录、联系人、群聊、最近会话等信息
- 支持 MCP SSE 协议，可与支持 MCP 的 AI 助手无缝集成


## TODO

- 支持多媒体数据
- 聊天数据全文索引
- 聊天数据统计 & Dashboard


## Install

### 从源码安装

```bash
go install github.com/sjzar/chatlog@latest
```

### 下载预编译版本

访问 [Releases](https://github.com/sjzar/chatlog/releases) 页面下载适合您系统的预编译版本。

## Quick Start

### 操作流程

1. 下载并安装微信客户端

2. 手机微信上操作 `我 - 设置 - 通用 - 聊天记录迁移与备份 - 迁移 - 迁移到电脑`，这一步的目的是将手机中的聊天记录传输到电脑上。可以放心操作，不会影响到手机上的聊天记录。

3. 下载 `chatlog` 预编译版本或从源码安装，推荐使用 go 进行安装。

4. 运行 `chatlog`，按照提示进行操作，解密数据并开启 HTTP 服务后，即可通过浏览器或 AI 助手访问聊天记录。

### macOS 版本提示

1. macOS 用户在获取密钥前，需要确认已经关闭 SIP 并安装 Xcode Command Line Tools。由于 macOS 的安全机制，在正常情况在无法读取微信进程的内存数据，所以需要临时关闭 SIP。关闭 SIP 的方法：

```shell
# 1. 进入恢复模式
  Apple Intel Mac: 关机后，按住 Command + R 键开机，直到出现苹果标志和进度条。
  Apple Silicon Mac: 关机后，按住开机键不松开，直到出现苹果标志和进度条。
# 2. 打开终端
  选项 - 实用工具 - 终端
# 3. 关闭 SIP
  输入以下命令关闭 SIP：
  csrutil disable
# 4. 重启系统
```

2. 目前的 macOS 版本方案依赖 `lldb` 工具，所以需要安装 Xcode Command Line Tools。

```shell
# 在 terminal 执行以下命令安装 Xcode Command Line Tools：
xcode-select --install
```

3. 仅获取数据密钥步骤需要关闭 SIP；获取数据密钥后即可重新打开 SIP，不影响解密数据和 HTTP 服务的运行。

4. 如果是 Apple Silicon 芯片的 mac 用户，请检查 微信、chatlog、terminal 均不要运行在 Rosetta 模式下运行，否则可能无法获取密钥。

### Terminal UI 模式

1. 启动程序：

```bash
./chatlog
```

2. 使用界面操作：
   - 使用方向键导航菜单
   - 按 `Enter` 选择菜单项
   - 按 `Esc` 返回上一级菜单
   - 按 `Ctrl+C` 退出程序

### 命令行模式

获取微信数据密钥：

```bash
./chatlog key
```

解密数据库文件：

```bash
./chatlog decrypt
```

## HTTP API

启动 HTTP 服务后，可以通过以下 API 访问数据：

### 聊天记录

```
GET /api/v1/chatlog?time=2023-01-01&talker=wxid_xxx&limit=100&offset=0&format=json
```

参数说明：
- `time`: 时间范围，格式为 `YYYY-MM-DD` 或 `YYYY-MM-DD~YYYY-MM-DD`
- `talker`: 聊天对象的 ID，不知道 ID 的话也可以尝试备注名、昵称、群聊 ID等
- `limit`: 返回记录数量限制
- `offset`: 分页偏移量
- `format`: 输出格式，支持 `json`、`csv` 或纯文本

### 联系人列表

```
GET /api/v1/contact
```

### 群聊列表

```
GET /api/v1/chatroom
```

### 会话列表

```
GET /api/v1/session
```

## MCP

支持 MCP SSE 协议，启动 HTTP 服务后，通过 SSE Endpoint 访问服务：

```
GET /sse
```

提供了 4 个 tool 用于与 AI 助手集成：
- `chatlog`: 查询聊天记录
- `query_contact`: 查询联系人
- `query_chat_room`: 查询群聊
- `query_recent_chat`: 查询最近会话

### 示例

以 [ChatWise](https://chatwise.app/) 工具为例，在 `设置 - 工具` 下新建工具，类型为 `sse`，ID 为 `chatlog`，URL 为 `http://127.0.0.1:5030/sse`，勾选自动执行工具，即可使用。

部分 AI 聊天工具暂时不支持 MCP SSE 协议，可以通过 [`mcp-proxy`](https://github.com/sparfenyuk/mcp-proxy) 工具转发请求，以 [Claude Desktop](https://claude.ai/download) 为例，在安装好 `mcp-proxy` 后，将 `mcp-proxy` 配置到 `Claude Desktop` 的 `config.json` 文件中，即可使用：

```json
{
  "mcpServers": {
    "mcp-proxy": {
      "command": "/Users/sarv/.local/bin/mcp-proxy",
      "args": [
        "http://localhost:5030/sse"
      ],
      "env": {}
    }
  },
  "globalShortcut": ""
}
```

## License

`chatlog` 是在 Apache-2.0 许可下的开源软件。

## Thanks

- [@0xlane](https://github.com/0xlane) 的 [wechat-dump-rs](https://github.com/0xlane/wechat-dump-rs) 项目
- [@xaoyaoo](https://github.com/xaoyaoo) 的 [PyWxDump](https://github.com/xaoyaoo/PyWxDump) 项目
- [Anthropic](https://www.anthropic.com/) 的 [MCP]((https://github.com/modelcontextprotocol) ) 协议
- 各个 Go 开源库的贡献者们