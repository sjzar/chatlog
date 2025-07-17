package model

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"runtime"

	"github.com/sjzar/chatlog/internal/model/wxproto"
	"github.com/sjzar/chatlog/pkg/util/zstd"
	"google.golang.org/protobuf/proto"
)

// CREATE TABLE Msg_md5(talker)(
// local_id INTEGER PRIMARY KEY AUTOINCREMENT,
// server_id INTEGER,
// local_type INTEGER,
// sort_seq INTEGER,
// real_sender_id INTEGER,
// create_time INTEGER,
// status INTEGER,
// upload_status INTEGER,
// download_status INTEGER,
// server_seq INTEGER,
// origin_source INTEGER,
// source TEXT,
// message_content TEXT,
// compress_content TEXT,
// packed_info_data BLOB,
// WCDB_CT_message_content INTEGER DEFAULT NULL,
// WCDB_CT_source INTEGER DEFAULT NULL
// )
type MessageV4 struct {
	SortSeq        int64  `json:"sort_seq"`         // 消息序号，10位时间戳 + 3位序号
	ServerID       int64  `json:"server_id"`        // 消息 ID，用于关联 voice
	LocalType      int64  `json:"local_type"`       // 消息类型
	UserName       string `json:"user_name"`        // 发送人，通过 Join Name2Id 表获得
	CreateTime     int64  `json:"create_time"`      // 消息创建时间，10位时间戳
	MessageContent []byte `json:"message_content"`  // 消息内容，文字聊天内容 或 zstd 压缩内容
	PackedInfoData []byte `json:"packed_info_data"` // 额外数据，类似 proto，格式与 v3 有差异
	Status         int    `json:"status"`           // 消息状态，2 是已发送，4 是已接收，可以用于判断 IsSender（FIXME 不准, 需要判断 UserName）
}

func (m *MessageV4) Wrap(talker string, dataDir string) *Message {

	_m := &Message{
		Seq:        m.SortSeq,
		Time:       time.Unix(m.CreateTime, 0),
		Talker:     talker,
		IsChatRoom: strings.HasSuffix(talker, "@chatroom"),
		Sender:     m.UserName,
		Type:       m.LocalType,
		Contents:   make(map[string]interface{}),
		Version:    WeChatV4,
	}

	// FIXME 后续通过 UserName 判断是否是自己发送的消息，目前可能不准确
	_m.IsSelf = m.Status == 2 || (!_m.IsChatRoom && talker != m.UserName)

	content := ""
	if bytes.HasPrefix(m.MessageContent, []byte{0x28, 0xb5, 0x2f, 0xfd}) {
		if b, err := zstd.Decompress(m.MessageContent); err == nil {
			content = string(b)
		}
	} else {
		content = string(m.MessageContent)
	}

	if _m.IsChatRoom {
		split := strings.SplitN(content, ":\n", 2)
		if len(split) == 2 {
			_m.Sender = split[0]
			content = split[1]
		}
	}

	_m.ParseMediaInfo(content)

	// 图片消息
	if _m.Type == 3 && dataDir != "" {
		// 计算talker的MD5
		talkerMD5 := md5.Sum([]byte(talker))
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
		talkerMD5 := md5.Sum([]byte(talker))
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
		_m.Contents["voice"] = fmt.Sprint(m.ServerID)
	}

	if len(m.PackedInfoData) != 0 {
		if packedInfo := ParsePackedInfo(m.PackedInfoData); packedInfo != nil {
			// FIXME 尝试解决 v4 版本 xml 数据无法匹配到 hardlink 记录的问题
			if _m.Type == 3 && packedInfo.Image != nil {
				_talkerMd5Bytes := md5.Sum([]byte(talker))
				talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
				outfix := "_M"
				if runtime.GOOS == "windows" {
					outfix = "_W"
				}
				_m.Contents["md5"] = packedInfo.Image.Md5
				_m.Contents["imgfile"] = filepath.Join("msg", "attach", talkerMd5, _m.Time.Format("2006-01"), "Img", fmt.Sprintf("%s%s.dat", packedInfo.Image.Md5, outfix))
				_m.Contents["thumb"] = filepath.Join("msg", "attach", talkerMd5, _m.Time.Format("2006-01"), "Img", fmt.Sprintf("%s_t%s.dat", packedInfo.Image.Md5, outfix))
			}
			if _m.Type == 43 && packedInfo.Video != nil {
				_m.Contents["md5"] = packedInfo.Video.Md5
				_m.Contents["videofile"] = filepath.Join("msg", "video", _m.Time.Format("2006-01"), fmt.Sprintf("%s.mp4", packedInfo.Video.Md5))
				_m.Contents["thumb"] = filepath.Join("msg", "video", _m.Time.Format("2006-01"), fmt.Sprintf("%s_thumb.jpg", packedInfo.Video.Md5))
			}
		}
	}

	return _m
}

func ParsePackedInfo(b []byte) *wxproto.PackedInfo {
	var pbMsg wxproto.PackedInfo
	if err := proto.Unmarshal(b, &pbMsg); err != nil {
		return nil
	}
	return &pbMsg
}
