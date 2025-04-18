package repository

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb/datasource"
)

// Repository 实现了 repository.Repository 接口
type Repository struct {
	ds datasource.DataSource

	// Cache for contact
	contactCache      map[string]*model.Contact
	aliasToContact    map[string][]*model.Contact
	remarkToContact   map[string][]*model.Contact
	nickNameToContact map[string][]*model.Contact
	chatRoomInContact map[string]*model.Contact
	contactList       []string
	aliasList         []string
	remarkList        []string
	nickNameList      []string

	// Cache for chat room
	chatRoomCache      map[string]*model.ChatRoom
	remarkToChatRoom   map[string][]*model.ChatRoom
	nickNameToChatRoom map[string][]*model.ChatRoom
	chatRoomList       []string
	chatRoomRemark     []string
	chatRoomNickName   []string

	// 快速查找索引
	chatRoomUserToInfo map[string]*model.Contact
}

// New 创建一个新的 Repository
func New(ds datasource.DataSource) (*Repository, error) {
	r := &Repository{
		ds:                 ds,
		contactCache:       make(map[string]*model.Contact),
		aliasToContact:     make(map[string][]*model.Contact),
		remarkToContact:    make(map[string][]*model.Contact),
		nickNameToContact:  make(map[string][]*model.Contact),
		chatRoomUserToInfo: make(map[string]*model.Contact),
		contactList:        make([]string, 0),
		aliasList:          make([]string, 0),
		remarkList:         make([]string, 0),
		nickNameList:       make([]string, 0),
		chatRoomCache:      make(map[string]*model.ChatRoom),
		remarkToChatRoom:   make(map[string][]*model.ChatRoom),
		nickNameToChatRoom: make(map[string][]*model.ChatRoom),
		chatRoomList:       make([]string, 0),
		chatRoomRemark:     make([]string, 0),
		chatRoomNickName:   make([]string, 0),
	}

	// 初始化缓存
	if err := r.initCache(context.Background()); err != nil {
		return nil, errors.InitCacheFailed(err)
	}

	ds.SetCallback("contact", r.contactCallback)
	ds.SetCallback("chatroom", r.chatroomCallback)

	return r, nil
}

// initCache 初始化缓存
func (r *Repository) initCache(ctx context.Context) error {
	// 初始化联系人缓存
	if err := r.initContactCache(ctx); err != nil {
		return err
	}

	// 初始化群聊缓存
	if err := r.initChatRoomCache(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Repository) contactCallback(event fsnotify.Event) error {
	if !event.Op.Has(fsnotify.Create) {
		return nil
	}
	if err := r.initContactCache(context.Background()); err != nil {
		log.Err(err).Msgf("Failed to reinitialize contact cache: %s", event.Name)
	}
	return nil
}

func (r *Repository) chatroomCallback(event fsnotify.Event) error {
	if !event.Op.Has(fsnotify.Create) {
		return nil
	}
	if err := r.initChatRoomCache(context.Background()); err != nil {
		log.Err(err).Msgf("Failed to reinitialize contact cache: %s", event.Name)
	}
	return nil
}

// Close 实现 Repository 接口的 Close 方法
func (r *Repository) Close() error {
	return r.ds.Close()
}
