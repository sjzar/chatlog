package dat2img

// Implementation based on:
// - https://github.com/tujiaw/wechat_dat_to_image
// - https://github.com/LC044/WeChatMsg/blob/6535ed0/wxManager/decrypt/decrypt_dat.py

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Format defines the header and extension for different image types
type Format struct {
	Header []byte
	AesKey []byte
	Ext    string
}

var (
	// Common image format definitions
	JPG     = Format{Header: []byte{0xFF, 0xD8, 0xFF}, Ext: "jpg"}
	PNG     = Format{Header: []byte{0x89, 0x50, 0x4E, 0x47}, Ext: "png"}
	GIF     = Format{Header: []byte{0x47, 0x49, 0x46, 0x38}, Ext: "gif"}
	TIFF    = Format{Header: []byte{0x49, 0x49, 0x2A, 0x00}, Ext: "tiff"}
	BMP     = Format{Header: []byte{0x42, 0x4D}, Ext: "bmp"}
	WXGF    = Format{Header: []byte{0x77, 0x78, 0x67, 0x66}, Ext: "wxgf"}
	Formats = []Format{JPG, PNG, GIF, TIFF, BMP, WXGF}

	V4Format1 = Format{Header: []byte{0x07, 0x08, 0x56, 0x31}, AesKey: []byte("cfcd208495d565ef")}
	V4Format2 = Format{Header: []byte{0x07, 0x08, 0x56, 0x32}, AesKey: []byte("0000000000000000")} // FIXME
	V4Formats = []*Format{&V4Format1, &V4Format2}

	// WeChat v4 related constants
	V4XorKey byte = 0x37               // Default XOR key for WeChat v4 dat files
	JpgTail       = []byte{0xFF, 0xD9} // JPG file tail marker
)

// Dat2Image converts WeChat dat file data to image data
// Returns the decoded image data, file extension, and any error encountered
func Dat2Image(data []byte) ([]byte, string, error) {
	if len(data) < 4 {
		return nil, "", fmt.Errorf("data length is too short: %d", len(data))
	}

	// Check if this is a WeChat v4 dat file
	if len(data) >= 6 {
		for _, format := range V4Formats {
			if bytes.Equal(data[:4], format.Header) {
				return Dat2ImageV4(data, format.AesKey)
			}
		}
	}

	// For older WeChat versions, use XOR decryption
	findFormat := func(data []byte, header []byte) bool {
		xorBit := data[0] ^ header[0]
		for i := 0; i < len(header); i++ {
			if data[i]^header[i] != xorBit {
				return false
			}
		}
		return true
	}

	var xorBit byte
	var found bool
	var ext string
	for _, format := range Formats {
		if found = findFormat(data, format.Header); found {
			xorBit = data[0] ^ format.Header[0]
			ext = format.Ext
			break
		}
	}

	if !found {
		return nil, "", fmt.Errorf("unknown image type: %x %x", data[0], data[1])
	}

	// Apply XOR decryption
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ xorBit
	}

	return out, ext, nil
}

// calculateXorKeyV4 calculates the XOR key for WeChat v4 dat files
// by analyzing the file tail against known JPG ending bytes (FF D9)
func calculateXorKeyV4(data []byte) (byte, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("data too short to calculate XOR key")
	}

	// Get the last two bytes of the file
	fileTail := data[len(data)-2:]

	// Assuming it's a JPG file, the tail should be FF D9
	xorKeys := make([]byte, 2)
	for i := 0; i < 2; i++ {
		xorKeys[i] = fileTail[i] ^ JpgTail[i]
	}

	// Verify that both bytes yield the same XOR key
	if xorKeys[0] == xorKeys[1] {
		return xorKeys[0], nil
	}

	// If inconsistent, return the first byte as key with a warning
	return xorKeys[0], fmt.Errorf("inconsistent XOR key, using first byte: 0x%x", xorKeys[0])
}

// ScanAndSetXorKey scans a directory for "_t.dat" files to calculate and set
// the global XOR key for WeChat v4 dat files
// Returns the found key and any error encountered
func ScanAndSetXorKey(dirPath string) (byte, error) {
	// Walk the directory recursively
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process "_t.dat" files (thumbnail files)
		if !strings.HasSuffix(info.Name(), "_t.dat") {
			return nil
		}

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check if it's a WeChat v4 dat file
		if len(data) < 6 || (!bytes.Equal(data[:4], V4Format1.Header) && !bytes.Equal(data[:4], V4Format2.Header)) {
			return nil
		}

		// Parse file header
		if len(data) < 15 {
			return nil
		}

		// Get XOR encryption length
		xorEncryptLen := binary.LittleEndian.Uint32(data[10:14])

		// Get data after header
		fileData := data[15:]

		// Skip if there's no XOR-encrypted part
		if xorEncryptLen == 0 || uint32(len(fileData)) <= uint32(len(fileData))-xorEncryptLen {
			return nil
		}

		// Get XOR-encrypted part
		xorData := fileData[uint32(len(fileData))-xorEncryptLen:]

		// Calculate XOR key
		key, err := calculateXorKeyV4(xorData)
		if err != nil {
			return nil
		}

		// Set global XOR key
		V4XorKey = key

		// Stop traversal after finding a valid key
		return filepath.SkipAll
	})

	if err != nil && err != filepath.SkipAll {
		return V4XorKey, fmt.Errorf("error scanning directory: %v", err)
	}

	return V4XorKey, nil
}

