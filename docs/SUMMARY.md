# Chatlog 文档总览

这是 Chatlog 项目的完整文档集合，包含了用户指南、技术文档、API 参考和故障排除指南。

## 📖 文档结构

```
docs/
├── README.md                    # 文档导航和快速开始
├── SUMMARY.md                   # 本文件 - 文档总览
│
├── 🎯 核心文档
│   ├── architecture.md          # 系统架构设计
│   ├── api.md                   # HTTP API 完整文档  
│   ├── database.md              # 微信数据库结构分析
│   ├── development.md           # 开发环境和贡献指南
│   ├── logging.md               # 日志系统详细说明
│   └── troubleshooting.md       # 故障排除指南
│
├── 📚 user-guides/ (用户指南)
│   ├── mcp.md                   # MCP 协议集成指南
│   └── prompt.md                # AI 对话 Prompt 优化
│
├── 🔧 technical/ (技术细节)  
│   ├── voice_id_analysis.md     # 语音消息 ID 分析
│   ├── voice_key_fix_summary.md # 语音密钥问题修复
│   ├── voice_logging_summary.md # 语音处理日志记录
│   ├── voice_mediamsg16_fix.md  # MediaMsg16 问题修复
│   ├── voice_processing_logic.md # 语音消息处理逻辑
│   └── 微信PC端各个数据库文件结构与功能简述 - Multi文件夹.md
│
└── 🚨 troubleshooting-guides/ (故障排除)
    └── voice_debug_guide.md     # 语音消息调试专项指南
```

## 🎯 按用途分类

### 👥 新用户必读
1. [主项目 README](../README.md) - 项目介绍和基本使用
2. [文档导航](README.md) - 文档使用指南
3. [MCP 集成指南](user-guides/mcp.md) - AI 助手集成
4. [故障排除指南](troubleshooting.md) - 常见问题解决

### 🔧 开发者资源
1. [系统架构](architecture.md) - 了解项目设计
2. [开发指南](development.md) - 开发环境搭建
3. [API 文档](api.md) - 接口开发参考
4. [数据库结构](database.md) - 数据模型理解

### 🚀 高级用户
1. [日志系统](logging.md) - 调试和监控
2. [语音处理技术细节](technical/) - 深入了解语音功能
3. [数据库结构分析](technical/微信PC端各个数据库文件结构与功能简述%20-%20Multi文件夹.md)

### 🆘 问题排查
1. [故障排除指南](troubleshooting.md) - 通用问题解决
2. [语音调试指南](troubleshooting-guides/voice_debug_guide.md) - 语音问题专项
3. [技术细节文档](technical/) - 深度技术问题

## 📊 文档统计

| 分类 | 文件数 | 说明 |
|------|--------|------|
| 核心文档 | 6 | 架构、API、开发、日志等核心内容 |
| 用户指南 | 2 | MCP 集成和 Prompt 优化 |
| 技术细节 | 6 | 语音处理和数据库结构深度分析 |
| 故障排除 | 2 | 通用故障排除和语音问题专项 |
| **总计** | **16** | **完整的文档体系** |

## 🔄 文档更新

### 版本控制
- 所有文档都跟随项目版本进行版本控制
- 重要变更会在 [CHANGELOG](../CHANGELOG.md) 中记录
- 文档问题请提交 [Issue](https://github.com/sjzar/chatlog/issues)

### 贡献指南
1. 发现文档错误或不足，请提交 Issue
2. 想要贡献文档，请参考 [开发指南](development.md)
3. 大的文档改动请先讨论，避免重复工作

### 维护状态

| 文档类型 | 维护状态 | 更新频率 |
|----------|----------|----------|
| 核心文档 | 🟢 积极维护 | 随版本更新 |
| 用户指南 | 🟢 积极维护 | 功能变更时 |
| 技术细节 | 🟡 按需更新 | 相关功能变更时 |
| 故障排除 | 🟢 积极维护 | 发现新问题时 |

## 🎯 文档质量

### 质量标准
- ✅ 内容准确性：与代码实现保持一致
- ✅ 完整性：覆盖主要功能和使用场景  
- ✅ 可读性：结构清晰，示例丰富
- ✅ 实用性：解决实际问题，提供可操作指导

### 反馈渠道
- **GitHub Issues**: 报告文档问题
- **Discussions**: 文档改进建议
- **Pull Requests**: 直接贡献文档改进

## 🏷️ 文档标签

为方便查找，文档使用以下标签体系：

- 🚀 **quickstart**: 快速开始相关
- 🔧 **technical**: 技术实现详情
- 📖 **tutorial**: 教程指南
- 🐛 **troubleshooting**: 问题排查
- 💡 **tips**: 使用技巧
- ⚠️ **important**: 重要提醒
- 🆕 **new**: 新增功能

---

> 💡 **提示**: 建议收藏本页面作为文档导航的起点，快速定位需要的文档内容。