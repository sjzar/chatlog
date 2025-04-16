package wechat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/pkg/filemonitor"
	"github.com/sjzar/chatlog/pkg/util"
)

var (
	DebounceTime = 1 * time.Second
	MaxWaitTime  = 10 * time.Second
)

type Service struct {
	ctx            *ctx.Context
	lastEvents     map[string]time.Time
	pendingActions map[string]bool
	mutex          sync.Mutex
	fm             *filemonitor.FileMonitor
}

func NewService(ctx *ctx.Context) *Service {
	return &Service{
		ctx:            ctx,
		lastEvents:     make(map[string]time.Time),
		pendingActions: make(map[string]bool),
	}
}

// GetWeChatInstances returns all running WeChat instances
func (s *Service) GetWeChatInstances() []*wechat.Account {
	wechat.Load()
	return wechat.GetAccounts()
}

// GetDataKey extracts the encryption key from a WeChat process
func (s *Service) GetDataKey(info *wechat.Account) (string, error) {
	if info == nil {
		return "", fmt.Errorf("no WeChat instance selected")
	}

	key, err := info.GetKey(context.Background())
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *Service) StartAutoDecrypt() error {
	dbGroup, err := filemonitor.NewFileGroup("wechat", s.ctx.DataDir, `.*\.db$`, []string{"fts"})
	if err != nil {
		return err
	}
	dbGroup.AddCallback(s.DecryptFileCallback)

	s.fm = filemonitor.NewFileMonitor()
	s.fm.AddGroup(dbGroup)
	if err := s.fm.Start(); err != nil {
		log.Debug().Err(err).Msg("failed to start file monitor")
		return err
	}
	return nil
}

func (s *Service) StopAutoDecrypt() error {
	if s.fm != nil {
		if err := s.fm.Stop(); err != nil {
			return err
		}
	}
	s.fm = nil
	return nil
}

func (s *Service) DecryptFileCallback(event fsnotify.Event) error {
	if event.Op.Has(fsnotify.Chmod) || !event.Op.Has(fsnotify.Write) {
		return nil
	}

	s.mutex.Lock()
	s.lastEvents[event.Name] = time.Now()

	if !s.pendingActions[event.Name] {
		s.pendingActions[event.Name] = true
		s.mutex.Unlock()
		go s.waitAndProcess(event.Name)
	} else {
		s.mutex.Unlock()
	}

	return nil
}

func (s *Service) waitAndProcess(dbFile string) {
	start := time.Now()
	for {
		time.Sleep(DebounceTime)

		s.mutex.Lock()
		lastEventTime := s.lastEvents[dbFile]
		elapsed := time.Since(lastEventTime)
		totalElapsed := time.Since(start)

		if elapsed >= DebounceTime || totalElapsed >= MaxWaitTime {
			s.pendingActions[dbFile] = false
			s.mutex.Unlock()

			log.Debug().Msgf("Processing file: %s", dbFile)
			s.DecryptDBFile(dbFile)
			return
		}
		s.mutex.Unlock()
	}
}

func (s *Service) DecryptDBFile(dbFile string) error {

	decryptor, err := decrypt.NewDecryptor(s.ctx.Platform, s.ctx.Version)
	if err != nil {
		return err
	}

	output := filepath.Join(s.ctx.WorkDir, dbFile[len(s.ctx.DataDir):])
	if err := util.PrepareDir(filepath.Dir(output)); err != nil {
		return err
	}

	outputTemp := output + ".tmp"
	outputFile, err := os.Create(outputTemp)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer func() {
		outputFile.Close()
		if err := os.Rename(outputTemp, output); err != nil {
			log.Debug().Err(err).Msgf("failed to rename %s to %s", outputTemp, output)
		}
	}()

	if err := decryptor.Decrypt(context.Background(), dbFile, s.ctx.DataKey, outputFile); err != nil {
		if err == errors.ErrAlreadyDecrypted {
			if data, err := os.ReadFile(dbFile); err == nil {
				outputFile.Write(data)
			}
			return nil
		}
		log.Err(err).Msgf("failed to decrypt %s", dbFile)
		return err
	}

	log.Debug().Msgf("Decrypted %s to %s", dbFile, output)

	return nil
}

func (s *Service) DecryptDBFiles() error {
	dbGroup, err := filemonitor.NewFileGroup("wechat", s.ctx.DataDir, `.*\.db$`, []string{"fts"})
	if err != nil {
		return err
	}

	dbFiles, err := dbGroup.List()
	if err != nil {
		return err
	}

	for _, dbFile := range dbFiles {
		if err := s.DecryptDBFile(dbFile); err != nil {
			log.Debug().Msgf("DecryptDBFile %s failed: %v", dbFile, err)
			continue
		}
	}

	return nil
}
