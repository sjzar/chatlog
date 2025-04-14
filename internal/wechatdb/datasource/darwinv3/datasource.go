package darwinv3

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/dbm"
)

const (
	Message  = "message"
	Contact  = "contact"
	ChatRoom = "chatroom"
	Session  = "session"
	Media    = "media"
)

var Groups = []dbm.Group{
	{
		Name:      Message,
		Pattern:   `^msg_([0-9]?[0-9])?\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Contact,
		Pattern:   `^wccontact_new2\.db$`,
		BlackList: []string{},
	},
	{
		Name:      ChatRoom,
		Pattern:   `group_new\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Session,
		Pattern:   `^session_new\.db$`,
		BlackList: []string{},
	},
	{
		Name:      Media,
		Pattern:   `^hldata\.db$`,
		BlackList: []string{},
	},
}

type DataSource struct {
	path string
	dbm  *dbm.DBManager

	talkerDBMap      map[string]string
	user2DisplayName map[string]string
}

func New(path string) (*DataSource, error) {
	ds := &DataSource{
		path:             path,
		dbm:              dbm.NewDBManager(path),
		talkerDBMap:      make(map[string]string),
		user2DisplayName: make(map[string]string),
	}

	for _, g := range Groups {
		ds.dbm.AddGroup(g)
	}

	if err := ds.dbm.Start(); err != nil {
		return nil, err
	}

	if err := ds.initMessageDbs(); err != nil {
		return nil, errors.DBInitFailed(err)
	}
	if err := ds.initChatRoomDb(); err != nil {
		return nil, errors.DBInitFailed(err)
	}

	ds.dbm.AddCallback(Message, func(event fsnotify.Event) error {
		if !event.Op.Has(fsnotify.Create) {
			return nil
		}
		if err := ds.initMessageDbs(); err != nil {
			log.Err(err).Msgf("Failed to reinitialize message DBs: %s", event.Name)
		}
		return nil
	})
	ds.dbm.AddCallback(ChatRoom, func(event fsnotify.Event) error {
		if !event.Op.Has(fsnotify.Create) {
			return nil
		}
		if err := ds.initChatRoomDb(); err != nil {
			log.Err(err).Msgf("Failed to reinitialize chatroom DB: %s", event.Name)
		}
		return nil
	})

	return ds, nil
}

func (ds *DataSource) SetCallback(name string, callback func(event fsnotify.Event) error) error {
	return ds.dbm.AddCallback(name, callback)
}

func (ds *DataSource) initMessageDbs() error {

	dbPaths, err := ds.dbm.GetDBPath(Message)
	if err != nil {
		return err
	}
	// 处理每个数据库文件
	talkerDBMap := make(map[string]string)
	for _, filePath := range dbPaths {
		db, err := ds.dbm.OpenDB(filePath)
		if err != nil {
			log.Err(err).Msgf("获取数据库 %s 失败", filePath)
			continue
		}

		// 获取所有表名
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Chat_%'")
		if err != nil {
			log.Err(err).Msgf("数据库 %s 中没有 Chat 表", filePath)
			continue
		}

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				log.Err(err).Msgf("数据库 %s 扫描表名失败", filePath)
				continue
			}

			// 从表名中提取可能的talker信息
			talkerMd5 := extractTalkerFromTableName(tableName)
			if talkerMd5 == "" {
				continue
			}
			talkerDBMap[talkerMd5] = filePath
		}
		rows.Close()
	}
	ds.talkerDBMap = talkerDBMap
	return nil
}

func (ds *DataSource) initChatRoomDb() error {
	db, err := ds.dbm.GetDB(ChatRoom)
	if err != nil {
		return err
	}

	rows, err := db.Query("SELECT m_nsUsrName, IFNULL(nickname,\"\") FROM GroupMember")
	if err != nil {
		log.Err(err).Msg("获取群聊成员失败")
		return nil
	}

	user2DisplayName := make(map[string]string)
	for rows.Next() {
		var user string
		var nickName string
		if err := rows.Scan(&user, &nickName); err != nil {
			log.Err(err).Msg("扫描表名失败")
			continue
		}
		user2DisplayName[user] = nickName
	}
	rows.Close()
	ds.user2DisplayName = user2DisplayName

	return nil
}

