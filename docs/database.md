# 微信数据库结构文档

## 概述

微信客户端使用 SQLite 数据库存储聊天记录、联系人信息、媒体文件等数据。不同版本和平台的微信在数据库结构上略有差异。本文档详细说明了各种数据库文件的结构和功能。

## 🗂️ 数据库文件分类

### 主要数据库文件

| 文件名 | 功能 | 加密状态 | 重要程度 |
|--------|------|----------|-----------|
| `MSG*.db` | 聊天消息记录 | 加密 | ⭐⭐⭐⭐⭐ |
| `MicroMsg.db` | 联系人和基础信息 | 加密 | ⭐⭐⭐⭐⭐ |
| `MediaMSG*.db` | 语音消息数据 | 加密 | ⭐⭐⭐⭐ |
| `FTSMSG*.db` | 全文搜索索引 | 加密 | ⭐⭐⭐ |
| `Session.db` | 会话列表 | 加密 | ⭐⭐⭐⭐ |
| `ContactHeadImg*.db` | 头像缓存 | 未加密 | ⭐⭐ |
| `Emotion.db` | 表情包数据 | 未加密 | ⭐⭐ |

### 版本差异

#### Windows 微信版本
- **v3.x**: 数据库文件相对独立，结构较为简单
- **v4.x**: 引入了新的数据库结构，增强了数据完整性

#### macOS 微信版本  
- **v3.x (Darwin)**: 与 Windows 版本结构类似，但字段略有差异
- **v4.x**: 结构与 Windows v4.x 基本一致

## 📊 核心数据库详解

### 1. MSG 数据库 (MSG*.db)

**主表: MSG**

聊天消息的核心存储表，包含所有类型的消息记录。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| localId | INTEGER | 本地消息ID (主键) | 123456 |
| TalkerId | INTEGER | 对话者ID | 1 |
| MsgSvrID | INTEGER | 服务器消息ID | 987654321 |
| Type | INTEGER | 消息类型 | 1(文本), 3(图片), 34(语音) |
| SubType | INTEGER | 消息子类型 | 0 |
| IsSender | INTEGER | 是否为发送者 | 1(是), 0(否) |
| CreateTime | INTEGER | 创建时间 (时间戳) | 1672531200 |
| Sequence | INTEGER | 消息序列号 | 1 |
| StatusEx | INTEGER | 扩展状态 | 0 |
| FlagEx | INTEGER | 扩展标志 | 0 |
| Status | INTEGER | 消息状态 | 2(已读) |
| MsgServerSeq | INTEGER | 服务器序列号 | 123 |
| MsgSequence | INTEGER | 消息序列 | 456 |
| StrTalker | TEXT | 对话者标识 | wxid_xxx 或 群ID@chatroom |
| StrContent | TEXT | 消息内容 | Hello World! |
| DisplayContent | TEXT | 显示内容 | Hello World! |
| Reserved0 | INTEGER | 保留字段0 | 0 |
| Reserved1 | INTEGER | 保留字段1 | 0 |
| Reserved3 | INTEGER | 保留字段3 | 0 |
| Reserved4 | TEXT | 保留字段4 | |
| Reserved5 | INTEGER | 保留字段5 | 0 |
| Reserved6 | TEXT | 保留字段6 | |
| CompressContent | BLOB | 压缩内容 | |
| BytesExtra | BLOB | 额外字节数据 | |
| BytesTrans | BLOB | 传输字节数据 | |

**消息类型说明:**

| Type | 说明 | StrContent 示例 |
|------|------|------------------|
| 1 | 文本消息 | "Hello World!" |
| 3 | 图片消息 | XML格式的图片信息 |
| 34 | 语音消息 | XML格式的语音信息 |
| 43 | 视频消息 | XML格式的视频信息 |
| 47 | 表情/动画表情 | XML格式的表情信息 |
| 48 | 位置消息 | XML格式的位置信息 |
| 49 | 链接/文件/小程序 | XML格式的资源信息 |
| 10000 | 系统消息 | "你已添加xxx，可以开始聊天了" |

### 2. MicroMsg 数据库 (MicroMsg.db)

**主表: Contact**

