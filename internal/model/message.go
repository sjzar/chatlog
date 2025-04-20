package model

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/sjzar/chatlog/pkg/util"
)

var Debug = false

const (
	WeChatV3       = "wechatv3"
	WeChatV4       = "wechatv4"
	WeChatDarwinV3 = "wechatdarwinv3"
)

type Message struct {
	Version    string                 `json:"-"`                  // 消息版本，内部判断
	Seq        int64                  `json:"seq"`                // 消息序号，10位时间戳 + 3位序号
	Time       time.Time              `json:"time"`               // 消息创建时间，10位时间戳
	Talker     string                 `json:"talker"`             // 聊天对象，微信 ID or 群 ID
	TalkerName string                 `json:"talkerName"`         // 聊天对象名称
	IsChatRoom bool                   `json:"isChatRoom"`         // 是否为群聊消息
	Sender     string                 `json:"sender"`             // 发送人，微信 ID
	SenderName string                 `json:"senderName"`         // 发送人名称
	IsSelf     bool                   `json:"isSelf"`             // 是否为自己发送的消息
	Type       int64                  `json:"type"`               // 消息类型
	SubType    int64                  `json:"subType"`            // 消息子类型
	Content    string                 `json:"content"`            // 消息内容，文字聊天内容
	Contents   map[string]interface{} `json:"contents,omitempty"` // 消息内容，多媒体消息，采用更灵活的记录方式

	// Debug Info
	MediaMsg *MediaMsg `json:"mediaMsg,omitempty"` // 原始多媒体消息，XML 格式
	SysMsg   *SysMsg   `json:"sysMsg,omitempty"`   // 原始系统消息，XML 格式
}

func (m *Message) ParseMediaInfo(data string) error {

	m.Type, m.SubType = util.SplitInt64ToTwoInt32(m.Type)

	if m.Type == 1 {
		m.Content = data
		return nil
	}

	if m.Type == 10000 {
		var sysMsg SysMsg
		if err := xml.Unmarshal([]byte(data), &sysMsg); err != nil {
			m.Content = data
			return nil
		}
		if Debug {
			m.SysMsg = &sysMsg
		}
		m.Sender = "系统消息"
		m.SenderName = ""
		m.Content = sysMsg.String()
		return nil
	}

	var msg MediaMsg
	err := xml.Unmarshal([]byte(data), &msg)
	if err != nil {
		return err
	}

	if m.Contents == nil {
		m.Contents = make(map[string]interface{})
	}

	if Debug {
		m.MediaMsg = &msg
	}

	switch m.Type {
	case 3:
		m.Contents["md5"] = msg.Image.MD5
	case 43:
		if msg.Video.Md5 != "" {
			m.Contents["md5"] = msg.Video.Md5
		}
		if msg.Video.RawMd5 != "" {
			m.Contents["rawmd5"] = msg.Video.RawMd5
		}
	case 49:
		m.SubType = int64(msg.App.Type)
		switch m.SubType {
		case 5:
			// 链接
			m.Contents["title"] = msg.App.Title
			m.Contents["url"] = msg.App.URL
		case 6:
			// 文件
			m.Contents["title"] = msg.App.Title
			m.Contents["md5"] = msg.App.MD5
		case 19:
			// 合并转发
			m.Contents["title"] = msg.App.Title
			m.Contents["desc"] = msg.App.Des
			if msg.App.RecordItem == nil {
				break
			}
			recordInfo := &RecordInfo{}
			err := xml.Unmarshal([]byte(msg.App.RecordItem.CDATA), recordInfo)
			if err != nil {
				return err
			}
			m.Contents["recordInfo"] = recordInfo
		case 33, 36:
			// 小程序
			m.Contents["title"] = msg.App.SourceDisplayName
			m.Contents["url"] = msg.App.URL
		case 51:
			// 视频号
			if msg.App.FinderFeed == nil {
				break
			}
			m.Contents["title"] = msg.App.FinderFeed.Desc
			if len(msg.App.FinderFeed.MediaList.Media) > 0 {
				m.Contents["url"] = msg.App.FinderFeed.MediaList.Media[0].URL
			}
		case 57:
			// 引用
			m.Content = msg.App.Title
			if msg.App.ReferMsg == nil {
				break
			}
			subMsg := &Message{
				Type:       int64(msg.App.ReferMsg.Type),
				Time:       time.Unix(msg.App.ReferMsg.CreateTime, 0),
				Sender:     msg.App.ReferMsg.ChatUsr,
				SenderName: msg.App.ReferMsg.DisplayName,
			}
			if subMsg.Sender == "" {
				subMsg.Sender = msg.App.ReferMsg.FromUsr
			}
			if err := subMsg.ParseMediaInfo(msg.App.ReferMsg.Content); err != nil {
				break
			}
			m.Contents["refer"] = subMsg
		case 62:
			// 拍一拍
			if msg.App.PatMsg == nil {
				break
			}
			if len(msg.App.PatMsg.Records.Record) == 0 {
				break
			}
			m.Sender = msg.App.PatMsg.Records.Record[0].FromUser
			m.Content = msg.App.PatMsg.Records.Record[0].Templete
		case 2000:
			// 微信转账
			if msg.App.WCPayInfo == nil {
				break
			}
			// 1 实时转账
			// 3 实时转账收钱回执
			// 4 转账退还回执
			// 5 非实时转账收钱回执
			// 7 非实时转账
			_type := ""
			switch msg.App.WCPayInfo.PaySubType {
			case 1, 7:
				_type = "发送 "
			case 3, 5:
				_type = "接收 "
			case 4:
				_type = "退还 "
			}
			payMemo := ""
			if len(msg.App.WCPayInfo.PayMemo) > 0 {
				payMemo = "(" + msg.App.WCPayInfo.PayMemo + ")"
			}
			m.Content = fmt.Sprintf("[转账|%s%s]%s", _type, msg.App.WCPayInfo.FeeDesc, payMemo)
		}
	}

	return nil
}

