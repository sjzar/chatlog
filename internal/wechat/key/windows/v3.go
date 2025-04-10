package windows

import (
	"context"

	"github.com/sjzar/chatlog/internal/wechat/decrypt"
)

type V3Extractor struct {
	validator *decrypt.Validator
}

func NewV3Extractor() *V3Extractor {
	return &V3Extractor{}
}

func (e *V3Extractor) SearchKey(ctx context.Context, memory []byte) (string, bool) {
	// TODO : Implement the key search logic for V3
	return "", false
}

func (e *V3Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}
