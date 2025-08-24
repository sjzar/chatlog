package repository

import (
	"context"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
)

// initChatRoomCache 初始化群聊缓存
func (r *Repository) initChatRoomCache(ctx context.Context) error {

	chatRoomMap := make(map[string]*model.ChatRoom)
	remarkToChatRoom := make(map[string][]*model.ChatRoom)
	nickNameToChatRoom := make(map[string][]*model.ChatRoom)
	chatRoomList := make([]string, 0)
	chatRoomRemark := make([]string, 0)
	chatRoomNickName := make([]string, 0)

	// 加载所有群聊到缓存
	// 暂时忽略获取不到群聊的错误
	chatRooms, err := r.ds.GetChatRooms(ctx, "", 0, 0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load chat rooms")
	}

	for _, chatRoom := range chatRooms {
		// 补充群聊信息（从联系人中获取 Remark 和 NickName）
		r.enrichChatRoom(chatRoom)
		chatRoomMap[chatRoom.Name] = chatRoom
		chatRoomList = append(chatRoomList, chatRoom.Name)
		if chatRoom.Remark != "" {
			remark, ok := remarkToChatRoom[chatRoom.Remark]
			if !ok {
				remark = make([]*model.ChatRoom, 0)
			}
			remark = append(remark, chatRoom)
			remarkToChatRoom[chatRoom.Remark] = remark
			chatRoomRemark = append(chatRoomRemark, chatRoom.Remark)
		}
		if chatRoom.NickName != "" {
			nickName, ok := nickNameToChatRoom[chatRoom.NickName]
			if !ok {
				nickName = make([]*model.ChatRoom, 0)
			}
			nickName = append(nickName, chatRoom)
			nickNameToChatRoom[chatRoom.NickName] = nickName
			chatRoomNickName = append(chatRoomNickName, chatRoom.NickName)
		}
	}

	for _, contact := range r.chatRoomInContact {
		if _, ok := chatRoomMap[contact.UserName]; !ok {
			chatRoom := &model.ChatRoom{
				Name:     contact.UserName,
				Remark:   contact.Remark,
				NickName: contact.NickName,
			}
			chatRoomMap[contact.UserName] = chatRoom
			chatRoomList = append(chatRoomList, contact.UserName)
			if contact.Remark != "" {
				remark, ok := remarkToChatRoom[chatRoom.Remark]
				if !ok {
					remark = make([]*model.ChatRoom, 0)
				}
				remark = append(remark, chatRoom)
				remarkToChatRoom[chatRoom.Remark] = remark
				chatRoomRemark = append(chatRoomRemark, contact.Remark)
			}
			if contact.NickName != "" {
				nickName, ok := nickNameToChatRoom[chatRoom.NickName]
				if !ok {
					nickName = make([]*model.ChatRoom, 0)
				}
				nickName = append(nickName, chatRoom)
				nickNameToChatRoom[chatRoom.NickName] = nickName
				chatRoomNickName = append(chatRoomNickName, contact.NickName)
			}
		}
	}
	sort.Strings(chatRoomList)
	sort.Strings(chatRoomRemark)
	sort.Strings(chatRoomNickName)

	r.chatRoomCache = chatRoomMap
	r.remarkToChatRoom = remarkToChatRoom
	r.nickNameToChatRoom = nickNameToChatRoom
	r.chatRoomList = chatRoomList
	r.chatRoomRemark = chatRoomRemark
	r.chatRoomNickName = chatRoomNickName

	return nil
}

func (r *Repository) GetChatRooms(ctx context.Context, key string, limit, offset int) ([]*model.ChatRoom, error) {

	ret := make([]*model.ChatRoom, 0)
	if key != "" {
		ret = r.findChatRooms(key)
		if len(ret) == 0 {
			return []*model.ChatRoom{}, nil
		}

		if limit > 0 {
			end := offset + limit
			if end > len(ret) {
				end = len(ret)
			}
			if offset >= len(ret) {
				return []*model.ChatRoom{}, nil
			}
			return ret[offset:end], nil
		}
	} else {
		list := r.chatRoomList
		if limit > 0 {
			end := offset + limit
			if end > len(list) {
				end = len(list)
			}
			if offset >= len(list) {
				return []*model.ChatRoom{}, nil
			}
			list = list[offset:end]
		}
		for _, name := range list {
			ret = append(ret, r.chatRoomCache[name])
		}
	}

	return ret, nil
}

func (r *Repository) GetChatRoom(ctx context.Context, key string) (*model.ChatRoom, error) {
	chatRoom := r.findChatRoom(key)
	if chatRoom == nil {
		return nil, errors.ChatRoomNotFound(key)
	}
	return chatRoom, nil
}

// enrichChatRoom 从联系人信息中补充群聊信息
func (r *Repository) enrichChatRoom(chatRoom *model.ChatRoom) {
	if contact, ok := r.contactCache[chatRoom.Name]; ok {
		chatRoom.Remark = contact.Remark
		chatRoom.NickName = contact.NickName
	}
}

func (r *Repository) findChatRoom(key string) *model.ChatRoom {
	if chatRoom, ok := r.chatRoomCache[key]; ok {
		return chatRoom
	}
	if chatRoom, ok := r.remarkToChatRoom[key]; ok {
		return chatRoom[0]
	}
	if chatRoom, ok := r.nickNameToChatRoom[key]; ok {
		return chatRoom[0]
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) {
			return r.remarkToChatRoom[remark][0]
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) {
			return r.nickNameToChatRoom[nickName][0]
		}
	}

	return nil
}

func (r *Repository) findChatRooms(key string) []*model.ChatRoom {
	ret := make([]*model.ChatRoom, 0)
	distinct := make(map[string]bool)
	if chatRoom, ok := r.chatRoomCache[key]; ok {
		ret = append(ret, chatRoom)
		distinct[chatRoom.Name] = true
	}
	if chatRooms, ok := r.remarkToChatRoom[key]; ok {
		for _, chatRoom := range chatRooms {
			if !distinct[chatRoom.Name] {
				ret = append(ret, chatRoom)
				distinct[chatRoom.Name] = true
			}
		}
	}
	if chatRooms, ok := r.nickNameToChatRoom[key]; ok {
		for _, chatRoom := range chatRooms {
			if !distinct[chatRoom.Name] {
				ret = append(ret, chatRoom)
				distinct[chatRoom.Name] = true
			}
		}
	}

	// Contain
	for _, remark := range r.chatRoomRemark {
		if strings.Contains(remark, key) {
			for _, chatRoom := range r.remarkToChatRoom[remark] {
				if !distinct[chatRoom.Name] {
					ret = append(ret, chatRoom)
					distinct[chatRoom.Name] = true
				}
			}
		}
	}
	for _, nickName := range r.chatRoomNickName {
		if strings.Contains(nickName, key) {
			for _, chatRoom := range r.nickNameToChatRoom[nickName] {
				if !distinct[chatRoom.Name] {
					ret = append(ret, chatRoom)
					distinct[chatRoom.Name] = true
				}
			}
		}
	}

	return ret
}
