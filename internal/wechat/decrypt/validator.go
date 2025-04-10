package decrypt

import (
	"path/filepath"

	"github.com/sjzar/chatlog/internal/wechat/decrypt/common"
)

type Validator struct {
	platform  string
	version   int
	dbPath    string
	decryptor Decryptor
	dbFile    *common.DBFile
}

// NewValidator 创建一个仅用于验证的验证器
func NewValidator(platform string, version int, dataDir string) (*Validator, error) {
	dbFile := GetSimpleDBFile(platform, version)
	dbPath := filepath.Join(dataDir + "/" + dbFile)
	return NewValidatorWithFile(platform, version, dbPath)
}

func NewValidatorWithFile(platform string, version int, dbPath string) (*Validator, error) {
	decryptor, err := NewDecryptor(platform, version)
	if err != nil {
		return nil, err
	}
	d, err := common.OpenDBFile(dbPath, decryptor.GetPageSize())
	if err != nil {
		return nil, err
	}

	return &Validator{
		platform:  platform,
		version:   version,
		dbPath:    dbPath,
		decryptor: decryptor,
		dbFile:    d,
	}, nil
}

func (v *Validator) Validate(key []byte) bool {
	return v.decryptor.Validate(v.dbFile.FirstPage, key)
}

func GetSimpleDBFile(platform string, version int) string {
	switch {
	case platform == "windows" && version == 3:
		return "Msg\\Misc.db"
	case platform == "windows" && version == 4:
		return "db_storage\\message\\message_0.db"
	case platform == "darwin" && version == 3:
		return "Message/msg_0.db"
	case platform == "darwin" && version == 4:
		return "db_storage/message/message_0.db"
	}
	return ""

}
