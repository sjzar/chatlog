package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/sjzar/chatlog/internal/mcp"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/gin-gonic/gin"
)

type Service struct {
	ctx *ctx.Context
	db  *database.Service

	mcp *mcp.MCP
}

func NewService(ctx *ctx.Context, db *database.Service) *Service {
	return &Service{
		ctx: ctx,
		db:  db,
	}
}

// GetMCP 获取底层MCP实例
func (s *Service) GetMCP() *mcp.MCP {
	return s.mcp
}

// Start 启动MCP服务
func (s *Service) Start() error {
	s.mcp = mcp.NewMCP()
	go s.worker()
	return nil
}

// Stop 停止MCP服务
func (s *Service) Stop() error {
	if s.mcp != nil {
		s.mcp.Close()
	}
	return nil
}

// worker 处理MCP请求
func (s *Service) worker() {
	for {
		select {
		case p, ok := <-s.mcp.ProcessChan:
			if !ok {
				return
			}
			s.processMCP(p.Session, p.Request)
		}
	}
}

func (s *Service) HandleSSE(c *gin.Context) {
	s.mcp.HandleSSE(c)
}

func (s *Service) HandleMessages(c *gin.Context) {
	s.mcp.HandleMessages(c)
}

// processMCP 处理MCP请求
func (s *Service) processMCP(session *mcp.Session, req *mcp.Request) {
	var err error
	switch req.Method {
	case mcp.MethodInitialize:
		err = s.initialize(session, req)
	case mcp.MethodToolsList:
		err = s.sendCustomParams(session, req, mcp.M{"tools": []mcp.Tool{
			ToolContact,
			ToolChatRoom,
			ToolRecentChat,
			ToolChatLog,
			ToolCurrentTime,
		}})
	case mcp.MethodToolsCall:
		err = s.toolsCall(session, req)
	case mcp.MethodPromptsList:
		err = s.sendCustomParams(session, req, mcp.M{"prompts": []mcp.Prompt{}})
	case mcp.MethodResourcesList:
		err = s.sendCustomParams(session, req, mcp.M{"resources": []mcp.Resource{
			ResourceRecentChat,
		}})
	case mcp.MethodResourcesTemplateList:
		err = s.sendCustomParams(session, req, mcp.M{"resourceTemplates": []mcp.ResourceTemplate{
			ResourceTemplateContact,
			ResourceTemplateChatRoom,
			ResourceTemplateChatlog,
		}})
	case mcp.MethodResourcesRead:
		err = s.resourcesRead(session, req)
	case mcp.MethodPing:
		err = s.sendCustomParams(session, req, struct{}{})
	}

	if err != nil {
		session.WriteError(req, err)
	}
}

// initialize 处理初始化请求
func (s *Service) initialize(session *mcp.Session, req *mcp.Request) error {
	initReq, err := parseParams[mcp.InitializeRequest](req.Params)
	if err != nil {
		return fmt.Errorf("解析初始化参数失败: %v", err)
	}
	session.SaveClientInfo(initReq.ClientInfo)

	return session.WriteResponse(req, InitializeResponse)
}

// toolsCall 处理工具调用
func (s *Service) toolsCall(session *mcp.Session, req *mcp.Request) error {
	callReq, err := parseParams[mcp.ToolsCallRequest](req.Params)
	if err != nil {
		return fmt.Errorf("解析工具调用参数失败: %v", err)
	}

	buf := &bytes.Buffer{}
	switch callReq.Name {
	case "query_contact":
		keyword := ""
		if v, ok := callReq.Arguments["keyword"]; ok {
			keyword = v.(string)
		}
		limit := util.MustAnyToInt(callReq.Arguments["limit"])
		offset := util.MustAnyToInt(callReq.Arguments["offset"])
		list, err := s.db.GetContacts(keyword, limit, offset)
		if err != nil {
			return fmt.Errorf("无法获取联系人列表: %v", err)
		}
		buf.WriteString("UserName,Alias,Remark,NickName\n")
		for _, contact := range list.Items {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", contact.UserName, contact.Alias, contact.Remark, contact.NickName))
		}
	case "query_chat_room":
		keyword := ""
		if v, ok := callReq.Arguments["keyword"]; ok {
			keyword = v.(string)
		}
		limit := util.MustAnyToInt(callReq.Arguments["limit"])
		offset := util.MustAnyToInt(callReq.Arguments["offset"])
		list, err := s.db.GetChatRooms(keyword, limit, offset)
		if err != nil {
			return fmt.Errorf("无法获取群聊列表: %v", err)
		}
		buf.WriteString("Name,Remark,NickName,Owner,UserCount\n")
		for _, chatRoom := range list.Items {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d\n", chatRoom.Name, chatRoom.Remark, chatRoom.NickName, chatRoom.Owner, len(chatRoom.Users)))
		}
	case "query_recent_chat":
		keyword := ""
		if v, ok := callReq.Arguments["keyword"]; ok {
			keyword = v.(string)
		}
		limit := util.MustAnyToInt(callReq.Arguments["limit"])
		offset := util.MustAnyToInt(callReq.Arguments["offset"])
		data, err := s.db.GetSessions(keyword, limit, offset)
		if err != nil {
			return fmt.Errorf("无法获取会话列表: %v", err)
		}
		for _, session := range data.Items {
			buf.WriteString(session.PlainText(120))
			buf.WriteString("\n")
		}
	case "chatlog":
		if callReq.Arguments == nil {
			return mcp.ErrInvalidParams
		}
		_time := ""
		if v, ok := callReq.Arguments["time"]; ok {
			_time = v.(string)
		}
		start, end, ok := util.TimeRangeOf(_time)
		if !ok {
			return fmt.Errorf("无法解析时间范围")
		}
		talker := ""
		if v, ok := callReq.Arguments["talker"]; ok {
			talker = v.(string)
		}
		sender := ""
		if v, ok := callReq.Arguments["sender"]; ok {
			sender = v.(string)
		}
		keyword := ""
		if v, ok := callReq.Arguments["keyword"]; ok {
			keyword = v.(string)
		}
		limit := util.MustAnyToInt(callReq.Arguments["limit"])
		offset := util.MustAnyToInt(callReq.Arguments["offset"])
		messages, err := s.db.GetMessages(start, end, talker, sender, keyword, limit, offset)
		if err != nil {
			return fmt.Errorf("无法获取聊天记录: %v", err)
		}
		if len(messages) == 0 {
			buf.WriteString("未找到符合查询条件的聊天记录")
		}
		for _, m := range messages {
			buf.WriteString(m.PlainText(strings.Contains(talker, ","), util.PerfectTimeFormat(start, end), ""))
			buf.WriteString("\n")
		}
	case "current_time":
		buf.WriteString(time.Now().Local().Format(time.RFC3339))
	default:
		return fmt.Errorf("未支持的工具: %s", callReq.Name)
	}

	resp := mcp.ToolsCallResponse{
		Content: []mcp.Content{
			{Type: "text", Text: buf.String()},
		},
		IsError: false,
	}
	return session.WriteResponse(req, resp)
}

