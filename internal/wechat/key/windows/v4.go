package windows

import (
	"context"

	"github.com/sjzar/chatlog/internal/wechat/decrypt"
)

type V4Extractor struct {
	validator *decrypt.Validator
}

func NewV4Extractor() *V4Extractor {
	return &V4Extractor{}
}

func (e *V4Extractor) SearchKey(ctx context.Context, memory []byte) (string, bool) {
	// TODO : Implement the key search logic for V4
	return "", false
}

func (e *V4Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}