func (m *Message) SetContent(key string, value interface{}) {
	if m.Contents == nil {
		m.Contents = make(map[string]interface{})
	}
	m.Contents[key] = value
}

func (m *Message) PlainText(showChatRoom bool, timeFormat string, host string) string {

	if timeFormat == "" {
		timeFormat = "01-02 15:04:05"
	}

	m.SetContent("host", host)

	buf := strings.Builder{}

	sender := m.Sender
	if m.IsSelf {
		sender = "我"
	}
	if m.SenderName != "" {
		buf.WriteString(m.SenderName)
		buf.WriteString("(")
		buf.WriteString(sender)
		buf.WriteString(")")
	} else {
		buf.WriteString(sender)
	}
	buf.WriteString(" ")

	if m.IsChatRoom && showChatRoom {
		buf.WriteString("[")
		if m.TalkerName != "" {
			buf.WriteString(m.TalkerName)
			buf.WriteString("(")
			buf.WriteString(m.Talker)
			buf.WriteString(")")
		} else {
			buf.WriteString(m.Talker)
		}
		buf.WriteString("] ")
	}

	buf.WriteString(m.Time.Format(timeFormat))
	buf.WriteString("\n")

	buf.WriteString(m.PlainTextContent())
	buf.WriteString("\n")

	return buf.String()
}

