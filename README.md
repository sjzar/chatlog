<div align="center">

![chatlog](https://github.com/user-attachments/assets/e085d3a2-e009-4463-b2fd-8bd7df2b50c3)

_聊天记录工具，帮助大家轻松使用自己的聊天数据_

[![ImgMCP](https://cdn.imgmcp.com/imgmcp-logo-small.png)](https://imgmcp.com)

[![Go Report Card](https://goreportcard.com/badge/github.com/sjzar/chatlog)](https://goreportcard.com/report/github.com/sjzar/chatlog)
[![GoDoc](https://godoc.org/github.com/sjzar/chatlog?status.svg)](https://godoc.org/github.com/sjzar/chatlog)
[![GitHub release](https://img.shields.io/github/release/sjzar/chatlog.svg)](https://github.com/sjzar/chatlog/releases)
[![GitHub license](https://img.shields.io/github/license/sjzar/chatlog.svg)](https://github.com/sjzar/chatlog/blob/main/LICENSE)


</div>

## Feature

- 从本地数据库文件中获取聊天数据
- 支持 Windows / macOS 系统，兼容微信 3.x / 4.x 版本
- 支持获取数据与图片密钥 (Windows < 4.0.3.36 / macOS < 4.0.3.80)
- 支持图片、语音等多媒体数据解密，支持 wxgf 格式解析
- 支持自动解密数据库，并提供新消息 Webhook 回调
- 提供 Terminal UI 界面，同时支持命令行工具和 Docker 镜像部署
- 提供 HTTP API 服务，可轻松查询聊天记录、联系人、群聊、最近会话等信息
- 支持 MCP Streamable HTTP 协议，可与 AI 助手无缝集成
- 支持多账号管理，可在不同账号间切换

## Quick Start

### 基本步骤

1. **安装 Chatlog**：[下载预编译版本](#下载预编译版本) 或 [使用 Go 安装](#从源码安装)
2. **运行程序**：执行 `chatlog` 启动 Terminal UI 界面
3. **解密数据**：选择 `解密数据` 菜单项
4. **开启 HTTP 服务**：选择 `开启 HTTP 服务` 菜单项
5. **访问数据**：通过 [HTTP API](#http-api) 或 [MCP 集成](#mcp-集成) 访问聊天记录

> 💡 **提示**: 如果电脑端微信聊天记录不全，可以[从手机端迁移数据](#从手机迁移聊天记录)  

### 常见问题快速解决

- **macOS 用户**：获取密钥前需[临时关闭 SIP](#macos-版本说明)
- **Windows 用户**：遇到界面显示问题请[使用 Windows Terminal](#windows-版本说明)
- **集成 AI 助手**：查看 [MCP 集成指南](#mcp-集成)
- **无法获取密钥**：查看 [FAQ](https://github.com/sjzar/chatlog/issues/197)

## 安装指南

### 从源码安装

```bash
go install github.com/sjzar/chatlog@latest
```

> 💡 **提示**: 部分功能有 cgo 依赖，编译前需确认本地有 C 编译环境。

### 下载预编译版本

访问 [Releases](https://github.com/sjzar/chatlog/releases) 页面下载适合您系统的预编译版本。

## 使用指南

### Terminal UI 模式

最简单的使用方式是通过 Terminal UI 界面操作：

```bash
chatlog
```

操作方法：
- 使用 `↑` `↓` 键选择菜单项
- 按 `Enter` 确认选择
- 按 `Esc` 返回上级菜单
- 按 `Ctrl+C` 退出程序

### 命令行模式

对于熟悉命令行的用户，可以直接使用以下命令：

```bash
# 获取微信数据密钥
chatlog key

# 解密数据库文件
chatlog decrypt

# 启动 HTTP 服务
chatlog server
```

### Docker 部署

由于 Docker 部署时，程序运行环境与宿主机隔离，所以不支持获取密钥等操作，需要提前获取密钥数据。

一般用于 NAS 等设备部署，详细指南可参考 [Docker 部署指南](docs/docker.md)

**0. 获取密钥信息**

```shell
# 从本机运行 chatlog 获取密钥信息
$ chatlog key
Data Key: [c0163e***ac3dc6]
Image Key: [38636***653361]
```

**1. 拉取镜像**

chatlog 提供了两个镜像源：

**Docker Hub**:
```shell
docker pull sjzar/chatlog:latest
```

**GitHub Container Registry (ghcr)**:
```shell
docker pull ghcr.io/sjzar/chatlog:latest
```

> 💡 **镜像地址**: 
> - Docker Hub: https://hub.docker.com/r/sjzar/chatlog
> - GitHub Container Registry: https://ghcr.io/sjzar/chatlog

**2. 运行容器**

```shell
$ docker run -d \
  --name chatlog \
  -p 5030:5030 \
  -v /path/to/your/wechat/data:/app/data \
  sjzar/chatlog:latest
```

### 从手机迁移聊天记录

如果电脑端微信聊天记录不全，可以从手机端迁移数据：

1. 打开手机微信，进入 `我 - 设置 - 通用 - 聊天记录迁移与备份`
2. 选择 `迁移 - 迁移到电脑`，按照提示操作
3. 完成迁移后，重新运行 `chatlog` 获取密钥并解密数据

> 此操作不会影响手机上的聊天记录，只是将数据复制到电脑端

## 平台特定说明

### Windows 版本说明

如遇到界面显示异常（如花屏、乱码等），请使用 [Windows Terminal](https://github.com/microsoft/terminal) 运行程序

### macOS 版本说明

macOS 用户在获取密钥前需要临时关闭 SIP（系统完整性保护）：

1. **关闭 SIP**：
   ```shell
   # 进入恢复模式
   # Intel Mac: 重启时按住 Command + R
   # Apple Silicon: 重启时长按电源键
   
   # 在恢复模式中打开终端并执行
   csrutil disable
   
   # 重启系统
   ```

2. **安装必要工具**：
   ```shell
   # 安装 Xcode Command Line Tools
   xcode-select --install
   ```

3. **获取密钥后**：可以重新启用 SIP（`csrutil enable`），不影响后续使用

> Apple Silicon 用户注意：确保微信、chatlog 和终端都不在 Rosetta 模式下运行

## HTTP API

启动 HTTP 服务后（默认地址 `http://127.0.0.1:5030`），可通过以下 API 访问数据：

### 聊天记录查询

```
GET /api/v1/chatlog?time=2023-01-01&talker=wxid_xxx
```

参数说明：
- `time`: 时间范围，格式为 `YYYY-MM-DD` 或 `YYYY-MM-DD~YYYY-MM-DD`
- `talker`: 聊天对象标识（支持 wxid、群聊 ID、备注名、昵称等）
- `limit`: 返回记录数量
- `offset`: 分页偏移量
- `format`: 输出格式，支持 `json`、`csv` 或纯文本

### 其他 API 接口

- **联系人列表**：`GET /api/v1/contact`
- **群聊列表**：`GET /api/v1/chatroom`
- **会话列表**：`GET /api/v1/session`

### 多媒体内容

聊天记录中的多媒体内容会通过 HTTP 服务进行提供，可通过以下路径访问：

- **图片内容**：`GET /image/<id>`
- **视频内容**：`GET /video/<id>`
- **文件内容**：`GET /file/<id>`
- **语音内容**：`GET /voice/<id>`
- **多媒体内容**：`GET /data/<data dir relative path>`

当请求图片、视频、文件内容时，将返回 302 跳转到多媒体内容 URL。  
当请求语音内容时，将直接返回语音内容，并对原始 SILK 语音做了实时转码 MP3 处理。  
多媒体内容 URL 地址为基于`数据目录`的相对地址，请求多媒体内容将直接返回对应文件，并针对加密图片做了实时解密处理。

## Webhook

需开启自动解密功能，当收到特定新消息时，可以通过 HTTP POST 请求将消息推送到指定的 URL。

> 延迟测试: 本地服务消息回调延迟约 13 秒; 远程同步消息回调延迟约 45 秒。

#### 0. 回调配置

使用 TUI 模式的话，在 `$HOME/.chatlog/chatlog.json` 配置文件中，新增 `webhook` 配置。  
（Windows 用户的配置文件在 `%USERPROFILE%/.chatlog/chatlog.json`)

```json
{
  "history": [],
  "last_account": "wxuser_x",
  "webhook": {
    "host": "localhost:5030",                   # 消息中的图片、文件等 URL host
    "items": [
      {
        "url": "http://localhost:8080/webhook", # 必填，webhook 请求的URL，可配置为 n8n 等 webhook 入口 
        "talker": "wxid_123",                   # 必填，需要监控的私聊、群聊名称
        "sender": "",                           # 选填，消息发送者
        "keyword": ""                           # 选填，关键词
      }
    ]
  }
}
```

使用 server 模式的话，可以通过 `CHATLOG_WEBHOOK` 环境变量进行设置。

```shell
# 方案 1
CHATLOG_WEBHOOK='{"host":"localhost:5030","items":[{"url":"http://localhost:8080/proxy","talker":"wxid_123","sender":"","keyword":""}]}'

# 方案 2（任选一种）
CHATLOG_WEBHOOK_HOST="localhost:5030"
CHATLOG_WEBHOOK_ITEMS='[{"url":"http://localhost:8080/proxy","talker":"wxid_123","sender":"","keyword":""}]'
```

#### 1. 测试效果

启动 chatlog 并开启自动解密功能，测试回调效果

```shell
POST /webhook HTTP/1.1
Host: localhost:8080
Accept-Encoding: gzip
Content-Length: 386
Content-Type: application/json
User-Agent: Go-http-client/1.1

Body:
{
  "keyword": "",
  "lastTime": "2025-08-27 00:00:00",
  "length": 1,
  "messages": [
    {
      "seq": 1756225000000,
      "time": "2025-08-27T00:00:00+08:00",
      "talker": "wxid_123",
      "talkerName": "",
      "isChatRoom": false,
      "sender": "wxid_123",
      "senderName": "Name",
      "isSelf": false,
      "type": 1,
      "subType": 0,
      "content": "测试消息",
      "contents": {
        "host": "localhost:5030"
      }
    }
  ],
  "sender": "",
  "talker": "wxid_123"
}
```

## MCP 集成

Chatlog 支持 MCP (Model Context Protocol) 协议，可与支持 MCP 的 AI 助手无缝集成。  
启动 HTTP 服务后，通过 Streamable HTTP Endpoint 访问服务：

```
GET /mcp
```

### 快速集成

Chatlog 可以与多种支持 MCP 的 AI 助手集成，包括：

- **ChatWise**: 直接支持 Streamable HTTP，在工具设置中添加 `http://127.0.0.1:5030/mcp`
- **Cherry Studio**: 直接支持 Streamable HTTP，在 MCP 服务器设置中添加 `http://127.0.0.1:5030/mcp`

对于不直接支持 Streamable HTTP 的客户端，可以使用 [mcp-proxy](https://github.com/sparfenyuk/mcp-proxy) 工具转发请求：

- **Claude Desktop**: 通过 mcp-proxy 支持，需要配置 `claude_desktop_config.json`
- **Monica Code**: 通过 mcp-proxy 支持，需要配置 VSCode 插件设置

### 详细集成指南

查看 [MCP 集成指南](docs/mcp.md) 获取各平台的详细配置步骤和注意事项。

## Prompt 示例

为了帮助大家更好地利用 Chatlog 与 AI 助手，我们整理了一些 prompt 示例。希望这些 prompt 可以启发大家更有效地查询和分析聊天记录，获取更精准的信息。

查看 [Prompt 指南](docs/prompt.md) 获取详细示例。

同时欢迎大家分享使用经验和 prompt！如果您有好的 prompt 示例或使用技巧，请通过 [Discussions](https://github.com/sjzar/chatlog/discussions) 进行分享，共同进步。

## 免责声明

⚠️ **重要提示：使用本项目前，请务必阅读并理解完整的 [免责声明](./DISCLAIMER.md)。**

本项目仅供学习、研究和个人合法使用，禁止用于任何非法目的或未授权访问他人数据。下载、安装或使用本工具即表示您同意遵守免责声明中的所有条款，并自行承担使用过程中的全部风险和法律责任。

### 摘要（请阅读完整免责声明）

- 仅限处理您自己合法拥有的聊天数据或已获授权的数据
- 严禁用于未经授权获取、查看或分析他人聊天记录
- 开发者不对使用本工具可能导致的任何损失承担责任
- 使用第三方 LLM 服务时，您应遵守这些服务的使用条款和隐私政策

**本项目完全免费开源，任何以本项目名义收费的行为均与本项目无关。**

## License

本项目基于 [Apache-2.0 许可证](./LICENSE) 开源。

## 隐私政策

本项目不收集任何用户数据。所有数据处理均在用户本地设备上进行。使用第三方服务时，请参阅相应服务的隐私政策。

## Thanks

- [@0xlane](https://github.com/0xlane) 的 [wechat-dump-rs](https://github.com/0xlane/wechat-dump-rs) 项目
- [@xaoyaoo](https://github.com/xaoyaoo) 的 [PyWxDump](https://github.com/xaoyaoo/PyWxDump) 项目
- [@git-jiadong](https://github.com/git-jiadong) 的 [go-lame](https://github.com/git-jiadong/go-lame) 和 [go-silk](https://github.com/git-jiadong/go-silk) 项目
- [Anthropic](https://www.anthropic.com/) 的 [MCP]((https://github.com/modelcontextprotocol) ) 协议
- 各个 Go 开源库的贡献者们