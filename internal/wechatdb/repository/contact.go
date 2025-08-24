package repository

import (
	"context"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/model"
)

// initContactCache 初始化联系人缓存
func (r *Repository) initContactCache(ctx context.Context) error {

	contactMap := make(map[string]*model.Contact)
	aliasMap := make(map[string][]*model.Contact)
	remarkMap := make(map[string][]*model.Contact)
	nickNameMap := make(map[string][]*model.Contact)
	chatRoomUserMap := make(map[string]*model.Contact)
	chatRoomInContactMap := make(map[string]*model.Contact)
	contactList := make([]string, 0)
	aliasList := make([]string, 0)
	remarkList := make([]string, 0)
	nickNameList := make([]string, 0)

	// 加载所有联系人到缓存
	// 暂时忽略获取不到联系人的错误
	contacts, err := r.ds.GetContacts(ctx, "", 0, 0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load contacts")
	}

	for _, contact := range contacts {
		contactMap[contact.UserName] = contact
		contactList = append(contactList, contact.UserName)

		// 建立快速查找索引
		if contact.Alias != "" {
			alias, ok := aliasMap[contact.Alias]
			if !ok {
				alias = make([]*model.Contact, 0)
			}
			alias = append(alias, contact)
			aliasMap[contact.Alias] = alias
			aliasList = append(aliasList, contact.Alias)
		}
		if contact.Remark != "" {
			remark, ok := remarkMap[contact.Remark]
			if !ok {
				remark = make([]*model.Contact, 0)
			}
			remark = append(remark, contact)
			remarkMap[contact.Remark] = remark
			remarkList = append(remarkList, contact.Remark)
		}
		if contact.NickName != "" {
			nickName, ok := nickNameMap[contact.NickName]
			if !ok {
				nickName = make([]*model.Contact, 0)
			}
			nickName = append(nickName, contact)
			nickNameMap[contact.NickName] = nickName
			nickNameList = append(nickNameList, contact.NickName)
		}

		// 如果是群聊成员（非好友），添加到群聊成员索引
		if !contact.IsFriend {
			chatRoomUserMap[contact.UserName] = contact
		}

		if strings.HasSuffix(contact.UserName, "@chatroom") {
			chatRoomInContactMap[contact.UserName] = contact
		}
	}

	sort.Strings(contactList)
	sort.Strings(aliasList)
	sort.Strings(remarkList)
	sort.Strings(nickNameList)

	r.contactCache = contactMap
	r.aliasToContact = aliasMap
	r.remarkToContact = remarkMap
	r.nickNameToContact = nickNameMap
	r.chatRoomUserToInfo = chatRoomUserMap
	r.chatRoomInContact = chatRoomInContactMap
	r.contactList = contactList
	r.aliasList = aliasList
	r.remarkList = remarkList
	r.nickNameList = nickNameList
	return nil
}

func (r *Repository) GetContact(ctx context.Context, key string) (*model.Contact, error) {
	// 先尝试从缓存中获取
	contact := r.findContact(key)
	if contact == nil {
		return nil, errors.ContactNotFound(key)
	}
	return contact, nil
}

func (r *Repository) GetContacts(ctx context.Context, key string, limit, offset int) ([]*model.Contact, error) {
	ret := make([]*model.Contact, 0)
	if key != "" {
		ret = r.findContacts(key)
		if len(ret) == 0 {
			return []*model.Contact{}, nil
		}
		if limit > 0 {
			end := offset + limit
			if end > len(ret) {
				end = len(ret)
			}
			if offset >= len(ret) {
				return []*model.Contact{}, nil
			}
			return ret[offset:end], nil
		}
	} else {
		list := r.contactList
		if limit > 0 {
			end := offset + limit
			if end > len(list) {
				end = len(list)
			}
			if offset >= len(list) {
				return []*model.Contact{}, nil
			}
			list = list[offset:end]
		}
		for _, name := range list {
			ret = append(ret, r.contactCache[name])
		}
	}
	return ret, nil
}

func (r *Repository) findContact(key string) *model.Contact {
	if contact, ok := r.contactCache[key]; ok {
		return contact
	}
	if contact, ok := r.aliasToContact[key]; ok {
		return contact[0]
	}
	if contact, ok := r.remarkToContact[key]; ok {
		return contact[0]
	}
	if contact, ok := r.nickNameToContact[key]; ok {
		return contact[0]
	}

	// Contain
	for _, alias := range r.aliasList {
		if strings.Contains(alias, key) {
			return r.aliasToContact[alias][0]
		}
	}
	for _, remark := range r.remarkList {
		if strings.Contains(remark, key) {
			return r.remarkToContact[remark][0]
		}
	}
	for _, nickName := range r.nickNameList {
		if strings.Contains(nickName, key) {
			return r.nickNameToContact[nickName][0]
		}
	}
	return nil
}

func (r *Repository) findContacts(key string) []*model.Contact {
	ret := make([]*model.Contact, 0)
	distinct := make(map[string]bool)
	if contact, ok := r.contactCache[key]; ok {
		ret = append(ret, contact)
		distinct[contact.UserName] = true
	}
	if contacts, ok := r.aliasToContact[key]; ok {
		for _, contact := range contacts {
			if !distinct[contact.UserName] {
				ret = append(ret, contact)
				distinct[contact.UserName] = true
			}
		}
	}
	if contacts, ok := r.remarkToContact[key]; ok {
		for _, contact := range contacts {
			if !distinct[contact.UserName] {
				ret = append(ret, contact)
				distinct[contact.UserName] = true
			}
		}
	}
	if contacts, ok := r.nickNameToContact[key]; ok {
		for _, contact := range contacts {
			if !distinct[contact.UserName] {
				ret = append(ret, contact)
				distinct[contact.UserName] = true
			}
		}
	}
	// Contain
	for _, alias := range r.aliasList {
		if strings.Contains(alias, key) {
			for _, contact := range r.aliasToContact[alias] {
				if !distinct[contact.UserName] {
					ret = append(ret, contact)
					distinct[contact.UserName] = true
				}
			}
		}
	}
	for _, remark := range r.remarkList {
		if strings.Contains(remark, key) {
			for _, contact := range r.remarkToContact[remark] {
				if !distinct[contact.UserName] {
					ret = append(ret, contact)
					distinct[contact.UserName] = true
				}
			}
		}
	}
	for _, nickName := range r.nickNameList {
		if strings.Contains(nickName, key) {
			for _, contact := range r.nickNameToContact[nickName] {
				if !distinct[contact.UserName] {
					ret = append(ret, contact)
					distinct[contact.UserName] = true
				}
			}
		}
	}

	return ret
}

// getFullContact 获取联系人信息，包括群聊成员
func (r *Repository) getFullContact(userName string) *model.Contact {
	// 先查找联系人缓存
	if contact, ok := r.contactCache[userName]; ok {
		return contact
	}

	// 再查找群聊成员缓存
	contact, ok := r.chatRoomUserToInfo[userName]

	if ok {
		return contact
	}

	return nil
}
