package darwinv3

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource/dbm"
	"github.com/sjzar/chatlog/pkg/util"
)

const (
	Message  = "message"
	Contact  = "contact"
	ChatRoom = "chatroom"
	Session  = "session"
	Media    = "media"
)

var Groups = []*dbm.Group{
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

func (ds *DataSource) SetCallback(group string, callback func(event fsnotify.Event) error) error {
	return ds.dbm.AddCallback(group, callback)
}

func (ds *DataSource) initMessageDbs() error {

	dbPaths, err := ds.dbm.GetDBPath(Message)
	if err != nil {
		if strings.Contains(err.Error(), "db file not found") {
			ds.talkerDBMap = make(map[string]string)
			return nil
		}
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
		if strings.Contains(err.Error(), "db file not found") {
			ds.user2DisplayName = make(map[string]string)
			return nil
		}
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

func (ds *DataSource) GetMessages(ctx context.Context, startTime, endTime time.Time, talker string, sender string, keyword string, limit, offset int) ([]*model.Message, error) {
	if talker == "" {
		return nil, errors.ErrTalkerEmpty
	}

	// 解析talker参数，支持多个talker（以英文逗号分隔）
	talkers := util.Str2List(talker, ",")
	if len(talkers) == 0 {
		return nil, errors.ErrTalkerEmpty
	}

	// 解析sender参数，支持多个发送者（以英文逗号分隔）
	senders := util.Str2List(sender, ",")

	// 预编译正则表达式（如果有keyword）
	var regex *regexp.Regexp
	if keyword != "" {
		var err error
		regex, err = regexp.Compile(keyword)
		if err != nil {
			return nil, errors.QueryFailed("invalid regex pattern", err)
		}
	}

	// 从每个相关数据库中查询消息，并在读取时进行过滤
	filteredMessages := []*model.Message{}

	// 对每个talker进行查询
	for _, talkerItem := range talkers {
		// 检查上下文是否已取消
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// 在 darwinv3 中，需要先找到对应的数据库
		_talkerMd5Bytes := md5.Sum([]byte(talkerItem))
		talkerMd5 := hex.EncodeToString(_talkerMd5Bytes[:])
		dbPath, ok := ds.talkerDBMap[talkerMd5]
		if !ok {
			// 如果找不到对应的数据库，跳过此talker
			continue
		}

		db, err := ds.dbm.OpenDB(dbPath)
		if err != nil {
			log.Error().Msgf("数据库 %s 未打开", dbPath)
			continue
		}

		tableName := fmt.Sprintf("Chat_%s", talkerMd5)

		// 构建查询条件
		query := fmt.Sprintf(`
			SELECT msgCreateTime, msgContent, messageType, mesDes
			FROM %s 
			WHERE msgCreateTime >= ? AND msgCreateTime <= ? 
			ORDER BY msgCreateTime ASC
		`, tableName)

		// 执行查询
		rows, err := db.QueryContext(ctx, query, startTime.Unix(), endTime.Unix())
		if err != nil {
			// 如果表不存在，跳过此talker
			if strings.Contains(err.Error(), "no such table") {
				continue
			}
			log.Err(err).Msgf("从数据库 %s 查询消息失败", dbPath)
			continue
		}

		// 处理查询结果，在读取时进行过滤
		for rows.Next() {
			var msg model.MessageDarwinV3
			err := rows.Scan(
				&msg.MsgCreateTime,
				&msg.MsgContent,
				&msg.MessageType,
				&msg.MesDes,
			)
			if err != nil {
				rows.Close()
				log.Err(err).Msgf("扫描消息行失败")
				continue
			}

			// 将消息包装为通用模型
			message := msg.Wrap(talkerItem)

			// 应用sender过滤
			if len(senders) > 0 {
				senderMatch := false
				for _, s := range senders {
					if message.Sender == s {
						senderMatch = true
						break
					}
				}
				if !senderMatch {
					continue // 不匹配sender，跳过此消息
				}
			}

			// 应用keyword过滤
			if regex != nil {
				plainText := message.PlainTextContent()
				if !regex.MatchString(plainText) {
					continue // 不匹配keyword，跳过此消息
				}
			}

			// 通过所有过滤条件，保留此消息
			filteredMessages = append(filteredMessages, message)

			// 检查是否已经满足分页处理数量
			if limit > 0 && len(filteredMessages) >= offset+limit {
				// 已经获取了足够的消息，可以提前返回
				rows.Close()

				// 对所有消息按时间排序
				sort.Slice(filteredMessages, func(i, j int) bool {
					return filteredMessages[i].Seq < filteredMessages[j].Seq
				})

				// 处理分页
				if offset >= len(filteredMessages) {
					return []*model.Message{}, nil
				}
				end := offset + limit
				if end > len(filteredMessages) {
					end = len(filteredMessages)
				}
				return filteredMessages[offset:end], nil
			}
		}
		rows.Close()
	}

	// 对所有消息按时间排序
	// FIXME 不同 talker 需要使用 Time 排序
	sort.Slice(filteredMessages, func(i, j int) bool {
		return filteredMessages[i].Time.Before(filteredMessages[j].Time)
	})

	// 处理分页
	if limit > 0 {
		if offset >= len(filteredMessages) {
			return []*model.Message{}, nil
		}
		end := offset + limit
		if end > len(filteredMessages) {
			end = len(filteredMessages)
		}
		return filteredMessages[offset:end], nil
	}

	return filteredMessages, nil
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
