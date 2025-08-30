# Chatlog API 文档

## 概述

Chatlog 提供了完整的 RESTful API 来访问微信聊天记录数据。API 默认运行在 `http://127.0.0.1:5030`，支持 JSON 响应格式。

## 🚀 快速开始

### 启动 API 服务

```bash
# 方式一：使用 Terminal UI
chatlog
# 然后选择 "启动 HTTP 服务"

# 方式二：直接启动服务器
chatlog server

# 方式三：指定配置启动
chatlog server --addr 127.0.0.1:8080 --data-dir /path/to/wechat/data
```

### 基本 API 调用

```bash
# 获取联系人列表
curl "http://127.0.0.1:5030/api/v1/contact"

# 获取聊天记录
curl "http://127.0.0.1:5030/api/v1/chatlog?talker=wxid_xxx&time=2024-01-01"
```

## 📚 API 接口

### 1. 聊天记录 API

#### GET /api/v1/chatlog

获取聊天记录，支持多种查询条件和输出格式。

**查询参数:**

| 参数 | 类型 | 必需 | 描述 | 示例 |
|------|------|------|------|------|
| `talker` | string | 否 | 聊天对象标识，支持 wxid、群聊ID、备注名、昵称 | `wxid_xxx` |
| `time` | string | 否 | 时间范围，支持多种格式 | `2024-01-01` 或 `2024-01-01~2024-01-31` |
| `keyword` | string | 否 | 搜索关键词 | `你好` |
| `msg_type` | int | 否 | 消息类型过滤 | `1` (文本消息) |
| `limit` | int | 否 | 返回记录数量，默认 100，最大 1000 | `50` |
| `offset` | int | 否 | 分页偏移量，默认 0 | `100` |
| `format` | string | 否 | 输出格式：`json`(默认)、`csv`、`txt` | `json` |
| `order` | string | 否 | 排序方式：`asc`、`desc`(默认) | `asc` |

**时间格式支持:**
- `2024-01-01`: 指定日期
- `2024-01-01~2024-01-31`: 时间范围
- `2024-01`: 指定月份
- `7d`: 最近7天
- `30d`: 最近30天

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 1523,
    "limit": 100,
    "offset": 0,
    "messages": [
      {
        "id": "123456789",
        "msg_svr_id": "987654321",
        "type": 1,
        "sub_type": 0,
        "is_sender": 1,
        "create_time": "2024-01-01T10:30:00Z",
        "sequence": 1,
        "status_ex": 0,
        "flag_ex": 0,
        "user_name": "wxid_xxx",
        "talker": "wxid_yyy",
        "content": "Hello, World!",
        "compressed_content": "",
        "byte_extra": "",
        "display_content": "Hello, World!",
        "reserved0": 0,
        "reserved1": 0,
        "reserved3": 0,
        "reserved4": "",
        "reserved5": 0,
        "reserved6": "",
        "reserved7": 0,
        "reserved8": 0,
        "reserved9": "",
        "compressed_content_v2": ""
      }
    ]
  }
}
```

### 2. 联系人 API

#### GET /api/v1/contact

获取联系人列表。

**查询参数:**

| 参数 | 类型 | 必需 | 描述 | 示例 |
|------|------|------|------|------|
| `keyword` | string | 否 | 搜索关键词，匹配昵称、备注、wxid | `张三` |
| `type` | string | 否 | 联系人类型：`friend`(好友)、`group`(群组)、`official`(公众号) | `friend` |
| `limit` | int | 否 | 返回记录数量，默认 100 | `50` |
| `offset` | int | 否 | 分页偏移量，默认 0 | `100` |

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 256,
    "limit": 100,
    "offset": 0,
    "contacts": [
      {
        "user_name": "wxid_xxx",
        "alias": "zhangsan",
        "nick_name": "张三",
        "remark_name": "小张",
        "label_id_list": "",
        "type": 3,
        "verify_flag": 0,
        "reserved1": 0,
        "reserved2": 0,
        "reserved3": "",
        "reserved4": ""
      }
    ]
  }
}
```

