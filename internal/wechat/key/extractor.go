package key

import (
	"context"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key/darwin"
	"github.com/sjzar/chatlog/internal/wechat/key/windows"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

// Extractor 定义密钥提取器接口
type Extractor interface {
	// Extract 从进程中提取密钥
	// dataKey, imgKey, error
	Extract(ctx context.Context, proc *model.Process) (string, string, error)

	// SearchKey 在内存中搜索密钥
	SearchKey(ctx context.Context, memory []byte) (string, bool)

	SetValidate(validator *decrypt.Validator)
}

// NewExtractor 创建适合当前平台的密钥提取器
func NewExtractor(platform string, version int) (Extractor, error) {
	switch {
	case platform == "windows" && version == 3:
		return windows.NewV3Extractor(), nil
	case platform == "windows" && version == 4:
		return windows.NewV4Extractor(), nil
	case platform == "darwin" && version == 3:
		return darwin.NewV3Extractor(), nil
	case platform == "darwin" && version == 4:
		return darwin.NewV4Extractor(), nil
	default:
		return nil, errors.PlatformUnsupported(platform, version)
	}
}
