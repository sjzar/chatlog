package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb"
)

// ProgressCallback 用于报告导出进度的回调函数
type ProgressCallback func(current, total int)

// ExportMessages 导出消息到文件
func ExportMessages(messages []*model.Message, outputPath string, format string, progress ProgressCallback) error {
	switch format {
	case "json":
		return exportJSON(messages, outputPath, progress)
	case "csv":
		return exportCSV(messages, outputPath, progress)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// GetMessagesForExport 获取要导出的消息
func GetMessagesForExport(db interface {
	GetMessages(startTime, endTime time.Time, talker, sender, content string, offset, limit int) ([]*model.Message, error)
	GetContacts(keyword string, offset, limit int) (*wechatdb.GetContactsResp, error)
}, startTime, endTime time.Time, talker string, onlySelf bool, progress ProgressCallback) ([]*model.Message, error) {
	// 如果没有指定时间范围，默认从2010年到现在
	if startTime.IsZero() {
		startTime, _ = time.Parse("2006-01-02", "2010-01-01")
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	// 如果指定了联系人，直接获取该联系人的消息
	if talker != "" {
		msgs, err := db.GetMessages(startTime, endTime, talker, "", "", 0, 0)
		if err != nil {
			return nil, err
		}
		if onlySelf {
			return filterSelfMessages(msgs), nil
		}
		return msgs, nil
	}

	// 获取所有联系人
	contacts, err := db.GetContacts("", 0, 0)
	if err != nil {
		return nil, err
	}

	// 检查联系人列表是否为空
	if contacts == nil || len(contacts.Items) == 0 {
		return nil, fmt.Errorf("no contacts found")
	}

	// 获取所有聊天记录
	var allMessages []*model.Message
	totalContacts := len(contacts.Items)

	for i, contact := range contacts.Items {
		// 跳过没有用户名的联系人
		if contact.UserName == "" {
			continue
		}

		// 更新进度：获取联系人列表的进度
		if progress != nil {
			progress(i+1, totalContacts)
		}

		// 获取该联系人的聊天记录
		msgs, err := db.GetMessages(startTime, endTime, contact.UserName, "", "", 0, 0)
		if err != nil {
			log.Error().Err(err).Str("contact", contact.UserName).Msg("failed to get messages")
			continue
		}

		// 如果成功获取到消息，添加到列表中
		if len(msgs) > 0 {
			if onlySelf {
				allMessages = append(allMessages, filterSelfMessages(msgs)...)
			} else {
				allMessages = append(allMessages, msgs...)
			}
			log.Info().Str("contact", contact.UserName).Int("count", len(msgs)).Msg("successfully got messages")
		}
	}

	if len(allMessages) == 0 {
		return nil, fmt.Errorf("no messages found")
	}

	return allMessages, nil
}

// filterSelfMessages 过滤出自己发送的消息
func filterSelfMessages(messages []*model.Message) []*model.Message {
	var selfMessages []*model.Message
	for _, msg := range messages {
		if msg.IsSelf {
			selfMessages = append(selfMessages, msg)
		}
	}
	return selfMessages
}

// MessageType 消息类型常量
const (
	TypeText   = 1     // 文本消息
	TypeImage  = 3     // 图片消息
	TypeVoice  = 34    // 语音消息
	TypeVideo  = 43    // 视频消息
	TypeApp    = 49    // 应用消息
	TypeSystem = 10000 // 系统消息
)

// AppMessageSubType 应用消息子类型常量
const (
	SubTypeLink     = 5  // 链接分享
	SubTypeFile     = 6  // 文件
	SubTypeForward  = 19 // 合并转发
	SubTypeMiniApp  = 33 // 小程序
	SubTypeMiniApp2 = 36 // 小程序
	SubTypeVideo    = 51 // 视频号
	SubTypeQuote    = 57 // 引用消息
	SubTypePat      = 62 // 拍一拍
)

// GetMessageTypeDesc 将消息类型转换为可读的中文描述
func GetMessageTypeDesc(msg *model.Message) string {
	// 基础消息类型描述
	typeDesc := map[int64]string{
		TypeText:   "文本消息",
		TypeImage:  "图片消息",
		TypeVoice:  "语音消息",
		TypeVideo:  "视频消息",
		TypeSystem: "系统消息",
	}

	// 如果是基础消息类型，直接返回描述
	if desc, ok := typeDesc[msg.Type]; ok {
		return desc
	}

	// 如果是应用消息，需要根据子类型返回描述
	if msg.Type == TypeApp {
		subTypeDesc := map[int64]string{
			SubTypeLink:     "链接分享",
			SubTypeFile:     "文件",
			SubTypeForward:  "合并转发",
			SubTypeMiniApp:  "小程序",
			SubTypeMiniApp2: "小程序",
			SubTypeVideo:    "视频号",
			SubTypeQuote:    "引用消息",
			SubTypePat:      "拍一拍",
		}

		if desc, ok := subTypeDesc[msg.SubType]; ok {
			return desc
		}
		return fmt.Sprintf("应用消息(%d)", msg.SubType)
	}

	// 未知消息类型
	return fmt.Sprintf("未知类型(%d)", msg.Type)
}

// MessageWithDesc 带描述的消息结构
type MessageWithDesc struct {
	Seq        int64                  `json:"seq"`
	Time       time.Time              `json:"time"`
	Talker     string                 `json:"talker"`
	TalkerName string                 `json:"talkerName"`
	IsChatRoom bool                   `json:"isChatRoom"`
	Sender     string                 `json:"sender"`
	SenderName string                 `json:"senderName"`
	IsSelf     bool                   `json:"isSelf"`
	Type       int64                  `json:"type"`
	SubType    int64                  `json:"subType"`
	Content    string                 `json:"content"`
	Contents   map[string]interface{} `json:"contents,omitempty"`
	TypeDesc   string                 `json:"typeDesc"`
}

func exportJSON(messages []*model.Message, outputPath string, progress ProgressCallback) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := len(messages)
	messagesWithDesc := make([]MessageWithDesc, total)

	// 批量处理消息，每100条更新一次进度
	batchSize := 100
	lastUpdate := time.Now()

	for i, msg := range messages {
		messagesWithDesc[i] = MessageWithDesc{
			Seq:        msg.Seq,
			Time:       msg.Time,
			Talker:     msg.Talker,
			TalkerName: msg.TalkerName,
			IsChatRoom: msg.IsChatRoom,
			Sender:     msg.Sender,
			SenderName: msg.SenderName,
			IsSelf:     msg.IsSelf,
			Type:       msg.Type,
			SubType:    msg.SubType,
			Content:    msg.Content,
			Contents:   msg.Contents,
			TypeDesc:   GetMessageTypeDesc(msg),
		}

		// 每处理batchSize条消息或距离上次更新超过100ms才更新进度
		if progress != nil && (i%batchSize == 0 || time.Since(lastUpdate) > 100*time.Millisecond) {
			progress(i+1, total)
			lastUpdate = time.Now()
		}
	}

	// 确保最后更新一次进度
	if progress != nil {
		progress(total, total)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(messagesWithDesc)
}

func exportCSV(messages []*model.Message, outputPath string, progress ProgressCallback) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	headers := []string{"Time", "Talker", "TalkerName", "Sender", "SenderName", "IsSelf", "Type", "TypeDesc", "Content"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	total := len(messages)
	// 批量处理消息，每100条更新一次进度
	batchSize := 100
	lastUpdate := time.Now()

	// 写入数据
	for i, msg := range messages {
		record := []string{
			msg.Time.Format("2006-01-02 15:04:05"),
			msg.Talker,
			msg.TalkerName,
			msg.Sender,
			msg.SenderName,
			fmt.Sprintf("%v", msg.IsSelf),
			fmt.Sprintf("%d", msg.Type),
			GetMessageTypeDesc(msg),
			msg.Content,
		}
		if err := writer.Write(record); err != nil {
			return err
		}

		// 每处理batchSize条消息或距离上次更新超过100ms才更新进度
		if progress != nil && (i%batchSize == 0 || time.Since(lastUpdate) > 100*time.Millisecond) {
			progress(i+1, total)
			lastUpdate = time.Now()
		}
	}

	// 确保最后更新一次进度
	if progress != nil {
		progress(total, total)
	}

	return nil
}
