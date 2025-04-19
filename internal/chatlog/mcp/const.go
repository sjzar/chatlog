package mcp

import (
	"github.com/sjzar/chatlog/internal/mcp"
)

// MCPTools 和资源定义
var (
	InitializeResponse = mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    mcp.DefaultCapabilities,
		ServerInfo: mcp.ServerInfo{
			Name:    "chatlog",
			Version: "0.0.1",
		},
	}

	ToolContact = mcp.Tool{
		Name:        "query_contact",
		Description: "查询用户的联系人信息。可以通过姓名、备注名或ID进行查询，返回匹配的联系人列表。当用户询问某人的联系方式、想了解联系人信息或需要查找特定联系人时使用此工具。参数为空时，将返回联系人列表",
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"keyword": mcp.M{
					"type":        "string",
					"description": "联系人的搜索关键词，可以是姓名、备注名或ID。",
				},
			},
			Required: []string{"keyword"},
		},
	}

	ToolChatRoom = mcp.Tool{
		Name:        "query_chat_room",
		Description: "查询用户参与的群聊信息。可以通过群名称、群ID或相关关键词进行查询，返回匹配的群聊列表。当用户询问群聊信息、想了解某个群的详情或需要查找特定群聊时使用此工具。",
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"keyword": mcp.M{
					"type":        "string",
					"description": "群聊的搜索关键词，可以是群名称、群ID或相关描述",
				},
			},
			Required: []string{"keyword"},
		},
	}

	ToolRecentChat = mcp.Tool{
		Name:        "query_recent_chat",
		Description: "查询最近会话列表，包括个人聊天和群聊。当用户想了解最近的聊天记录、查看最近联系过的人或群组时使用此工具。不需要参数，直接返回最近的会话列表。",
		InputSchema: mcp.ToolSchema{
			Type:       "object",
			Properties: mcp.M{},
		},
	}

	ToolChatLog = mcp.Tool{
		Name: "chatlog",
		Description: `检索历史聊天记录，可根据时间、对话方、发送者和关键词等条件进行精确查询。当用户需要查找特定信息或想了解与某人/某群的历史交流时使用此工具。

【强制多步查询流程!】
当查询特定话题或特定发送者发言时，必须严格按照以下流程使用，任何偏离都会导致错误的结果：

步骤1: 初步定位相关消息
- 使用keyword参数查找特定话题
- 使用sender参数查找特定发送者的消息
- 使用较宽时间范围初步查询

步骤2: 【必须执行】针对每个关键结果点分别获取上下文
- 必须对步骤1返回的每个时间点T1, T2, T3...分别执行独立查询（时间范围接近的消息可以合并为一个查询）
- 每次独立查询必须移除keyword参数
- 每次独立查询必须移除sender参数
- 每次独立查询使用"Tn前后15-30分钟"的窄范围
- 每次独立查询仅保留talker参数

步骤3: 【必须执行】综合分析所有上下文
- 必须等待所有步骤2的查询结果返回后再进行分析
- 必须综合考虑所有上下文信息后再回答用户

【严格执行规则！】
- 禁止仅凭步骤1的结果直接回答用户
- 禁止在步骤2使用过大的时间范围一次性查询所有上下文
- 禁止跳过步骤2或步骤3
- 必须对每个关键结果点分别执行独立的上下文查询

【执行示例】
正确流程示例:
1. 步骤1: chatlog(time="2023-04-01~2023-04-30", talker="工作群", keyword="项目进度")
   返回结果: 4月5日、4月12日、4月20日有相关消息
2. 步骤2:
   - 查询1: chatlog(time="2023-04-05/09:30~2023-04-05/10:30", talker="工作群") // 注意没有keyword
   - 查询2: chatlog(time="2023-04-12/14:00~2023-04-12/15:00", talker="工作群") // 注意没有keyword
   - 查询3: chatlog(time="2023-04-20/16:00~2023-04-20/17:00", talker="工作群") // 注意没有keyword
3. 步骤3: 综合分析所有上下文后回答用户

错误流程示例:
- 仅执行步骤1后直接回答
- 步骤2使用time="2023-04-01~2023-04-30"一次性查询
- 步骤2仍然保留keyword或sender参数

【自我检查】回答用户前必须自问:
- 我是否对每个关键时间点都执行了独立的上下文查询?
- 我是否在上下文查询中移除了keyword和sender参数?
- 我是否分析了所有上下文后再回答?
- 如果上述任一问题答案为"否"，则必须纠正流程

返回格式："昵称(ID) 时间\n消息内容\n昵称(ID) 时间\n消息内容"
当查询多个Talker时，返回格式为："昵称(ID)\n[TalkerName(Talker)] 时间\n消息内容"

重要提示：
1. 当用户询问特定时间段内的聊天记录时，必须使用正确的时间格式，特别是包含小时和分钟的查询
2. 对于"今天下午4点到5点聊了啥"这类查询，正确的时间参数格式应为"2023-04-18/16:00~2023-04-18/17:00"
3. 当用户询问具体群聊中某人的聊天记录时，使用"sender"参数
4. 当用户询问包含特定关键词的聊天记录时，使用"keyword"参数`,
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"time": mcp.M{
					"type": "string",
					"description": `指定查询的时间点或时间范围，格式必须严格遵循以下规则：

【单一时间点格式】
- 精确到日："2023-04-18"或"20230418"
- 精确到分钟（必须包含斜杠和冒号）："2023-04-18/14:30"或"20230418/14:30"（表示2023年4月18日14点30分）

【时间范围格式】（使用"~"分隔起止时间）
- 日期范围："2023-04-01~2023-04-18"
- 同一天的时间段："2023-04-18/14:30~2023-04-18/15:45"
  * 表示2023年4月18日14点30分到15点45分之间

【重要提示】包含小时分钟的格式必须使用斜杠和冒号："/"和":"
正确示例："2023-04-18/16:30"（4月18日下午4点30分）
错误示例："2023-04-18 16:30"、"2023-04-18T16:30"

【其他支持的格式】
- 年份："2023"
- 月份："2023-04"或"202304"`,
				},
				"talker": mcp.M{
					"type": "string",
					"description": `指定对话方（联系人或群组）
- 可使用ID、昵称或备注名
- 多个对话方用","分隔，如："张三,李四,工作群"
- 【重要】这是多步查询中唯一应保留的参数`,
				},
				"sender": mcp.M{
					"type": "string",
					"description": `指定群聊中的发送者
- 仅在查询群聊记录时有效
- 多个发送者用","分隔，如："张三,李四"
- 可使用ID、昵称或备注名
【重要】查询特定发送者的消息时：
  1. 第一步：使用sender参数初步定位多个相关消息时间点
  2. 后续步骤：必须移除sender参数，分别查询每个时间点前后的完整对话
  3. 错误示例：对所有找到的消息一次性查询大范围上下文
  4. 正确示例：对每个时间点T分别执行查询"T前后15-30分钟"（不带sender）`,
				},
				"keyword": mcp.M{
					"type": "string",
					"description": `搜索内容中的关键词
- 支持正则表达式匹配
- 【重要】查询特定话题时：
  1. 第一步：使用keyword参数初步定位多个相关消息时间点
  2. 后续步骤：必须移除keyword参数，分别查询每个时间点前后的完整对话
  3. 错误示例：对所有找到的关键词消息一次性查询大范围上下文
  4. 正确示例：对每个时间点T分别执行查询"T前后15-30分钟"（不带keyword）`,
				},
			},
			Required: []string{"time", "talker"},
		},
	}

	ToolCurrentTime = mcp.Tool{
		Name: "current_time",
		Description: `获取当前系统时间，返回RFC3339格式的时间字符串（包含用户本地时区信息）。
使用场景：
- 当用户询问"总结今日聊天记录"、"本周都聊了啥"等当前时间问题
- 当用户提及"昨天"、"上周"、"本月"等相对时间概念，需要确定基准时间点
- 需要执行依赖当前时间的计算（如"上个月5号我们有开会吗"）
返回示例：2025-04-18T21:29:00+08:00
注意：此工具不需要任何输入参数，直接调用即可获取当前时间。`,
		InputSchema: mcp.ToolSchema{
			Type:       "object",
			Properties: mcp.M{},
		},
	}

	ResourceRecentChat = mcp.Resource{
		Name:        "最近会话",
		URI:         "session://recent",
		Description: "获取最近的聊天会话列表",
	}

	ResourceTemplateContact = mcp.ResourceTemplate{
		Name:        "联系人信息",
		URITemplate: "contact://{username}",
		Description: "获取指定联系人的详细信息",
	}

	ResourceTemplateChatRoom = mcp.ResourceTemplate{
		Name:        "群聊信息",
		URITemplate: "chatroom://{roomid}",
		Description: "获取指定群聊的详细信息",
	}

	ResourceTemplateChatlog = mcp.ResourceTemplate{
		Name:        "聊天记录",
		URITemplate: "chatlog://{talker}/{timeframe}?limit,offset",
		Description: "获取与特定联系人或群聊的聊天记录",
	}
)
