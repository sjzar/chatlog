package chatlog

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

// ExportMessages 导出消息到文件
func ExportMessages(messages []*model.Message, outputPath string, format string) error {
	switch format {
	case "json":
		return exportJSON(messages, outputPath)
	case "csv":
		return exportCSV(messages, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// GetMessagesForExport 获取要导出的消息
func GetMessagesForExport(db interface {
	GetMessages(startTime, endTime time.Time, talker, sender, content string, offset, limit int) ([]*model.Message, error)
	GetContacts(keyword string, offset, limit int) (*wechatdb.GetContactsResp, error)
}, startTime, endTime time.Time, talker string, onlySelf bool) ([]*model.Message, error) {
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
	for _, contact := range contacts.Items {
		// 跳过没有用户名的联系人
		if contact.UserName == "" {
			continue
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

// getMessageTypeDesc 将消息类型转换为可读的中文描述
func getMessageTypeDesc(msg *model.Message) string {
	switch msg.Type {
	case 1:
		return "文本消息"
	case 3:
		return "图片消息"
	case 34:
		return "语音消息"
	case 43:
		return "视频消息"
	case 49:
		switch msg.SubType {
		case 5:
			return "链接分享"
		case 6:
			return "文件"
		case 19:
			return "合并转发"
		case 33, 36:
			return "小程序"
		case 51:
			return "视频号"
		case 57:
			return "引用消息"
		case 62:
			return "拍一拍"
		default:
			return fmt.Sprintf("应用消息(%d)", msg.SubType)
		}
	case 10000:
		return "系统消息"
	default:
		return fmt.Sprintf("未知类型(%d)", msg.Type)
	}
}

func exportJSON(messages []*model.Message, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个新的消息列表，添加类型描述
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

	messagesWithDesc := make([]MessageWithDesc, len(messages))
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
			TypeDesc:   getMessageTypeDesc(msg),
		}
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(messagesWithDesc)
}

func exportCSV(messages []*model.Message, outputPath string) error {
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

	// 写入数据
	for _, msg := range messages {
		record := []string{
			msg.Time.Format("2006-01-02 15:04:05"),
			msg.Talker,
			msg.TalkerName,
			msg.Sender,
			msg.SenderName,
			fmt.Sprintf("%v", msg.IsSelf),
			fmt.Sprintf("%d", msg.Type),
			getMessageTypeDesc(msg),
			msg.Content,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