### 3. 群聊 API

#### GET /api/v1/chatroom

获取群聊列表和群聊信息。

**查询参数:**

| 参数 | 类型 | 必需 | 描述 | 示例 |
|------|------|------|------|------|
| `keyword` | string | 否 | 搜索关键词，匹配群名称 | `工作群` |
| `chat_room_name` | string | 否 | 具体群聊ID | `123456789@chatroom` |
| `limit` | int | 否 | 返回记录数量，默认 100 | `50` |
| `offset` | int | 否 | 分页偏移量，默认 0 | `100` |

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 15,
    "limit": 100,
    "offset": 0,
    "chatrooms": [
      {
        "chat_room_name": "123456789@chatroom",
        "user_name_list": "wxid_aaa;wxid_bbb;wxid_ccc",
        "display_name_list": "Alice;Bob;Charlie",
        "room_data": "群聊数据",
        "member_count": 3,
        "reserved1": 0,
        "reserved2": 0,
        "reserved3": "",
        "reserved4": ""
      }
    ]
  }
}
```

### 4. 会话 API

#### GET /api/v1/session

获取最近会话列表。

**查询参数:**

| 参数 | 类型 | 必需 | 描述 | 示例 |
|------|------|------|------|------|
| `limit` | int | 否 | 返回记录数量，默认 50 | `20` |
| `offset` | int | 否 | 分页偏移量，默认 0 | `50` |

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 128,
    "limit": 50,
    "offset": 0,
    "sessions": [
      {
        "str_talker": "wxid_xxx",
        "n_order": 1,
        "parent_ref": "",
        "n_unread_count": 0,
        "n_status": 0,
        "dt_update_time": "2024-01-01T15:30:00Z",
        "str_content": "最后一条消息内容",
        "n_msg_type": 1,
        "str_digest": "消息摘要",
        "dt_digest_time": "2024-01-01T15:30:00Z",
        "reserved1": 0,
        "reserved2": ""
      }
    ]
  }
}
```

## 🎯 多媒体内容 API

### 1. 图片内容

#### GET /image/:id

获取图片内容，自动解密并返回原始图片数据。

**路径参数:**
- `id`: 图片ID或文件名

**响应:**
- 成功: 返回 302 重定向到实际图片文件
- 失败: 返回 JSON 错误信息

### 2. 语音内容

#### GET /voice/:id

获取语音内容，自动将 SILK 格式转换为 MP3。

**路径参数:**
- `id`: 语音消息ID

**响应:**
- Content-Type: `audio/mpeg`
- 直接返回 MP3 音频数据

### 3. 视频内容

#### GET /video/:id

获取视频内容。

**路径参数:**
- `id`: 视频文件ID

**响应:**
- 返回 302 重定向到实际视频文件

### 4. 文件内容

#### GET /file/:id

获取文件内容。

**路径参数:**
- `id`: 文件ID

**响应:**
- 返回 302 重定向到实际文件

### 5. 通用数据内容

#### GET /data/:path

根据相对路径获取数据目录中的文件。

**路径参数:**
- `path`: 数据目录的相对路径

**响应:**
- 直接返回文件内容
- 自动处理加密图片解密

## 🔧 MCP 协议支持

### Server-Sent Events 端点

#### GET /sse

提供 MCP (Model Context Protocol) 的 SSE 支持。

**特性:**
- 实时双向通信
- 支持工具调用
- 资源管理
- 会话管理

**连接示例:**

```javascript
const eventSource = new EventSource('http://127.0.0.1:5030/sse');

eventSource.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('MCP Message:', data);
};
```

## 🎛️ 配置与管理

### 1. 健康检查

#### GET /health

检查服务健康状态。

**响应:**