存储联系人信息，包括好友、群组、公众号等。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| UserName | TEXT | 用户标识 (主键) | wxid_xxx 或 群ID@chatroom |
| Alias | TEXT | 微信号 | zhangsan |
| EncryptUserName | TEXT | 加密用户名 | |
| DelFlag | INTEGER | 删除标志 | 0(正常), 1(已删除) |
| Type | INTEGER | 联系人类型 | 3(好友), 2(群组) |
| VerifyFlag | INTEGER | 验证标志 | 0(无), 8(公众号) |
| Reserved1 | INTEGER | 保留字段1 | 0 |
| Reserved2 | INTEGER | 保留字段2 | 0 |
| Reserved3 | TEXT | 保留字段3 | |
| Reserved4 | TEXT | 保留字段4 | |
| Remark | TEXT | 备注名 | 小张 |
| NickName | TEXT | 昵称 | 张三 |
| LabelIDList | TEXT | 标签ID列表 | |
| DomainList | TEXT | 域名列表 | |
| ChatRoomType | INTEGER | 群聊类型 | 0 |
| PYInitial | TEXT | 拼音首字母 | ZS |
| QuanPin | TEXT | 全拼 | zhangsan |
| RemarkPYInitial | TEXT | 备注拼音首字母 | XZ |
| RemarkQuanPin | TEXT | 备注全拼 | xiaozhang |
| BigHeadImgUrl | TEXT | 大头像URL | |
| SmallHeadImgUrl | TEXT | 小头像URL | |
| HeadImgMd5 | TEXT | 头像MD5 | |
| ChatRoomNotify | INTEGER | 群聊通知 | 0 |
| Reserved5 | INTEGER | 保留字段5 | 0 |
| Reserved6 | TEXT | 保留字段6 | |
| Reserved7 | INTEGER | 保留字段7 | 0 |
| ExtraBuf | BLOB | 额外缓冲区 | |

**主表: ChatRoom**

存储群组详细信息。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| ChatRoomName | TEXT | 群组标识 (主键) | 123456789@chatroom |
| UserNameList | TEXT | 成员用户名列表 | wxid_a;wxid_b;wxid_c |
| DisplayNameList | TEXT | 成员显示名列表 | Alice;Bob;Charlie |
| ChatRoomFlag | INTEGER | 群组标志 | 0 |
| Owner | INTEGER | 群主标志 | 0 |
| IsShowName | INTEGER | 是否显示成员名称 | 1 |
| SelfDisplayName | TEXT | 自己的群内昵称 | |
| Reserved1 | INTEGER | 保留字段1 | 0 |
| Reserved2 | INTEGER | 保留字段2 | 0 |
| Reserved3 | TEXT | 保留字段3 | |
| Reserved4 | TEXT | 保留字段4 | |
| Reserved5 | INTEGER | 保留字段5 | 0 |
| Reserved6 | TEXT | 保留字段6 | |
| RoomData | BLOB | 群组数据 (ProtoBuf) | |

### 3. MediaMSG 数据库 (MediaMSG*.db)

**主表: Media**

存储语音消息的二进制数据。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| Key | TEXT | 媒体键值 (主键) | voice_key_xxx |
| Reserved0 | TEXT | 关联的消息ID | 对应 MSG 表的 MsgSvrID |
| Buf | BLOB | 媒体数据 | SILK 格式的语音数据 |
| Reserved1 | INTEGER | 保留字段1 | 0 |
| Reserved2 | TEXT | 保留字段2 | |

**SILK 音频格式:**
- 微信使用 SILK 编码格式存储语音数据
- 需要解码为 PCM 后转换为 MP3 等常见格式
- 文件头通常为 `#!SILK_V3`

### 4. Session 数据库 (Session.db)

**主表: Session**

存储最近会话列表。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| strTalker | TEXT | 对话者标识 (主键) | wxid_xxx |
| nOrder | INTEGER | 排序顺序 | 1 |
| parentRef | TEXT | 父引用 | |
| nUnReadCount | INTEGER | 未读消息数 | 0 |
| nStatus | INTEGER | 会话状态 | 0 |
| nIsSend | INTEGER | 是否发送 | 1 |
| msgType | INTEGER | 消息类型 | 1 |
| nMsgLocalID | INTEGER | 本地消息ID | 123456 |
| nMsgStatus | INTEGER | 消息状态 | 2 |
| strUsrName | TEXT | 用户名 | wxid_xxx |
| strNickName | TEXT | 昵称 | 张三 |
| nTime | INTEGER | 时间戳 | 1672531200 |
| strContent | TEXT | 内容摘要 | 最后一条消息内容 |
| strDigest | TEXT | 摘要 | 消息摘要 |
| digestTime | INTEGER | 摘要时间 | 1672531200 |
| Reserved1 | INTEGER | 保留字段1 | 0 |
| Reserved2 | TEXT | 保留字段2 | |

