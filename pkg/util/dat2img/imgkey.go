package dat2img

import (
	"bytes"
	"crypto/aes"
	"os"
	"path/filepath"
	"strings"
)

type AesKeyValidator struct {
	Path          string
	EncryptedData []byte
}

func NewImgKeyValidator(path string) *AesKeyValidator {
	validator := &AesKeyValidator{
		Path: path,
	}

	// Walk the directory to find *.dat files (excluding *_t.dat files)
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process *.dat files but exclude *_t.dat files
		if !strings.HasSuffix(info.Name(), ".dat") || strings.HasSuffix(info.Name(), "_t.dat") {
			return nil
		}

		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}

		// Check if header matches V4Format2.Header
		// Get aes.BlockSize (16) bytes starting from position 15
		if len(data) >= 15+aes.BlockSize && bytes.Equal(data[:4], V4Format2.Header) {
			validator.EncryptedData = make([]byte, aes.BlockSize)
			copy(validator.EncryptedData, data[15:15+aes.BlockSize])
			return filepath.SkipAll // Found what we need, stop walking
		}

		return nil
	})

	// image data not found
	if len(validator.EncryptedData) == 0 {
		return nil
	}

	return validator
}

func (v *AesKeyValidator) Validate(key []byte) bool {
	if len(v.EncryptedData) == 0 {
		return false
	}
	if len(key) < 16 {
		return false
	}
	aesKey := key[:16]

	cipher, err := aes.NewCipher(aesKey)
	if err != nil {
		return false
	}

	decrypted := make([]byte, len(v.EncryptedData))
	cipher.Decrypt(decrypted, v.EncryptedData)

	return bytes.HasPrefix(decrypted, JPG.Header) || bytes.HasPrefix(decrypted, WXGF.Header)
}
