package http

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/pkg/util"
	"github.com/sjzar/chatlog/pkg/version"
)

func (s *Service) initMCPServer() {
	s.mcpServer = server.NewMCPServer(conf.AppName, version.Version)
	s.mcpServer.AddTool(ContactTool, s.handleMCPContact)
	s.mcpServer.AddTool(ChatRoomTool, s.handleMCPChatRoom)
	s.mcpServer.AddTool(RecentChatTool, s.handleMCPRecentChat)
	s.mcpServer.AddTool(ChatLogTool, s.handleMCPChatLog)
	s.mcpServer.AddTool(CurrentTimeTool, s.handleMCPCurrentTime)
	s.mcpSSEServer = server.NewSSEServer(s.mcpServer)
	s.mcpStreamableServer = server.NewStreamableHTTPServer(s.mcpServer)
}

var ContactTool = mcp.NewTool(
	"query_contact",
	mcp.WithDescription(`查询用户的联系人信息。可以通过姓名、备注名或ID进行查询，返回匹配的联系人列表。当用户询问某人的联系方式、想了解联系人信息或需要查找特定联系人时使用此工具。参数为空时，将返回联系人列表`),
	mcp.WithString("keyword", mcp.Description("联系人的搜索关键词，可以是姓名、备注名或ID。")),
)

var ChatRoomTool = mcp.NewTool(
	"query_chat_room",
	mcp.WithDescription(`查询用户参与的群聊信息。可以通过群名称、群ID或相关关键词进行查询，返回匹配的群聊列表。当用户询问群聊信息、想了解某个群的详情或需要查找特定群聊时使用此工具。`),
	mcp.WithString("keyword", mcp.Description("群聊的搜索关键词，可以是群名称、群ID或相关描述")),
)

var RecentChatTool = mcp.NewTool(
	"query_recent_chat",
	mcp.WithDescription(`查询最近会话列表，包括个人聊天和群聊。当用户想了解最近的聊天记录、查看最近联系过的人或群组时使用此工具。不需要参数，直接返回最近的会话列表。`),
)

var ChatLogTool = mcp.NewTool(
	"query_chat_log",
	mcp.WithDescription(`检索历史聊天记录，可根据时间、对话方、发送者和关键词等条件进行精确查询。当用户需要查找特定信息或想了解与某人/某群的历史交流时使用此工具。

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
4. 当用户询问包含特定关键词的聊天记录时，使用"keyword"参数`),
	mcp.WithString("time", mcp.Description(`指定查询的时间点或时间范围，格式必须严格遵循以下规则：

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
- 月份："2023-04"或"202304"`), mcp.Required()),
	mcp.WithString("talker", mcp.Description(`指定对话方（联系人或群组）
- 可使用ID、昵称或备注名
- 多个对话方用","分隔，如："张三,李四,工作群"
- 【重要】这是多步查询中唯一应保留的参数`), mcp.Required()),
	mcp.WithString("sender", mcp.Description(`指定群聊中的发送者
- 仅在查询群聊记录时有效
- 多个发送者用","分隔，如："张三,李四"
- 可使用ID、昵称或备注名
【重要】查询特定发送者的消息时：
1. 第一步：使用sender参数初步定位多个相关消息时间点
2. 后续步骤：必须移除sender参数，分别查询每个时间点前后的完整对话
3. 错误示例：对所有找到的消息一次性查询大范围上下文
4. 正确示例：对每个时间点T分别执行查询"T前后15-30分钟"（不带sender）`)),
	mcp.WithString("keyword", mcp.Description(`搜索内容中的关键词
- 支持正则表达式匹配
- 【重要】查询特定话题时：
1. 第一步：使用keyword参数初步定位多个相关消息时间点
2. 后续步骤：必须移除keyword参数，分别查询每个时间点前后的完整对话
3. 错误示例：对所有找到的关键词消息一次性查询大范围上下文
4. 正确示例：对每个时间点T分别执行查询"T前后15-30分钟"（不带keyword）`)),
)