### 5. FTSMSG 数据库 (FTSMSG*.db)

**主表: FTSChatMsg_content**

全文搜索内容表。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| docid | INTEGER | 文档ID (主键) | 1 |
| c0content | TEXT | 搜索内容 | Hello World |
| c1entityId | TEXT | 实体ID | xxx |

**主表: FTSChatMsg_MetaData**

全文搜索元数据表。

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| docid | INTEGER | 文档ID (主键) | 1 |
| msgId | INTEGER | 消息ID | 对应 MSG 表的 localId |
| entityId | TEXT | 实体ID | xxx |
| type | INTEGER | 类型 | 1 |

## 🔄 版本差异详解

### Windows v3.x vs v4.x

#### 消息表 (MSG) 差异

**v3.x 特有字段:**
```sql
-- v3.x 特有的字段和索引
CREATE INDEX "MSG_Index_CreateTime" ON "MSG" ("CreateTime");
CREATE INDEX "MSG_Index_MsgSvrID" ON "MSG" ("MsgSvrID");
```

**v4.x 新增字段:**
```sql
-- v4.x 增强的字段
ALTER TABLE MSG ADD COLUMN CompressContentV2 BLOB;
ALTER TABLE MSG ADD COLUMN BytesTransV2 BLOB;
```

#### 联系人表 (Contact) 差异

**v4.x 增强:**
- 增加了更多的保留字段用于扩展
- 优化了索引结构
- 增强了群组管理功能

### macOS (Darwin) 特殊性

#### Darwin v3.x
```sql
-- macOS 特有的表结构调整
-- 某些字段类型略有差异，但功能相同
CREATE TABLE "Contact" (
    "UserName" TEXT PRIMARY KEY,
    "Alias" TEXT,
    -- 其他字段基本相同，但可能有顺序差异
);
```

## 🔍 数据查询示例

### 常用查询语句

#### 1. 获取所有联系人
```sql
SELECT 
    UserName,
    NickName,
    Remark,
    Type
FROM Contact 
WHERE DelFlag = 0
ORDER BY Type, NickName;
```

#### 2. 查询特定联系人的聊天记录
```sql
SELECT 
    m.CreateTime,
    m.Type,
    m.IsSender,
    m.StrContent,
    c.NickName
FROM MSG m
LEFT JOIN Contact c ON m.StrTalker = c.UserName
WHERE m.StrTalker = 'wxid_xxx'
ORDER BY m.CreateTime DESC
LIMIT 100;
```

#### 3. 获取群聊信息
```sql
SELECT 
    cr.ChatRoomName,
    cr.UserNameList,
    cr.DisplayNameList,
    c.NickName as GroupName
FROM ChatRoom cr
LEFT JOIN Contact c ON cr.ChatRoomName = c.UserName
WHERE cr.ChatRoomName LIKE '%@chatroom';
```

#### 4. 搜索包含关键词的消息
```sql
SELECT 
    m.CreateTime,
    m.StrTalker,
    m.StrContent,
    c.NickName
FROM MSG m
LEFT JOIN Contact c ON m.StrTalker = c.UserName
WHERE m.Type = 1 
  AND m.StrContent LIKE '%关键词%'
ORDER BY m.CreateTime DESC;
```

#### 5. 获取语音消息
```sql
SELECT 
    m.MsgSvrID,
    m.CreateTime,
    m.StrTalker,
    media.Key,
    LENGTH(media.Buf) as VoiceSize
FROM MSG m
JOIN Media media ON m.MsgSvrID = media.Reserved0
WHERE m.Type = 34
ORDER BY m.CreateTime DESC;
```