```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 2. 服务信息

#### GET /api/v1/info

获取服务配置信息。

**响应:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "version": "v1.0.0",
    "platform": "windows",
    "wechat_version": "3.9.5.81",
    "data_dir": "C:\\Users\\...\\WeChat Files\\wxid_xxx",
    "work_dir": "C:\\Users\\...\\chatlog_data",
    "features": {
      "auto_decrypt": true,
      "mcp_support": true,
      "voice_convert": true,
      "image_decrypt": true
    }
  }
}
```

## 🚨 错误处理

### 标准错误响应

```json
{
  "code": 400,
  "message": "Invalid parameter",
  "error": "time format should be YYYY-MM-DD or YYYY-MM-DD~YYYY-MM-DD",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 常见错误码

| 错误码 | 描述 | 解决方案 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 401 | 未授权访问 | 检查认证信息 |
| 404 | 资源不存在 | 检查请求路径和参数 |
| 500 | 服务器内部错误 | 检查服务器日志 |
| 503 | 服务不可用 | 检查数据库连接和配置 |

## 📝 使用示例

### Python 示例

```python
import requests
import json

# 基础配置
BASE_URL = "http://127.0.0.1:5030"

def get_contacts():
    """获取联系人列表"""
    response = requests.get(f"{BASE_URL}/api/v1/contact")
    return response.json()

def get_chat_history(talker, days=7):
    """获取最近聊天记录"""
    params = {
        "talker": talker,
        "time": f"{days}d",
        "limit": 100
    }
    response = requests.get(f"{BASE_URL}/api/v1/chatlog", params=params)
    return response.json()

def search_messages(keyword):
    """搜索消息"""
    params = {
        "keyword": keyword,
        "limit": 50
    }
    response = requests.get(f"{BASE_URL}/api/v1/chatlog", params=params)
    return response.json()

# 使用示例
contacts = get_contacts()
print(f"总共 {contacts['data']['total']} 个联系人")

history = get_chat_history("wxid_xxx", 30)
print(f"最近30天共 {history['data']['total']} 条消息")
```

### JavaScript 示例

```javascript
class ChatlogAPI {
  constructor(baseURL = 'http://127.0.0.1:5030') {
    this.baseURL = baseURL;
  }

  async request(endpoint, params = {}) {
    const url = new URL(`${this.baseURL}${endpoint}`);
    Object.keys(params).forEach(key => {
      if (params[key] !== undefined) {
        url.searchParams.append(key, params[key]);
      }
    });

    const response = await fetch(url);
    return await response.json();
  }

  async getContacts(keyword = '', limit = 100) {
    return await this.request('/api/v1/contact', { keyword, limit });
  }

  async getChatLog(options = {}) {
    return await this.request('/api/v1/chatlog', options);
  }

  async getSessions(limit = 50) {
    return await this.request('/api/v1/session', { limit });
  }
}

// 使用示例
const api = new ChatlogAPI();

// 获取联系人
api.getContacts('张三').then(data => {
  console.log('搜索结果:', data.data.contacts);
});

// 获取聊天记录
api.getChatLog({
  talker: 'wxid_xxx',
  time: '2024-01-01~2024-01-31',
  limit: 50
}).then(data => {
  console.log('聊天记录:', data.data.messages);
});
```

## 🔒 安全注意事项

1. **本地访问**: API 默认只绑定 127.0.0.1，仅允许本地访问
2. **数据保护**: 所有数据处理都在本地进行，不会上传到外部服务器
3. **权限控制**: 确保只有授权用户可以访问聊天数据
4. **HTTPS**: 生产环境建议使用 HTTPS 和适当的认证机制

## 🚀 性能优化

1. **分页查询**: 使用 `limit` 和 `offset` 进行分页
2. **时间范围**: 合理设置时间范围以提高查询效率
3. **索引利用**: 使用 `talker` 和 `time` 参数可以利用数据库索引
4. **缓存策略**: 频繁访问的数据会被自动缓存