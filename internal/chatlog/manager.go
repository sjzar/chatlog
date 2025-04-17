package chatlog

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/chatlog/database"
	"github.com/sjzar/chatlog/internal/chatlog/http"
	"github.com/sjzar/chatlog/internal/chatlog/mcp"
	"github.com/sjzar/chatlog/internal/chatlog/wechat"
	iwechat "github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/pkg/util"
	"github.com/sjzar/chatlog/pkg/util/dat2img"
)

// Manager 管理聊天日志应用
type Manager struct {
	conf *conf.Service
	ctx  *ctx.Context

	// Services
	db     *database.Service
	http   *http.Service
	mcp    *mcp.Service
	wechat *wechat.Service

	// Terminal UI
	app *App
}

func New(configPath string) (*Manager, error) {

	// 创建配置服务
	conf, err := conf.NewService(configPath)
	if err != nil {
		return nil, err
	}

	// 创建应用上下文
	ctx := ctx.New(conf)

	wechat := wechat.NewService(ctx)

	db := database.NewService(ctx)

	mcp := mcp.NewService(ctx, db)

	http := http.NewService(ctx, db, mcp)

	return &Manager{
		conf:   conf,
		ctx:    ctx,
		db:     db,
		mcp:    mcp,
		http:   http,
		wechat: wechat,
	}, nil
}

func (m *Manager) Run() error {

	m.ctx.WeChatInstances = m.wechat.GetWeChatInstances()
	if len(m.ctx.WeChatInstances) >= 1 {
		m.ctx.SwitchCurrent(m.ctx.WeChatInstances[0])
	}

	if m.ctx.HTTPEnabled {
		// 启动HTTP服务
		if err := m.StartService(); err != nil {
			m.StopService()
		}
	}
	// 启动终端UI
	m.app = NewApp(m.ctx, m)
	m.app.Run() // 阻塞
	return nil
}

func (m *Manager) Switch(info *iwechat.Account, history string) error {
	if m.ctx.AutoDecrypt {
		if err := m.StopAutoDecrypt(); err != nil {
			return err
		}
	}
	if m.ctx.HTTPEnabled {
		if err := m.stopService(); err != nil {
			return err
		}
	}
	if info != nil {
		m.ctx.SwitchCurrent(info)
	} else {
		m.ctx.SwitchHistory(history)
	}

	if m.ctx.HTTPEnabled {
		// 启动HTTP服务
		if err := m.StartService(); err != nil {
			log.Info().Err(err).Msg("启动服务失败")
			m.StopService()
		}
	}
	return nil
}

func (m *Manager) StartService() error {

	// 按依赖顺序启动服务
	if err := m.db.Start(); err != nil {
		return err
	}

	if err := m.mcp.Start(); err != nil {
		m.db.Stop() // 回滚已启动的服务
		return err
	}

	if err := m.http.Start(); err != nil {
		m.mcp.Stop() // 回滚已启动的服务
		m.db.Stop()
		return err
	}

	// 如果是 4.0 版本，更新下 xorkey
	if m.ctx.Version == 4 {
		go dat2img.ScanAndSetXorKey(m.ctx.DataDir)
	}

	// 更新状态
	m.ctx.SetHTTPEnabled(true)

	return nil
}

func (m *Manager) StopService() error {
	if err := m.stopService(); err != nil {
		return err
	}

	// 更新状态
	m.ctx.SetHTTPEnabled(false)

	return nil
}

