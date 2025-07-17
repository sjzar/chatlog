package model

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CREATE TABLE Chat_md5(talker)(
// mesLocalID INTEGER PRIMARY KEY AUTOINCREMENT,
// mesSvrID INTEGER,msgCreateTime INTEGER,
// msgContent TEXT,msgStatus INTEGER,
// msgImgStatus INTEGER,
// messageType INTEGER,
// mesDes INTEGER,
// msgSource TEXT,
// IntRes1 INTEGER,
// IntRes2 INTEGER,
// StrRes1 TEXT,
// StrRes2 TEXT,
// msgVoiceText TEXT,
// msgSeq INTEGER,
// CompressContent BLOB,
// ConBlob BLOB
// )
type MessageDarwinV3 struct {
	MsgCreateTime int64  `json:"msgCreateTime"`
	MsgContent    string `json:"msgContent"`
	MessageType   int64  `json:"messageType"`
	MesDes        int    `json:"mesDes"` // 0: 发送, 1: 接收
}

func (m *MessageDarwinV3) Wrap(talker string, dataDir string) *Message {

	_m := &Message{
		Time:       time.Unix(m.MsgCreateTime, 0),
		Type:       m.MessageType,
		Talker:     talker,
		IsChatRoom: strings.HasSuffix(talker, "@chatroom"),
		IsSelf:     m.MesDes == 0,
		Version:    WeChatDarwinV3,
		Contents:   make(map[string]interface{}),
	}

	content := m.MsgContent
	if _m.IsChatRoom {
		split := strings.SplitN(content, ":\n", 2)
		if len(split) == 2 {
			_m.Sender = split[0]
			content = split[1]
		}
	} else if !_m.IsSelf {
		_m.Sender = talker
	}

	_m.ParseMediaInfo(content)

	// 图片消息
	if _m.Type == 3 && dataDir != "" {
		// 计算talker的MD5
		talkerMD5 := md5.Sum([]byte(talker))
		talkerMD5Str := hex.EncodeToString(talkerMD5[:])
		
		// 将消息时间转换成秒
		timeStr := fmt.Sprintf("%d", m.MsgCreateTime)
		
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
		talkerMD5 := md5.Sum([]byte(talker))
		talkerMD5Str := hex.EncodeToString(talkerMD5[:])
		
		// 将消息时间转换成秒
		timeStr := fmt.Sprintf("%d", m.MsgCreateTime)
		
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

	return _m
}