func (m *Message) PlainTextContent() string {
	switch m.Type {
	case 1:
		return m.Content
	case 3:
		keylist := make([]string, 0)
		if m.Contents["md5"] != nil {
			if md5, ok := m.Contents["md5"].(string); ok {
				keylist = append(keylist, md5)
			}
		}
		if m.Contents["imgfile"] != nil {
			if imgfile, ok := m.Contents["imgfile"].(string); ok {
				keylist = append(keylist, imgfile)
			}
		}
		if m.Contents["thumb"] != nil {
			if thumb, ok := m.Contents["thumb"].(string); ok {
				keylist = append(keylist, thumb)
			}
		}
		return fmt.Sprintf("![图片](http://%s/image/%s)", m.Contents["host"], strings.Join(keylist, ","))
	case 34:
		if voice, ok := m.Contents["voice"]; ok {
			return fmt.Sprintf("[语音](http://%s/voice/%s)", m.Contents["host"], voice)
		}
		return "[语音]"
	case 42:
		return "[名片]"
	case 43:
		keylist := make([]string, 0)
		if m.Contents["md5"] != nil {
			if md5, ok := m.Contents["md5"].(string); ok {
				keylist = append(keylist, md5)
			}
		}
		if m.Contents["rawmd5"] != nil {
			if rawmd5, ok := m.Contents["rawmd5"].(string); ok {
				keylist = append(keylist, rawmd5)
			}
		}
		if m.Contents["videofile"] != nil {
			if videofile, ok := m.Contents["videofile"].(string); ok {
				keylist = append(keylist, videofile)
			}
		}
		if m.Contents["thumb"] != nil {
			if thumb, ok := m.Contents["thumb"].(string); ok {
				keylist = append(keylist, thumb)
			}
		}
		return fmt.Sprintf("![视频](http://%s/video/%s)", m.Contents["host"], strings.Join(keylist, ","))
	case 47:
		return "[动画表情]"
	case 49:
		switch m.SubType {
		case 5:
			return fmt.Sprintf("[链接|%s](%s)", m.Contents["title"], m.Contents["url"])
		case 6:
			return fmt.Sprintf("[文件|%s](http://%s/file/%s)", m.Contents["title"], m.Contents["host"], m.Contents["md5"])
		case 8:
			return "[GIF表情]"
		case 19:
			_recordInfo, ok := m.Contents["recordInfo"]
			if !ok {
				return "[合并转发]"
			}
			recordInfo, ok := _recordInfo.(*RecordInfo)
			if !ok {
				return "[合并转发]"
			}
			host := ""
			if m.Contents["host"] != nil {
				host = m.Contents["host"].(string)
			}
			return recordInfo.String("", host)
		case 33, 36:
			if m.Contents["title"] == "" {
				return "[小程序]"
			}
			return fmt.Sprintf("[小程序|%s](%s)", m.Contents["title"], m.Contents["url"])
		case 51:
			if m.Contents["title"] == "" {
				return "[视频号]"
			} else {
				return fmt.Sprintf("[视频号|%s](%s)", m.Contents["title"], m.Contents["url"])
			}
		case 57:
			_refer, ok := m.Contents["refer"]
			if !ok {
				if m.Content == "" {
					return "[引用]"
				}
				return "> [引用]\n" + m.Content
			}
			refer, ok := _refer.(*Message)
			if !ok {
				if m.Content == "" {
					return "[引用]"
				}
				return "> [引用]\n" + m.Content
			}
			buf := strings.Builder{}
			host := ""
			if m.Contents["host"] != nil {
				host = m.Contents["host"].(string)
			}
			referContent := refer.PlainText(false, "", host)
			for _, line := range strings.Split(referContent, "\n") {
				if line == "" {
					continue
				}
				buf.WriteString("> ")
				buf.WriteString(line)
				buf.WriteString("\n")
			}
			buf.WriteString(m.Content)
			return buf.String()
		case 62:
			return m.Content
		case 63:
			return "[视频号]"
		case 87:
			return "[群公告]"
		case 2000:
			return m.Content
		case 2001:
			return "[红包]"
		case 2003:
			return "[红包封面]"
		default:
			return "[分享]"
		}
	case 50:
		return "[语音通话]"
	case 10000:
		return m.Content
	default:
		content := m.Content
		if len(content) > 120 {
			content = content[:120] + "<...>"
		}
		return fmt.Sprintf("Type: %d Content: %s", m.Type, content)
	}
}