#### 6. 统计每日消息数量
```sql
SELECT 
    DATE(m.CreateTime, 'unixepoch', 'localtime') as Date,
    COUNT(*) as MessageCount
FROM MSG m
WHERE m.CreateTime > strftime('%s', 'now', '-30 days')
GROUP BY DATE(m.CreateTime, 'unixepoch', 'localtime')
ORDER BY Date DESC;
```

#### 7. 获取最活跃的联系人
```sql
SELECT 
    m.StrTalker,
    c.NickName,
    c.Remark,
    COUNT(*) as MessageCount
FROM MSG m
LEFT JOIN Contact c ON m.StrTalker = c.UserName
WHERE m.CreateTime > strftime('%s', 'now', '-7 days')
  AND m.StrTalker NOT LIKE '%@chatroom'
GROUP BY m.StrTalker
ORDER BY MessageCount DESC
LIMIT 10;
```

## 🗃️ Multi 文件夹特殊结构

### 文件组织方式

Multi 文件夹包含大容量数据的分片存储：

```
Multi/
├── MSG0.db          # 消息数据库分片1
├── MSG1.db          # 消息数据库分片2
├── MediaMSG0.db     # 媒体消息分片1
├── MediaMSG1.db     # 媒体消息分片2
└── FTSMSG0.db       # 搜索索引分片1
```

### 分片规则

1. **消息分片**: 按时间或大小自动分片
2. **媒体分片**: 按媒体类型或大小分片
3. **索引分片**: 对应消息分片的索引

### 查询跨分片数据

```sql
-- 需要联合查询多个分片
SELECT * FROM (
    SELECT * FROM MSG0.MSG
    UNION ALL
    SELECT * FROM MSG1.MSG
) 
WHERE StrTalker = 'wxid_xxx'
ORDER BY CreateTime DESC;
```

## 🔒 加密机制

### 数据库加密

微信使用 SQLCipher 对数据库进行加密：

1. **密钥获取**: 从微信进程内存中提取
2. **密钥验证**: 通过尝试打开数据库验证密钥正确性
3. **解密过程**: 使用提取的密钥创建解密后的副本

### 解密后的结构

解密后的数据库与标准 SQLite 数据库完全兼容，可以使用任何 SQLite 工具进行操作。

## 📈 性能优化建议

### 索引优化

```sql
-- 为常用查询创建索引
CREATE INDEX IF NOT EXISTS idx_msg_talker_time 
ON MSG(StrTalker, CreateTime);

CREATE INDEX IF NOT EXISTS idx_msg_type_time 
ON MSG(Type, CreateTime);

CREATE INDEX IF NOT EXISTS idx_contact_nickname 
ON Contact(NickName);
```

### 查询优化

1. **使用分页**: 避免一次性加载大量数据
2. **时间范围**: 限制查询的时间范围
3. **类型过滤**: 先过滤消息类型再查询内容
4. **索引利用**: 确保 WHERE 条件能够利用索引

### 存储优化

1. **定期清理**: 删除不必要的临时数据
2. **压缩数据库**: 使用 `VACUUM` 命令压缩数据库
3. **分表存储**: 对于超大数据量，考虑按时间分表

## 🔧 开发者工具

### 数据库查看工具

1. **SQLite Browser**: 图形化数据库管理工具
2. **DBeaver**: 通用数据库管理工具  
3. **sqlite3 命令行**: 轻量级命令行工具

### 调试技巧

```bash
# 检查数据库结构
sqlite3 MSG0.db ".schema"

# 检查数据库完整性
sqlite3 MSG0.db "PRAGMA integrity_check;"

# 查看数据库统计信息
sqlite3 MSG0.db ".dbinfo"

# 导出数据
sqlite3 MSG0.db ".dump" > backup.sql
```

## ⚠️ 注意事项

### 数据安全

1. **备份原数据**: 在解密前务必备份原始数据库文件
2. **权限控制**: 确保解密后的数据库文件权限设置正确
3. **清理临时文件**: 及时清理解密过程中产生的临时文件

### 兼容性

1. **版本差异**: 不同版本的微信数据库结构可能有差异
2. **平台差异**: Windows 和 macOS 版本略有不同
3. **字段变化**: 新版本可能增加或修改字段

### 性能考虑

1. **内存使用**: 大型数据库可能消耗大量内存
2. **查询超时**: 复杂查询可能需要较长时间
3. **并发访问**: 避免同时访问同一数据库文件