package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/model/wxproto"
	"github.com/sjzar/chatlog/pkg/util/lz4"
	"google.golang.org/protobuf/proto"
)

// CREATE TABLE MSG (
// localId INTEGER PRIMARY KEY AUTOINCREMENT,
// TalkerId INT DEFAULT 0,
// MsgSvrID INT,
// Type INT,
// SubType INT,
// IsSender INT,
// CreateTime INT,
// Sequence INT DEFAULT 0,
// StatusEx INT DEFAULT 0,
// FlagEx INT,
// Status INT,
// MsgServerSeq INT,
// MsgSequence INT,
// StrTalker TEXT,
// StrContent TEXT,
// DisplayContent TEXT,
// Reserved0 INT DEFAULT 0,
// Reserved1 INT DEFAULT 0,
// Reserved2 INT DEFAULT 0,
// Reserved3 INT DEFAULT 0,
// Reserved4 TEXT,
// Reserved5 TEXT,
// Reserved6 TEXT,
// CompressContent BLOB,
// BytesExtra BLOB,
// BytesTrans BLOB
// )
type MessageV3 struct {
	MsgSvrID        int64  `json:"MsgSvrID"`        // 消息 ID
	Sequence        int64  `json:"Sequence"`        // 消息序号，10位时间戳 + 3位序号
	CreateTime      int64  `json:"CreateTime"`      // 消息创建时间，10位时间戳
	StrTalker       string `json:"StrTalker"`       // 聊天对象，微信 ID or 群 ID
	IsSender        int    `json:"IsSender"`        // 是否为发送消息，0 接收消息，1 发送消息
	Type            int64  `json:"Type"`            // 消息类型
	SubType         int    `json:"SubType"`         // 消息子类型
	StrContent      string `json:"StrContent"`      // 消息内容，文字聊天内容 或 XML
	CompressContent []byte `json:"CompressContent"` // 非文字聊天内容，如图片、语音、视频等
	BytesExtra      []byte `json:"BytesExtra"`      // protobuf 额外数据，记录群聊发送人等信息
}

func (m *MessageV3) Wrap() *Message {

	_m := &Message{
		Seq:        m.Sequence,
		Time:       time.Unix(m.CreateTime, 0),
		Talker:     m.StrTalker,
		IsChatRoom: strings.HasSuffix(m.StrTalker, "@chatroom"),
		IsSelf:     m.IsSender == 1,
		Type:       m.Type,
		SubType:    int64(m.SubType),
		Content:    m.StrContent,
		Version:    WeChatV3,
	}

	if !_m.IsChatRoom && !_m.IsSelf {
		_m.Sender = m.StrTalker
	}

	if _m.Type == 49 {
		b, err := lz4.Decompress(m.CompressContent)
		if err == nil {
			_m.Content = string(b)
		}
	}

	_m.ParseMediaInfo(_m.Content)

	// 语音消息
	if _m.Type == 34 {
		_m.Contents["voice"] = fmt.Sprint(m.MsgSvrID)
	}

	if len(m.BytesExtra) != 0 {
		if bytesExtra := ParseBytesExtra(m.BytesExtra); bytesExtra != nil {
			if _m.IsChatRoom {
				_m.Sender = bytesExtra[1]
			}

			// 图片处理
			if _m.Type == MessageTypeImage {
				if len(bytesExtra[4]) > 0 {
					_m.Contents["path"] = ParseBytesExtraPath(bytesExtra[4])
				}
				if len(bytesExtra[3]) > 0 {
					_m.Contents["thumbpath"] = ParseBytesExtraPath(bytesExtra[3])
				}
			}

			// FIXME xml 中的 md5 数据无法匹配到 hardlink 记录，所以直接用 proto 数据
			if _m.Type == MessageTypeVideo {
				if len(bytesExtra[4]) > 0 {
					_m.Contents["path"] = ParseBytesExtraPath(bytesExtra[4])
				}
			}
		}
	}

	return _m
}

// ParseBytesExtra 解析额外数据
// 按需解析
func ParseBytesExtra(b []byte) map[int]string {
	var pbMsg wxproto.BytesExtra
	if err := proto.Unmarshal(b, &pbMsg); err != nil {
		return nil
	}
	if pbMsg.Items == nil {
		return nil
	}

	ret := make(map[int]string, len(pbMsg.Items))
	for _, item := range pbMsg.Items {
		ret[int(item.Type)] = item.Value
	}

	return ret
}

func ParseBytesExtraPath(s string) string {
	parts := strings.Split(filepath.ToSlash(s), "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/")
	}
	return s
}
