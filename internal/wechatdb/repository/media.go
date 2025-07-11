package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/model"
)

func (r *Repository) GetMedia(ctx context.Context, _type string, key string) (*model.Media, error) {
	log.Debug().Str("type", _type).Str("key", key).Msg("Repository层开始获取媒体文件")

	media, err := r.ds.GetMedia(ctx, _type, key)
	if err != nil {
		log.Error().Err(err).Str("type", _type).Str("key", key).Msg("Repository层获取媒体文件失败")
		return nil, err
	}

	log.Debug().Str("type", _type).Str("key", key).Int("data_size", len(media.Data)).Msg("Repository层成功获取媒体文件")
	return media, nil
}