func SetAesKey(key string) {
	if key == "" {
		return
	}
	decoded, err := hex.DecodeString(key)
	if err != nil {
		log.Error().Err(err).Msg("invalid aes key")
		return
	}
	V4Format2.AesKey = decoded
}

// Dat2ImageV4 processes WeChat v4 dat image files
// WeChat v4 uses a combination of AES-ECB and XOR encryption
func Dat2ImageV4(data []byte, aeskey []byte) ([]byte, string, error) {
	if len(data) < 15 {
		return nil, "", fmt.Errorf("data length is too short for WeChat v4 format: %d", len(data))
	}

	// Parse dat file header:
	// - 6 bytes: 0x07085631 or 0x07085632 (dat file identifier)
	// - 4 bytes: int (little-endian) AES-ECB128 encryption length
	// - 4 bytes: int (little-endian) XOR encryption length
	// - 1 byte:  0x01 (unknown)

	// Read AES encryption length
	aesEncryptLen := binary.LittleEndian.Uint32(data[6:10])
	// Read XOR encryption length
	xorEncryptLen := binary.LittleEndian.Uint32(data[10:14])

	// Data after header
	fileData := data[15:]

	// AES encrypted part (max 1KB)
	// Round up to multiple of 16 bytes for AES block size
	aesEncryptLen0 := (aesEncryptLen)/16*16 + 16
	if aesEncryptLen0 > uint32(len(fileData)) {
		aesEncryptLen0 = uint32(len(fileData))
	}

	// Decrypt AES part
	aesDecryptedData, err := decryptAESECB(fileData[:aesEncryptLen0], aeskey)
	if err != nil {
		return nil, "", fmt.Errorf("AES decrypt error: %v", err)
	}

	// Prepare result buffer
	var result []byte

	// Add decrypted AES part (remove padding if necessary)
	if len(aesDecryptedData) > int(aesEncryptLen) {
		result = append(result, aesDecryptedData[:aesEncryptLen]...)
	} else {
		result = append(result, aesDecryptedData...)
	}

	// Add unencrypted middle part
	middleStart := aesEncryptLen0
	middleEnd := uint32(len(fileData)) - xorEncryptLen
	if middleStart < middleEnd {
		result = append(result, fileData[middleStart:middleEnd]...)
	}

	// Process XOR-encrypted part (file tail)
	if xorEncryptLen > 0 && middleEnd < uint32(len(fileData)) {
		xorData := fileData[middleEnd:]

		// Apply XOR decryption using global key
		xorDecrypted := make([]byte, len(xorData))
		for i := range xorData {
			xorDecrypted[i] = xorData[i] ^ V4XorKey
		}

		result = append(result, xorDecrypted...)
	}

	// Identify image type from decrypted data
	imgType := ""
	for _, format := range Formats {
		if len(result) >= len(format.Header) && bytes.Equal(result[:len(format.Header)], format.Header) {
			imgType = format.Ext
			break
		}
	}

	if imgType == "wxgf" {
		return Wxam2pic(result)
	}

	if imgType == "" {
		return nil, "", fmt.Errorf("unknown image type after decryption")
	}

	return result, imgType, nil
}

// decryptAESECB decrypts data using AES in ECB mode
func decryptAESECB(data, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Create AES cipher
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Ensure data length is a multiple of block size
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("data length is not a multiple of block size")
	}

	decrypted := make([]byte, len(data))

	// ECB mode requires block-by-block decryption
	for bs, be := 0, aes.BlockSize; bs < len(data); bs, be = bs+aes.BlockSize, be+aes.BlockSize {
		cipher.Decrypt(decrypted[bs:be], data[bs:be])
	}

	// Handle PKCS#7 padding
	padding := int(decrypted[len(decrypted)-1])
	if padding > 0 && padding <= aes.BlockSize {
		// Validate padding
		valid := true
		for i := len(decrypted) - padding; i < len(decrypted); i++ {
			if decrypted[i] != byte(padding) {
				valid = false
				break
			}
		}

		if valid {
			return decrypted[:len(decrypted)-padding], nil
		}
	}

	return decrypted, nil
}