// resourcesRead 处理资源读取
func (s *Service) resourcesRead(session *mcp.Session, req *mcp.Request) error {
	readReq, err := parseParams[mcp.ResourcesReadRequest](req.Params)
	if err != nil {
		return fmt.Errorf("解析资源读取参数失败: %v", err)
	}

	u, err := url.Parse(readReq.URI)
	if err != nil {
		return fmt.Errorf("无法解析URI: %v", err)
	}

	buf := &bytes.Buffer{}
	switch u.Scheme {
	case "contact":
		list, err := s.db.GetContacts(u.Host, 0, 0)
		if err != nil {
			return fmt.Errorf("无法获取联系人列表: %v", err)
		}
		buf.WriteString("UserName,Alias,Remark,NickName\n")
		for _, contact := range list.Items {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", contact.UserName, contact.Alias, contact.Remark, contact.NickName))
		}
	case "chatroom":
		list, err := s.db.GetChatRooms(u.Host, 0, 0)
		if err != nil {
			return fmt.Errorf("无法获取群聊列表: %v", err)
		}
		buf.WriteString("Name,Remark,NickName,Owner,UserCount\n")
		for _, chatRoom := range list.Items {
			buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d\n", chatRoom.Name, chatRoom.Remark, chatRoom.NickName, chatRoom.Owner, len(chatRoom.Users)))
		}
	case "session":
		data, err := s.db.GetSessions("", 0, 0)
		if err != nil {
			return fmt.Errorf("无法获取会话列表: %v", err)
		}
		for _, session := range data.Items {
			buf.WriteString(session.PlainText(120))
			buf.WriteString("\n")
		}
	case "chatlog":
		start, end, ok := util.TimeRangeOf(strings.TrimPrefix(u.Path, "/"))
		if !ok {
			return fmt.Errorf("无法解析时间范围")
		}
		limit := util.MustAnyToInt(u.Query().Get("limit"))
		offset := util.MustAnyToInt(u.Query().Get("offset"))
		messages, err := s.db.GetMessages(start, end, u.Host, "", "", limit, offset)
		if err != nil {
			return fmt.Errorf("无法获取聊天记录: %v", err)
		}
		if len(messages) == 0 {
			buf.WriteString("未找到符合查询条件的聊天记录")
		}
		for _, m := range messages {
			buf.WriteString(m.PlainText(strings.Contains(u.Host, ","), util.PerfectTimeFormat(start, end), ""))
			buf.WriteString("\n")
		}
	default:
		return fmt.Errorf("不支持的URI: %s", readReq.URI)
	}

	resp := mcp.ReadingResource{
		Contents: []mcp.ReadingResourceContent{
			{URI: readReq.URI, Text: buf.String()},
		},
	}
	return session.WriteResponse(req, resp)
}

// sendCustomParams 发送自定义参数
func (s *Service) sendCustomParams(session *mcp.Session, req *mcp.Request, params interface{}) error {
	b, err := json.Marshal(mcp.NewResponse(req.ID, params))
	if err != nil {
		return fmt.Errorf("无法序列化响应: %v", err)
	}
	session.Write(b)
	return nil
}

// parseParams 解析参数
func parseParams[T any](params interface{}) (*T, error) {
	if params == nil {
		return nil, errors.New("params is nil")
	}

	// 将 params 重新编码为 JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("无法编码 params: %v", err)
	}

	// 解码到目标结构体
	var result T
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("无法解码为目标结构体: %v", err)
	}

	return &result, nil
}