func (m *Manager) stopService() error {
	// 按依赖的反序停止服务
	var errs []error

	if err := m.http.Stop(); err != nil {
		errs = append(errs, err)
	}

	if err := m.mcp.Stop(); err != nil {
		errs = append(errs, err)
	}

	if err := m.db.Stop(); err != nil {
		errs = append(errs, err)
	}

	// 如果有错误，返回第一个错误
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (m *Manager) SetHTTPAddr(text string) error {
	var addr string
	if util.IsNumeric(text) {
		addr = fmt.Sprintf("127.0.0.1:%s", text)
	} else if strings.HasPrefix(text, "http://") {
		addr = strings.TrimPrefix(text, "http://")
	} else if strings.HasPrefix(text, "https://") {
		addr = strings.TrimPrefix(text, "https://")
	} else {
		addr = text
	}
	m.ctx.SetHTTPAddr(addr)
	return nil
}

func (m *Manager) GetDataKey() error {
	if m.ctx.Current == nil {
		return fmt.Errorf("未选择任何账号")
	}
	if _, err := m.wechat.GetDataKey(m.ctx.Current); err != nil {
		return err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()
	return nil
}

func (m *Manager) DecryptDBFiles() error {
	if m.ctx.DataKey == "" {
		if m.ctx.Current == nil {
			return fmt.Errorf("未选择任何账号")
		}
		if err := m.GetDataKey(); err != nil {
			return err
		}
	}
	if m.ctx.WorkDir == "" {
		m.ctx.WorkDir = util.DefaultWorkDir(m.ctx.Account)
	}

	if err := m.wechat.DecryptDBFiles(); err != nil {
		return err
	}
	m.ctx.Refresh()
	m.ctx.UpdateConfig()
	return nil
}

func (m *Manager) StartAutoDecrypt() error {
	if m.ctx.DataKey == "" || m.ctx.DataDir == "" {
		return fmt.Errorf("请先获取密钥")
	}
	if m.ctx.WorkDir == "" {
		return fmt.Errorf("请先执行解密数据")
	}

	if err := m.wechat.StartAutoDecrypt(); err != nil {
		return err
	}

	m.ctx.SetAutoDecrypt(true)
	return nil
}

func (m *Manager) StopAutoDecrypt() error {
	if err := m.wechat.StopAutoDecrypt(); err != nil {
		return err
	}

	m.ctx.SetAutoDecrypt(false)
	return nil
}

func (m *Manager) RefreshSession() error {
	if m.db.GetDB() == nil {
		if err := m.db.Start(); err != nil {
			return err
		}
	}
	resp, err := m.db.GetSessions("", 1, 0)
	if err != nil {
		return err
	}
	if len(resp.Items) == 0 {
		return nil
	}
	m.ctx.LastSession = resp.Items[0].NTime
	return nil
}

func (m *Manager) CommandKey(pid int) (string, error) {
	instances := m.wechat.GetWeChatInstances()
	if len(instances) == 0 {
		return "", fmt.Errorf("wechat process not found")
	}
	if len(instances) == 1 {
		return instances[0].GetKey(context.Background())
	}
	if pid == 0 {
		str := "Select a process:\n"
		for _, ins := range instances {
			str += fmt.Sprintf("PID: %d. %s[Version: %s Data Dir: %s ]\n", ins.PID, ins.Name, ins.FullVersion, ins.DataDir)
		}
		return str, nil
	}
	for _, ins := range instances {
		if ins.PID == uint32(pid) {
			return ins.GetKey(context.Background())
		}
	}
	return "", fmt.Errorf("wechat process not found")
}

func (m *Manager) CommandDecrypt(dataDir string, workDir string, key string, platform string, version int) error {
	if dataDir == "" {
		return fmt.Errorf("dataDir is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if workDir == "" {
		workDir = util.DefaultWorkDir(filepath.Base(filepath.Dir(dataDir)))
	}
	m.ctx.DataDir = dataDir
	m.ctx.WorkDir = workDir
	m.ctx.DataKey = key
	m.ctx.Platform = platform
	m.ctx.Version = version
	if err := m.wechat.DecryptDBFiles(); err != nil {
		return err
	}

	return nil
}

func (m *Manager) CommandHTTPServer(addr string, dataDir string, workDir string, platform string, version int) error {

	if addr == "" {
		addr = "127.0.0.1:5030"
	}

	if workDir == "" {
		return fmt.Errorf("workDir is required")
	}

	if platform == "" {
		return fmt.Errorf("platform is required")
	}

	if version == 0 {
		return fmt.Errorf("version is required")
	}

	m.ctx.HTTPAddr = addr
	m.ctx.DataDir = dataDir
	m.ctx.WorkDir = workDir
	m.ctx.Platform = platform
	m.ctx.Version = version

	// 如果是 4.0 版本，更新下 xorkey
	if m.ctx.Version == 4 && m.ctx.DataDir != "" {
		go dat2img.ScanAndSetXorKey(m.ctx.DataDir)
	}

	// 按依赖顺序启动服务
	if err := m.db.Start(); err != nil {
		return err
	}

	if err := m.mcp.Start(); err != nil {
		return err
	}

	return m.http.ListenAndServe()
}