// GetMessages 实现获取消息的方法
func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, limit, offset int) ([]*model.Message, error) {
	// 在 darwinv3 中，每个联系人/群聊的消息存储在单独的表中，表名为 Chat_md5(talker)
	// 首先需要找到对应的表名
	if talker == "" {
		return nil, errors.ErrTalkerEmpty
	}

	_talkerMd5Bytes := md5.Sum([]byte(talker))
	talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
	dbPath, ok := ds.talkerDBMap[talkerMd5]
	if !ok {
		return nil, errors.TalkerNotFound(talker)
	}
	db, err := ds.dbm.OpenDB(dbPath)
	if err != nil {
		return nil, err
	}
	tableName := fmt.Sprintf("Chat_%s", talkerMd5)

	// 构建查询条件
	query := fmt.Sprintf(`
		SELECT msgCreateTime, msgContent, messageType, mesDes
		FROM %s 
		WHERE msgCreateTime >= ? AND msgCreateTime <= ? 
		ORDER BY msgCreateTime ASC
	`, tableName)

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)

		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	rows, err := db.QueryContext(ctx, query, startTime.Unix(), endTime.Unix())
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	// 处理查询结果
	messages := []*model.Message{}
	for rows.Next() {
		var msg model.MessageDarwinV3
		err := rows.Scan(
			&msg.MsgCreateTime,
			&msg.MsgContent,
			&msg.MessageType,
			&msg.MesDes,
		)
		if err != nil {
			log.Err(err).Msgf("扫描消息行失败")
			continue
		}

		// 将消息包装为通用模型
		message := msg.Wrap(talker)
		messages = append(messages, message)
	}

	return messages, nil
}

// 从表名中提取 talker
func extractTalkerFromTableName(tableName string) string {

	if !strings.HasPrefix(tableName, "Chat_") {
		return ""
	}

	if strings.HasSuffix(tableName, "_dels") {
		return ""
	}

	return strings.TrimPrefix(tableName, "Chat_")
}

// GetContacts 实现获取联系人信息的方法
func (ds *DataSource) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), m_uiSex, IFNULL(m_nsAliasName,"") 
				FROM WCContact 
				WHERE m_nsUsrName = ? OR nickname = ? OR m_nsRemark = ? OR m_nsAliasName = ?`
		args = []interface{}{key, key, key, key}
	} else {
		// 查询所有联系人
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), m_uiSex, IFNULL(m_nsAliasName,"") 
				FROM WCContact`
	}

	// 添加排序、分页
	query += ` ORDER BY m_nsUsrName`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	db, err := ds.dbm.GetDB(Contact)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	contacts := []*model.Contact{}
	for rows.Next() {
		var contactDarwinV3 model.ContactDarwinV3
		err := rows.Scan(
			&contactDarwinV3.M_nsUsrName,
			&contactDarwinV3.Nickname,
			&contactDarwinV3.M_nsRemark,
			&contactDarwinV3.M_uiSex,
			&contactDarwinV3.M_nsAliasName,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		contacts = append(contacts, contactDarwinV3.Wrap())
	}

	return contacts, nil
}

