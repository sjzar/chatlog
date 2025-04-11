package model

import (
	"path/filepath"
)

type Media struct {
	Type       string `json:"type"` // 媒体类型：image, video, voice, file
	Key        string `json:"key"`  // MD5
	Path       string `json:"path"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Data       []byte `json:"data"` // for voice
	ModifyTime int64  `json:"modifyTime"`
}

type MediaV3 struct {
	Type       string `json:"type"`
	Key        string `json:"key"`
	Dir1       string `json:"dir1"`
	Dir2       string `json:"dir2"`
	Name       string `json:"name"`
	ModifyTime int64  `json:"modifyTime"`
}

func (m *MediaV3) Wrap() *Media {

	var path string
	switch m.Type {
	case "image":
		path = filepath.Join("FileStorage", "MsgAttach", m.Dir1, "Image", m.Dir2, m.Name)
	case "video":
		path = filepath.Join("FileStorage", "Video", m.Dir2, m.Name)
	case "file":
		path = filepath.Join("FileStorage", "File", m.Dir2, m.Name)
	}

	return &Media{
		Type:       m.Type,
		Key:        m.Key,
		ModifyTime: m.ModifyTime,
		Path:       path,
		Name:       m.Name,
	}
}
