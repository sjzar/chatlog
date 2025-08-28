# Docker 部署指南

## 目录
- [Docker 部署指南](#docker-部署指南)
  - [目录](#目录)
  - [部署准备](#部署准备)
    - [获取微信密钥](#获取微信密钥)
    - [定位微信数据目录](#定位微信数据目录)
  - [Docker 镜像获取](#docker-镜像获取)
  - [部署方式](#部署方式)
    - [Docker Run 方式](#docker-run-方式)
    - [Docker Compose 方式](#docker-compose-方式)
  - [环境变量配置](#环境变量配置)
  - [数据目录挂载](#数据目录挂载)
    - [微信数据目录](#微信数据目录)
    - [工作目录](#工作目录)
  - [远程同步部署](#远程同步部署)
      - [配置指南](#配置指南)
      - [部署注意事项](#部署注意事项)
  - [部署验证](#部署验证)
  - [常见问题](#常见问题)
    - [1. 容器启动失败](#1-容器启动失败)
    - [2. 无法访问 HTTP 服务](#2-无法访问-http-服务)
    - [3. 数据目录权限问题](#3-数据目录权限问题)
    - [4. 密钥格式错误](#4-密钥格式错误)
    - [5. 微信版本检测失败](#5-微信版本检测失败)
    - [6. 端口冲突](#6-端口冲突)

## 部署准备

由于 Docker 容器运行环境与宿主机隔离，无法直接获取微信进程密钥，因此需要预先在宿主机上获取密钥信息。

### 获取微信密钥

在宿主机上运行 chatlog 获取密钥信息：

```shell
# 下载并运行 chatlog
$ chatlog key

# 输出示例
Data Key: [c0163e***ac3dc6]
Image Key: [38636***653361]
```

> 💡 **提示**: 
> - macOS 用户需要临时关闭 SIP 才能获取密钥，详见 [macOS 版本说明](../README.md#macos-版本说明)

### 定位微信数据目录

根据不同操作系统，微信数据目录位置如下：

**Windows 系统**:
```
# 微信 3.x 版本
C:\Users\{用户名}\Documents\WeChat Files\{微信ID}

# 微信 4.x 版本
C:\Users\{用户名}\Documents\xwechat_files\{微信ID}
```

**macOS 系统**:
```
# 微信 3.x 版本
/Users/{用户名}/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/{版本号}/{微信ID}

# 微信 4.x 版本
/Users/{用户名}/Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/{微信ID}
```

## Docker 镜像获取

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

## 部署方式

### Docker Run 方式

**基础部署**:
```shell
docker run -d \
  --name chatlog \
  -p 5030:5030 \
  -v /path/to/your/wechat/data:/app/data \
  sjzar/chatlog:latest
```

> 这种部署方式依赖于数据目录下的 chatlog.json 文件作为配置，通过 chatlog 获取密钥时将自动更新 chatlog.json 文件

**完整配置示例**:
```shell
docker run -d \
  --name chatlog \
  -p 5030:5030 \
  -e TZ=Asia/Shanghai \
  -e CHATLOG_PLATFORM=darwin \
  -e CHATLOG_VERSION=4 \
  -e CHATLOG_DATA_KEY="your-data-key" \
  -e CHATLOG_IMG_KEY="your-img-key" \
  -e CHATLOG_AUTO_DECRYPT=true \
  -e CHATLOG_HTTP_ADDR=0.0.0.0:5030 \
  -e CHATLOG_DATA_DIR=/app/data \
  -e CHATLOG_WORK_DIR=/app/work \
  -v /path/to/your/wechat/data:/app/data \
  -v /path/to/work:/app/work \
  --restart unless-stopped \
  sjzar/chatlog:latest
```

### Docker Compose 方式

**1. 创建 docker-compose.yml 文件**

```yaml
version: '3.8'

services:
  chatlog:
    image: sjzar/chatlog:latest
    restart: unless-stopped
    ports:
      - "5030:5030"  # 可修改主机端口，如 "8080:5030"
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Asia/Shanghai
      # 微信平台类型，可选：windows, darwin
      - CHATLOG_PLATFORM=darwin
      # 微信版本，可选：3, 4
      - CHATLOG_VERSION=4
      # 微信数据密钥
      - CHATLOG_DATA_KEY=your-data-key
      # 微信图片密钥
      - CHATLOG_IMG_KEY=your-img-key
      # 是否自动解密
      - CHATLOG_AUTO_DECRYPT=true
      # 服务地址
      - CHATLOG_HTTP_ADDR=0.0.0.0:5030
      # 数据目录
      - CHATLOG_DATA_DIR=/app/data
      # 工作目录
      - CHATLOG_WORK_DIR=/app/work
    volumes:
      # 微信数据目录挂载
      - "/path/to/your/wechat/data:/app/data"
      # 工作目录挂载
      - "work-dir:/app/work"

volumes:
  work-dir:
    driver: local
```

**2. 启动服务**

```shell
# 启动服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs chatlog

# 停止服务
docker-compose down
```

## 环境变量配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `PUID` | 用户 ID | `1000` | `1000` |
| `PGID` | 用户组 ID | `1000` | `1000` |
| `TZ` | 时区设置 | `UTC` | `Asia/Shanghai` |
| `CHATLOG_PLATFORM` | 微信平台类型 | **必填** | `windows`, `darwin` |
| `CHATLOG_VERSION` | 微信版本 | **必填** | `3`, `4` |
| `CHATLOG_DATA_KEY` | 微信数据密钥 | **必填** | `c0163e***ac3dc6` |
| `CHATLOG_IMG_KEY` | 微信图片密钥 | 可选 | `38636***653361` |
| `CHATLOG_HTTP_ADDR` | HTTP 服务监听地址 | `0.0.0.0:5030` | `0.0.0.0:8080` |
| `CHATLOG_AUTO_DECRYPT` | 是否自动解密 | `false` | `true`, `false` |
| `CHATLOG_DATA_DIR` | 数据目录路径 | `/app/data` | `/app/data` |
| `CHATLOG_WORK_DIR` | 工作目录路径 | `/app/work` | `/app/work` |

## 数据目录挂载

### 微信数据目录

**Windows 示例**:
```shell
# 微信 4.x 版本
-v "/c/Users/username/Documents/xwechat_files/wxid_xxx:/app/data"

# 微信 3.x 版本
-v "/c/Users/username/Documents/WeChat\ Files/wxid_xxx:/app/data"
```

**macOS 示例**:
```shell
# 微信 4.x 版本
-v "/Users/username/Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/wxid_xxx:/app/data"

# 微信 3.x 版本
-v "/Users/username/Library/Containers/com.tencent.xinWeChat/Data/Library/Application\ Support/com.tencent.xinWeChat/2.0b4.0.9:/app/data"
```

### 工作目录

工作目录用于存放解密后的数据库文件，可以使用以下两种方式：

**本地路径方式**:
```shell
-v "/path/to/local/work:/app/work"
```

**命名卷方式**:
```shell
-v "chatlog-work:/app/work"
```


## 远程同步部署

对于需要将 chatlog 服务与微信客户端分离部署的场景，可以通过文件同步工具将微信数据同步到远程服务器，然后在远程服务器上运行 chatlog 服务。这种方式具有以下优势：

- **解耦部署**：微信客户端和 chatlog 服务可以运行在不同的设备上
- **灵活性**：可以在 NAS、VPS 等服务器上统一管理聊天数据
- **安全性**：避免在个人电脑上长期运行服务

文件同步工具这里不做过多推荐，个人使用 [Syncthing](https://github.com/syncthing/syncthing)，其他选择有 [Resilio Sync](https://www.resilio.com/sync/)、[rsync + inotify](https://github.com/RsyncProject/rsync) 等，可以按需选择。

#### 配置指南

- 本地配置: 同步数据目录(Data Dir)，可设置为仅发送；在首次完整同步文件后，建议将 "rescanIntervalS" 设置为 0，全局扫描较为耗时，且扫描过程中会暂停同步
- 远程服务器配置: 设置为仅接收，同样建议将 "rescanIntervalS" 设置为 0
- 使用 Docker / Docker Compose 启动 chatlog，将数据目录映射到容器的 `/app/data` 目录
- 按需配置 `/app/work` 映射目录，可配置到远程服务器本地路径或命名卷
- 启动容器后，等待首次解密完成后，即可正常请求 API 或接入 MCP 服务

#### 部署注意事项

- 千万注意数据安全！chatlog 本身未提供授权机制，一定要确保服务处于安全网络环境中。

通过远程同步部署，您可以在保持微信客户端正常使用的同时，将 chatlog 服务部署到更适合的环境中，实现数据处理与日常使用的分离。

## 部署验证

部署完成后，通过以下方式验证服务是否正常运行：

**1. 检查容器状态**
```shell
docker ps | grep chatlog
```

**2. 查看服务日志**
```shell
docker logs chatlog
```

**3. 访问 HTTP API**
```shell
# 检查服务健康状态
curl http://localhost:5030/api/v1/session

# 查看联系人列表
curl http://localhost:5030/api/v1/contact
```

**4. 访问 MCP 服务**
```shell
http://localhost:5030/mcp
```

**5. 访问 Web 界面**

在浏览器中打开：http://localhost:5030

## 常见问题

### 1. 容器启动失败

**问题**: 容器启动后立即退出

**解决方案**:
- 检查密钥是否正确：`docker logs chatlog`
- 确认数据目录挂载路径是否正确
- 检查环境变量配置是否完整

### 2. 无法访问 HTTP 服务

**问题**: 浏览器无法访问 http://localhost:5030

**解决方案**:
- 检查端口映射是否正确：`docker port chatlog`
- 确认防火墙是否允许 5030 端口访问
- 检查容器内服务是否正常启动

### 3. 数据目录权限问题

**问题**: 日志显示权限不足或文件无法访问

**解决方案**:
```shell
# Linux/macOS 系统
chmod -R 755 /path/to/your/wechat/data

# 或者使用 Docker 用户权限
docker run --user $(id -u):$(id -g) ...
```

### 4. 密钥格式错误

**问题**: 显示密钥格式不正确

**解决方案**:
- 确保密钥为十六进制格式，不包含方括号
- 正确格式：`CHATLOG_DATA_KEY=c0163eac3dc6`
- 错误格式：`CHATLOG_DATA_KEY=[c0163e***ac3dc6]`

### 5. 微信版本检测失败

**问题**: 无法自动检测微信版本

**解决方案**:
- 手动设置微信平台：`CHATLOG_PLATFORM=darwin` 或 `CHATLOG_PLATFORM=windows`
- 手动设置微信版本：`CHATLOG_VERSION=4` 或 `CHATLOG_VERSION=3`

### 6. 端口冲突

**问题**: 5030 端口已被占用

**解决方案**:
```shell
# 使用其他端口，如 8080
docker run -p 8080:5030 ...

# 或在 docker-compose.yml 中修改
ports:
  - "8080:5030"
```

> 💡 **获取更多帮助**: 如遇到其他问题，请查看项目的 [Issues](https://github.com/sjzar/chatlog/issues) 页面或提交新的问题反馈。