var CurrentTimeTool = mcp.NewTool(
	"current_time",
	mcp.WithDescription(`获取当前系统时间，返回RFC3339格式的时间字符串（包含用户本地时区信息）。
使用场景：
- 当用户询问"总结今日聊天记录"、"本周都聊了啥"等当前时间问题
- 当用户提及"昨天"、"上周"、"本月"等相对时间概念，需要确定基准时间点
- 需要执行依赖当前时间的计算（如"上个月5号我们有开会吗"）
返回示例：2025-04-18T21:29:00+08:00
注意：此工具不需要任何输入参数，直接调用即可获取当前时间。`),
)

type ContactRequest struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

func (s *Service) handleMCPContact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var req ContactRequest
	if err := request.BindArguments(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind arguments")
		log.Error().Interface("request", request.GetRawArguments()).Msg("Failed to bind arguments")
		return errors.ErrMCPTool(err), nil
	}

	list, err := s.db.GetContacts(req.Keyword, req.Limit, req.Offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get contacts")
		return errors.ErrMCPTool(err), nil
	}
	buf := &bytes.Buffer{}
	buf.WriteString("UserName,Alias,Remark,NickName\n")
	for _, contact := range list.Items {
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", contact.UserName, contact.Alias, contact.Remark, contact.NickName))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: buf.String(),
			},
		},
	}, nil
}

type ChatRoomRequest struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

func (s *Service) handleMCPChatRoom(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	var req ChatRoomRequest
	if err := request.BindArguments(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind arguments")
		log.Error().Interface("request", request.GetRawArguments()).Msg("Failed to bind arguments")
		return errors.ErrMCPTool(err), nil
	}

	list, err := s.db.GetChatRooms(req.Keyword, req.Limit, req.Offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get chat rooms")
		return errors.ErrMCPTool(err), nil
	}
	buf := &bytes.Buffer{}
	buf.WriteString("Name,Remark,NickName,Owner,UserCount\n")
	for _, chatRoom := range list.Items {
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d\n", chatRoom.Name, chatRoom.Remark, chatRoom.NickName, chatRoom.Owner, len(chatRoom.Users)))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: buf.String(),
			},
		},
	}, nil
}

type RecentChatRequest struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

func (s *Service) handleMCPRecentChat(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	var req RecentChatRequest
	if err := request.BindArguments(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind arguments")
		log.Error().Interface("request", request.GetRawArguments()).Msg("Failed to bind arguments")
		return errors.ErrMCPTool(err), nil
	}

	data, err := s.db.GetSessions(req.Keyword, req.Limit, req.Offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get sessions")
		return errors.ErrMCPTool(err), nil
	}
	buf := &bytes.Buffer{}
	for _, session := range data.Items {
		buf.WriteString(session.PlainText(120))
		buf.WriteString("\n")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: buf.String(),
			},
		},
	}, nil
}

type ChatLogRequest struct {
	Time    string `form:"time"`
	Talker  string `form:"talker"`
	Sender  string `form:"sender"`
	Keyword string `form:"keyword"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
	Format  string `form:"format"`
}

func (s *Service) handleMCPChatLog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	var req ChatLogRequest
	if err := request.BindArguments(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind arguments")
		log.Error().Interface("request", request.GetRawArguments()).Msg("Failed to bind arguments")
		return errors.ErrMCPTool(err), nil
	}

	var err error
	start, end, ok := util.TimeRangeOf(req.Time)
	if !ok {
		log.Error().Err(err).Msg("Failed to get messages")
		return errors.ErrMCPTool(err), nil
	}
	if req.Limit < 0 {
		req.Limit = 0
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	messages, err := s.db.GetMessages(start, end, req.Talker, req.Sender, req.Keyword, req.Limit, req.Offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get messages")
		return errors.ErrMCPTool(err), nil
	}

	buf := &bytes.Buffer{}
	if len(messages) == 0 {
		buf.WriteString("未找到符合查询条件的聊天记录")
	}
	for _, m := range messages {
		buf.WriteString(m.PlainText(strings.Contains(req.Talker, ","), util.PerfectTimeFormat(start, end), ""))
		buf.WriteString("\n")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: buf.String(),
			},
		},
	}, nil
}

func (s *Service) handleMCPCurrentTime(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: time.Now().Local().Format(time.RFC3339),
			},
		},
	}, nil
}
