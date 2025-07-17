package model

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
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

func (m *MessageV3) Wrap(dataDir string) *Message {

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
		Contents:   make(map[string]interface{}),
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

	// 图片消息
	if _m.Type == 3 && dataDir != "" {
		// 计算talker的MD5
		talkerMD5 := md5.Sum([]byte(m.StrTalker))
		talkerMD5Str := hex.EncodeToString(talkerMD5[:])
		
		// 将消息时间转换成秒
		timeStr := fmt.Sprintf("%d", m.CreateTime)
		
		// 构建搜索目录
		searchDir := filepath.Join(dataDir, "Message", "MessageTemp", talkerMD5Str, "Image")
		
		// 按优先级查找图片文件
		patterns := []string{
			fmt.Sprintf("%s_.pic.jpg", timeStr),
			fmt.Sprintf("%s_.pic_hd.jpg", timeStr),
			fmt.Sprintf("%s_.pic_thumb.jpg", timeStr),
		}
		
		for _, pattern := range patterns {
			if entries, err := os.ReadDir(searchDir); err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), pattern) {
						// 找到文件，设置相对路径
						_m.Contents["imgfile"] = filepath.Join("/Message/MessageTemp", talkerMD5Str, "Image", entry.Name())
						break
					}
				}
				if _, exists := _m.Contents["imgfile"]; exists {
					break // 找到文件就退出
				}
			}
		}
	}

	// 视频消息
	if _m.Type == 43 && dataDir != "" {
		// 计算talker的MD5
		talkerMD5 := md5.Sum([]byte(m.StrTalker))
		talkerMD5Str := hex.EncodeToString(talkerMD5[:])
		
		// 将消息时间转换成秒
		timeStr := fmt.Sprintf("%d", m.CreateTime)
		
		// 构建搜索目录
		searchDir := filepath.Join(dataDir, "Message", "MessageTemp", talkerMD5Str, "Video")
		
		// 查找视频文件
		videoPattern := fmt.Sprintf("%s.mp4", timeStr)
		if entries, err := os.ReadDir(searchDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), videoPattern) {
					// 找到文件，设置相对路径
					_m.Contents["videofile"] = filepath.Join("/Message/MessageTemp", talkerMD5Str, "Video", entry.Name())
					break
				}
			}
		}
	}

	// 语音消息
	if _m.Type == 34 {
		_m.Contents["voice"] = fmt.Sprint(m.MsgSvrID)
	}

	if len(m.BytesExtra) != 0 {
		if bytesExtra := ParseBytesExtra(m.BytesExtra); bytesExtra != nil {
			if _m.IsChatRoom {
				_m.Sender = bytesExtra[1]
			}
			// FIXME xml 中的 md5 数据无法匹配到 hardlink 记录，所以直接用 proto 数据
			if _m.Type == 43 {
				path := bytesExtra[4]
				parts := strings.Split(filepath.ToSlash(path), "/")
				if len(parts) > 1 {
					path = strings.Join(parts[1:], "/")
				}
				_m.Contents["videofile"] = path
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
