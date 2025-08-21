package database

import (
	"time"

	"github.com/sjzar/chatlog/internal/model"
	"github.com/sjzar/chatlog/internal/wechatdb"
)

const (
	StateInit = iota
	StateDecrypting
	StateReady
	StateError
)

type Service struct {
	State    int
	StateMsg string
	conf     Config
	db       *wechatdb.DB
}

type Config interface {
	GetWorkDir() string
	GetPlatform() string
	GetVersion() int
}

func NewService(conf Config) *Service {
	return &Service{
		conf: conf,
	}
}

func (s *Service) Start() error {
	db, err := wechatdb.New(s.conf.GetWorkDir(), s.conf.GetPlatform(), s.conf.GetVersion())
	if err != nil {
		return err
	}
	s.SetReady()
	s.db = db
	return nil
}

func (s *Service) Stop() error {
	if s.db != nil {
		s.db.Close()
	}
	s.SetInit()
	s.db = nil
	return nil
}

func (s *Service) SetInit() {
	s.State = StateInit
}

func (s *Service) SetDecrypting() {
	s.State = StateDecrypting
}

func (s *Service) SetReady() {
	s.State = StateReady
}

func (s *Service) SetError(msg string) {
	s.State = StateError
	s.StateMsg = msg
}

func (s *Service) GetDB() *wechatdb.DB {
	return s.db
}

func (s *Service) GetMessages(start, end time.Time, talker string, sender string, keyword string, limit, offset int) ([]*model.Message, error) {
	return s.db.GetMessages(start, end, talker, sender, keyword, limit, offset)
}

func (s *Service) GetContacts(key string, limit, offset int) (*wechatdb.GetContactsResp, error) {
	return s.db.GetContacts(key, limit, offset)
}

func (s *Service) GetChatRooms(key string, limit, offset int) (*wechatdb.GetChatRoomsResp, error) {
	return s.db.GetChatRooms(key, limit, offset)
}

// GetSession retrieves session information
func (s *Service) GetSessions(key string, limit, offset int) (*wechatdb.GetSessionsResp, error) {
	return s.db.GetSessions(key, limit, offset)
}

func (s *Service) GetMedia(_type string, key string) (*model.Media, error) {
	return s.db.GetMedia(_type, key)
}

// Close closes the database connection
func (s *Service) Close() {
	// Add cleanup code if needed
	s.db.Close()
}
