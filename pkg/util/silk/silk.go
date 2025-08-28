package silk

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func Silk2MP3(data []byte) ([]byte, error) {
	log.Warn().Msg("SILK到MP3转换功能已禁用 - 缺少必要的音频库")
	return nil, fmt.Errorf("silk to mp3 conversion disabled - missing audio libraries")
}