// GetChatRooms 实现获取群聊信息的方法
func (ds *DataSource) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact 
				WHERE m_nsUsrName = ? OR nickname = ? OR m_nsRemark = ?`
		args = []interface{}{key, key, key}
	} else {
		// 查询所有群聊
		query = `SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact`
	}

	// 添加排序、分页
	query += ` ORDER BY m_nsUsrName`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	db, err := ds.dbm.GetDB(ChatRoom)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	chatRooms := []*model.ChatRoom{}
	for rows.Next() {
		var chatRoomDarwinV3 model.ChatRoomDarwinV3
		err := rows.Scan(
			&chatRoomDarwinV3.M_nsUsrName,
			&chatRoomDarwinV3.Nickname,
			&chatRoomDarwinV3.M_nsRemark,
			&chatRoomDarwinV3.M_nsChatRoomMemList,
			&chatRoomDarwinV3.M_nsChatRoomAdminList,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		chatRooms = append(chatRooms, chatRoomDarwinV3.Wrap(ds.user2DisplayName))
	}

	// 如果没有找到群聊，尝试通过联系人查找
	if len(chatRooms) == 0 && key != "" {
		contacts, err := ds.GetContacts(ctx, key, 1, 0)
		if err == nil && len(contacts) > 0 && strings.HasSuffix(contacts[0].UserName, "@chatroom") {
			// 再次尝试通过用户名查找群聊
			rows, err := db.QueryContext(ctx,
				`SELECT IFNULL(m_nsUsrName,""), IFNULL(nickname,""), IFNULL(m_nsRemark,""), IFNULL(m_nsChatRoomMemList,""), IFNULL(m_nsChatRoomAdminList,"") 
				FROM GroupContact 
				WHERE m_nsUsrName = ?`,
				contacts[0].UserName)

			if err != nil {
				return nil, errors.QueryFailed(query, err)
			}
			defer rows.Close()

			for rows.Next() {
				var chatRoomDarwinV3 model.ChatRoomDarwinV3
				err := rows.Scan(
					&chatRoomDarwinV3.M_nsUsrName,
					&chatRoomDarwinV3.Nickname,
					&chatRoomDarwinV3.M_nsRemark,
					&chatRoomDarwinV3.M_nsChatRoomMemList,
					&chatRoomDarwinV3.M_nsChatRoomAdminList,
				)

				if err != nil {
					return nil, errors.ScanRowFailed(err)
				}

				chatRooms = append(chatRooms, chatRoomDarwinV3.Wrap(ds.user2DisplayName))
			}

			// 如果群聊记录不存在，但联系人记录存在，创建一个模拟的群聊对象
			if len(chatRooms) == 0 {
				chatRooms = append(chatRooms, &model.ChatRoom{
					Name:  contacts[0].UserName,
					Users: make([]model.ChatRoomUser, 0),
				})
			}
		}
	}

	return chatRooms, nil
}

// GetSessions 实现获取会话信息的方法
func (ds *DataSource) GetSessions(ctx context.Context, key string, limit, offset int) ([]*model.Session, error) {
	var query string
	var args []interface{}

	if key != "" {
		// 按照关键字查询
		query = `SELECT m_nsUserName, m_uLastTime 
				FROM SessionAbstract 
				WHERE m_nsUserName = ?`
		args = []interface{}{key}
	} else {
		// 查询所有会话
		query = `SELECT m_nsUserName, m_uLastTime 
				FROM SessionAbstract`
	}

	// 添加排序、分页
	query += ` ORDER BY m_uLastTime DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// 执行查询
	db, err := ds.dbm.GetDB(Session)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	sessions := []*model.Session{}
	for rows.Next() {
		var sessionDarwinV3 model.SessionDarwinV3
		err := rows.Scan(
			&sessionDarwinV3.M_nsUserName,
			&sessionDarwinV3.M_uLastTime,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		// 包装成通用模型
		session := sessionDarwinV3.Wrap()

		// 尝试获取联系人信息以补充会话信息
		contacts, err := ds.GetContacts(ctx, session.UserName, 1, 0)
		if err == nil && len(contacts) > 0 {
			session.NickName = contacts[0].DisplayName()
		} else {
			// 尝试获取群聊信息
			chatRooms, err := ds.GetChatRooms(ctx, session.UserName, 1, 0)
			if err == nil && len(chatRooms) > 0 {
				session.NickName = chatRooms[0].DisplayName()
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (ds *DataSource) GetMedia(ctx context.Context, _type string, key string) (*model.Media, error) {
	if key == "" {
		return nil, errors.ErrKeyEmpty
	}
	query := `SELECT 
    r.mediaMd5,
    r.mediaSize,
    r.inodeNumber,
    r.modifyTime,
    d.relativePath,
    d.fileName
FROM 
    HlinkMediaRecord r
JOIN 
    HlinkMediaDetail d ON r.inodeNumber = d.inodeNumber
WHERE 
    r.mediaMd5 = ?`
	args := []interface{}{key}
	// 执行查询
	db, err := ds.dbm.GetDB(Media)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.QueryFailed(query, err)
	}
	defer rows.Close()

	var media *model.Media
	for rows.Next() {
		var mediaDarwinV3 model.MediaDarwinV3
		err := rows.Scan(
			&mediaDarwinV3.MediaMd5,
			&mediaDarwinV3.MediaSize,
			&mediaDarwinV3.InodeNumber,
			&mediaDarwinV3.ModifyTime,
			&mediaDarwinV3.RelativePath,
			&mediaDarwinV3.FileName,
		)

		if err != nil {
			return nil, errors.ScanRowFailed(err)
		}

		// 包装成通用模型
		media = mediaDarwinV3.Wrap()
	}

	if media == nil {
		return nil, errors.ErrMediaNotFound
	}

	return media, nil
}

// Close 实现关闭数据库连接的方法
func (ds *DataSource) Close() error {
	return ds.dbm.Close()
}
