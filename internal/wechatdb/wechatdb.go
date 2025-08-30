package wechatdb

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource"
	"github.com/sjzar/chatlog/internal/wechatdb/repository"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	path     string
	platform string
	version  int
	ds       datasource.DataSource
	repo     *repository.Repository
}

func New(path string, platform string, version int) (*DB, error) {

	w := &DB{
		path:     path,
		platform: platform,
		version:  version,
	}

	// 初始化，加载数据库文件信息
	if err := w.Initialize(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *DB) Close() error {
	if w.repo != nil {
		return w.repo.Close()
	}
	return nil
}

func (w *DB) Initialize() error {
	var err error
	w.ds, err = datasource.New(w.path, w.platform, w.version)
	if err != nil {
		return err
	}

	w.repo, err = repository.New(w.ds)
	if err != nil {
		return err
	}

	return nil
}

func (w *DB) GetMessages(start, end time.Time, talker string, sender string, keyword string, limit, offset int) ([]*model.Message, error) {
	ctx := context.Background()

	// 使用 repository 获取消息
	messages, err := w.repo.GetMessages(ctx, start, end, talker, sender, keyword, limit, offset)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

type GetContactsResp struct {
	Items []*model.Contact `json:"items"`
}

func (w *DB) GetContacts(key string, limit, offset int) (*GetContactsResp, error) {
	ctx := context.Background()

	contacts, err := w.repo.GetContacts(ctx, key, limit, offset)
	if err != nil {
		return nil, err
	}

	return &GetContactsResp{
		Items: contacts,
	}, nil
}

type GetChatRoomsResp struct {
	Items []*model.ChatRoom `json:"items"`
}

func (w *DB) GetChatRooms(key string, limit, offset int) (*GetChatRoomsResp, error) {
	ctx := context.Background()

	chatRooms, err := w.repo.GetChatRooms(ctx, key, limit, offset)
	if err != nil {
		return nil, err
	}

	return &GetChatRoomsResp{
		Items: chatRooms,
	}, nil
}

type GetSessionsResp struct {
	Items []*model.Session `json:"items"`
}

func (w *DB) GetSessions(key string, limit, offset int) (*GetSessionsResp, error) {
	ctx := context.Background()

	// 使用 repository 获取会话列表
	sessions, err := w.repo.GetSessions(ctx, key, limit, offset)
	if err != nil {
		return nil, err
	}

	return &GetSessionsResp{
		Items: sessions,
	}, nil
}

func (w *DB) GetMedia(_type string, key string) (*model.Media, error) {
	log.Info().Str("type", _type).Str("key", key).Str("platform", w.platform).Int("version", w.version).Msg("开始获取媒体文件")

	media, err := w.repo.GetMedia(context.Background(), _type, key)
	if err != nil {
		log.Error().Err(err).Str("type", _type).Str("key", key).Str("platform", w.platform).Int("version", w.version).Msg("获取媒体文件失败")
		return nil, err
	}

	log.Info().Str("type", _type).Str("key", key).Str("platform", w.platform).Int("version", w.version).Int("data_size", len(media.Data)).Msg("成功获取媒体文件")
	return media, nil
}

func (w *DB) SetCallback(group string, callback func(event fsnotify.Event) error) error {
	if w.ds != nil {
		return w.ds.SetCallback(group, callback)
	}
	return nil
}
