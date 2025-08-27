package model

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

func (m *MessageV4) Wrap(talker string) *Message {

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
				_m.Contents["path"] = filepath.Join("msg", "attach", talkerMd5, _m.Time.Format("2006-01"), "Img", packedInfo.Image.Md5)
			}
			if _m.Type == 43 && packedInfo.Video != nil {
				_m.Contents["path"] = filepath.Join("msg", "video", _m.Time.Format("2006-01"), packedInfo.Video.Md5)
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
