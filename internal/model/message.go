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

const (
	// MessageTypeText 文本
	MessageTypeText = 1

	// MessageTypeImage 图片
	MessageTypeImage = 3

	// MessageTypeVoice 语音
	MessageTypeVoice = 34

	// MessageTypeCard 名片
	MessageTypeCard = 42

	// MessageTypeVideo 视频
	MessageTypeVideo = 43

	// MessageTypeAnimation 动画表情
	MessageTypeAnimation = 47

	// MessageTypeLocation 位置
	MessageTypeLocation = 48

	// MessageTypeShare 分享
	MessageTypeShare = 49

	// MessageTypeVOIP 语音通话
	MessageTypeVOIP = 50

	// MessageTypeSystem 系统
	MessageTypeSystem = 10000
)

const (
	// MessageSubTypeText 文本
	MessageSubTypeText = 1

	// MessageSubTypeLink 链接分享
	MessageSubTypeLink = 4

	// MessageSubTypeLink2 链接分享
	MessageSubTypeLink2 = 5

	// MessageSubTypeFile 文件
	MessageSubTypeFile = 6

	// MessageSubTypeGIF 动图
	MessageSubTypeGIF = 8

	// MessageSubTypeMergeForward 合并转发
	MessageSubTypeMergeForward = 19

	// MessageSubTypeNote 笔记
	MessageSubTypeNote = 24

	// MessageSubTypeMiniProgram 小程序
	MessageSubTypeMiniProgram = 33

	// MessageSubTypeMiniProgram2 小程序
	MessageSubTypeMiniProgram2 = 36

	// MessageSubTypeChannel 视频号
	MessageSubTypeChannel = 51

	// MessageSubTypeQuote 引用
	MessageSubTypeQuote = 57

	// MessageSubTypePat 拍一拍
	MessageSubTypePat = 62

	// MessageSubTypeChannelLive 视频号直播
	MessageSubTypeChannelLive = 63

	// MessageSubTypeChatRoomNotice 群公告
	MessageSubTypeChatRoomNotice = 87

	// MessageSubTypeMusic 音乐
	MessageSubTypeMusic = 92

	// MessageSubTypePay 转账
	MessageSubTypePay = 2000

	// MessageSubTypeRedEnvelope 红包
	MessageSubTypeRedEnvelope = 2001

	// MessageSubTypeRedEnvelopeCover 红包封面
	MessageSubTypeRedEnvelopeCover = 2003
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

	if m.Type == MessageTypeSystem {
		m.Sender = "系统消息"
		m.SenderName = ""
		var sysMsg SysMsg
		if err := xml.Unmarshal([]byte(data), &sysMsg); err != nil {
			m.Content = data
			return nil
		}
		if Debug {
			m.SysMsg = &sysMsg
		}
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
	case MessageTypeImage:
		m.Contents["md5"] = msg.Image.MD5
	case MessageTypeVideo:
		if msg.Video.Md5 != "" {
			m.Contents["md5"] = msg.Video.Md5
		}
		if msg.Video.RawMd5 != "" {
			m.Contents["rawmd5"] = msg.Video.RawMd5
		}
	case MessageTypeAnimation:
		m.Contents["cdnurl"] = msg.Emoji.CdnURL
	case MessageTypeLocation:
		m.Contents["x"] = msg.Location.X
		m.Contents["y"] = msg.Location.Y
		m.Contents["label"] = msg.Location.Label
		m.Contents["cityname"] = msg.Location.CityName
	case MessageTypeShare:
		m.SubType = int64(msg.App.Type)
		switch m.SubType {
		case MessageSubTypeText, MessageSubTypeLink, MessageSubTypeLink2:
			// 链接
			m.Contents["title"] = msg.App.Title
			m.Contents["desc"] = msg.App.Des
			m.Contents["url"] = msg.App.URL
		case MessageSubTypeFile:
			// 文件
			m.Contents["title"] = msg.App.Title
			m.Contents["md5"] = msg.App.MD5
		case MessageSubTypeMergeForward, MessageSubTypeNote, MessageSubTypeChatRoomNotice:
			// 合并转发 & 笔记
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
		case MessageSubTypeMiniProgram, MessageSubTypeMiniProgram2:
			// 小程序
			m.Contents["title"] = msg.App.SourceDisplayName
			m.Contents["url"] = msg.App.URL
		case MessageSubTypeChannel:
			// 视频号
			if msg.App.FinderFeed == nil {
				break
			}
			m.Contents["title"] = strings.TrimSpace(strings.ReplaceAll(msg.App.FinderFeed.Desc, "\n", " "))
			if len(msg.App.FinderFeed.MediaList.Media) > 0 {
				m.Contents["url"] = msg.App.FinderFeed.MediaList.Media[0].URL
			}
		case MessageSubTypeQuote:
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
		case MessageSubTypePat:
			// 拍一拍
			if msg.App.PatMsg != nil {
				if len(msg.App.PatMsg.Records.Record) != 0 {
					m.Sender = msg.App.PatMsg.Records.Record[0].FromUser
					m.Content = msg.App.PatMsg.Records.Record[0].Templete
				}
			}
			if msg.App.PatInfo != nil {
				m.Content = msg.App.Title
			}
		case MessageSubTypeChannelLive:
			// 视频号直播
			if msg.App.FinderLive == nil {
				break
			}
			m.Contents["title"] = msg.App.FinderLive.Desc
		case MessageSubTypeMusic:
			// 音乐
			m.Contents["title"] = msg.App.Title
			m.Contents["desc"] = msg.App.Des
			m.Contents["url"] = msg.App.URL
		case MessageSubTypePay:
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
	case MessageTypeText:
		return m.Content
	case MessageTypeImage:
		keylist := make([]string, 0)
		if m.Contents["md5"] != nil {
			if md5, ok := m.Contents["md5"].(string); ok {
				keylist = append(keylist, md5)
			}
		}
		if m.Contents["path"] != nil {
			if path, ok := m.Contents["path"].(string); ok {
				keylist = append(keylist, path)
			}
		}
		if m.Contents["thumbpath"] != nil {
			if thumbpath, ok := m.Contents["thumbpath"].(string); ok {
				keylist = append(keylist, thumbpath)
			}
		}
		return fmt.Sprintf("![图片](http://%s/image/%s)", m.Contents["host"], strings.Join(keylist, ","))
	case MessageTypeVoice:
		if voice, ok := m.Contents["voice"]; ok {
			return fmt.Sprintf("[语音](http://%s/voice/%s)", m.Contents["host"], voice)
		}
		return "[语音]"
	case MessageTypeCard:
		return "[名片]"
	case MessageTypeVideo:
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
		if m.Contents["path"] != nil {
			if path, ok := m.Contents["path"].(string); ok {
				keylist = append(keylist, path)
			}
		}
		return fmt.Sprintf("![视频](http://%s/video/%s)", m.Contents["host"], strings.Join(keylist, ","))
	case MessageTypeAnimation:
		if m.Contents["cdnurl"] != nil {
			if cdnURL, ok := m.Contents["cdnurl"].(string); ok {
				return fmt.Sprintf("![动画表情](%s)", cdnURL)
			}
		}
		return "[动画表情]"
	case MessageTypeLocation:
		keylist := make([]string, 0)
		for _, key := range []string{"label", "cityname", "x", "y"} {
			if m.Contents[key] != nil {
				if value, ok := m.Contents[key].(string); ok {
					keylist = append(keylist, value)
				}
			}
		}
		return fmt.Sprintf("[位置|%s]", strings.Join(keylist, "|"))
	case MessageTypeShare:
		switch m.SubType {
		case MessageSubTypeText:
			return fmt.Sprintf("[链接|%s](%s)", m.Contents["title"], m.Contents["desc"])
		case MessageSubTypeLink, MessageSubTypeLink2:
			return fmt.Sprintf("[链接|%s](%s)", m.Contents["title"], m.Contents["url"])
		case MessageSubTypeFile:
			return fmt.Sprintf("[文件|%s](http://%s/file/%s)", m.Contents["title"], m.Contents["host"], m.Contents["md5"])
		case MessageSubTypeGIF:
			return "[GIF表情]"
		case MessageSubTypeMergeForward:
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
			return recordInfo.String("合并转发", "", host)
		case MessageSubTypeNote:
			_recordInfo, ok := m.Contents["recordInfo"]
			if !ok {
				return "[笔记]"
			}
			recordInfo, ok := _recordInfo.(*RecordInfo)
			if !ok {
				return "[笔记]"
			}
			host := ""
			if m.Contents["host"] != nil {
				host = m.Contents["host"].(string)
			}
			return recordInfo.String("笔记", "", host)
		case MessageSubTypeMiniProgram, MessageSubTypeMiniProgram2:
			if m.Contents["title"] == "" {
				return "[小程序]"
			}
			return fmt.Sprintf("[小程序|%s](%s)", m.Contents["title"], m.Contents["url"])
		case MessageSubTypeChannel:
			if m.Contents["title"] == "" {
				return "[视频号]"
			} else {
				return fmt.Sprintf("[视频号|%s](%s)", m.Contents["title"], m.Contents["url"])
			}
		case MessageSubTypeQuote:
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
		case MessageSubTypePat:
			return m.Content
		case MessageSubTypeChannelLive:
			if m.Contents["title"] != nil {
				return fmt.Sprintf("[视频号直播|%s]", m.Contents["title"])
			}
			return "[视频号直播]"
		case MessageSubTypeChatRoomNotice:
			_recordInfo, ok := m.Contents["recordInfo"]
			if !ok {
				return "[群公告]"
			}
			recordInfo, ok := _recordInfo.(*RecordInfo)
			if !ok {
				return "[群公告]"
			}
			host := ""
			if m.Contents["host"] != nil {
				host = m.Contents["host"].(string)
			}
			return recordInfo.String("群公告", "", host)
		case MessageSubTypeMusic:
			return fmt.Sprintf("[音乐|%s](%s)", m.Contents["title"], m.Contents["url"])
		case MessageSubTypePay:
			return m.Content
		case MessageSubTypeRedEnvelope:
			return "[红包]"
		case MessageSubTypeRedEnvelopeCover:
			return "[红包封面]"
		default:
			return "[分享]"
		}
	case MessageTypeVOIP:
		return "[语音通话]"
	case MessageTypeSystem:
		return m.Content
	default:
		content := m.Content
		if len(content) > 120 {
			content = content[:120] + "<...>"
		}
		return fmt.Sprintf("Type: %d Content: %s", m.Type, content)
	}
}

func (m *Message) CSV(host string) []string {
	m.SetContent("host", host)
	return []string{
		m.Time.Format("2006-01-02 15:04:05"),
		m.SenderName,
		m.Sender,
		m.TalkerName,
		m.Talker,
		m.PlainTextContent(),
	}
}
