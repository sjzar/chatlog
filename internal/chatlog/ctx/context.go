package ctx

import (
	"sync"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/wechat"
	"github.com/sjzar/chatlog/pkg/util"
)

// Context is a context for a chatlog.
// It is used to store information about the chatlog.
type Context struct {
	conf *conf.Service
	mu   sync.RWMutex

	History map[string]conf.ProcessConfig

	// 微信账号相关状态
	Account     string
	Platform    string
	Version     int
	FullVersion string
	DataDir     string
	DataKey     string
	DataUsage   string

	// 工作目录相关状态
	WorkDir   string
	WorkUsage string

	// HTTP服务相关状态
	HTTPEnabled bool
	HTTPAddr    string

	// 自动解密
	AutoDecrypt bool
	LastSession time.Time

	// 当前选中的微信实例
	Current *wechat.Account
	PID     int
	ExePath string
	Status  string

	// 所有可用的微信实例
	WeChatInstances []*wechat.Account
}

func New(conf *conf.Service) *Context {
	ctx := &Context{
		conf: conf,
	}

	ctx.loadConfig()

	return ctx
}

func (c *Context) loadConfig() {
	conf := c.conf.GetConfig()
	c.History = conf.ParseHistory()
	c.SwitchHistory(conf.LastAccount)
	c.Refresh()
}

func (c *Context) SwitchHistory(account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Current = nil
	c.PID = 0
	c.ExePath = ""
	c.Status = ""
	history, ok := c.History[account]
	if ok {
		c.Account = history.Account
		c.Platform = history.Platform
		c.Version = history.Version
		c.FullVersion = history.FullVersion
		c.DataKey = history.DataKey
		c.DataDir = history.DataDir
		c.WorkDir = history.WorkDir
		c.HTTPEnabled = history.HTTPEnabled
		c.HTTPAddr = history.HTTPAddr
	} else {
		c.Account = ""
		c.Platform = ""
		c.Version = 0
		c.FullVersion = ""
		c.DataKey = ""
		c.DataDir = ""
		c.WorkDir = ""
		c.HTTPEnabled = false
		c.HTTPAddr = ""
	}
}

func (c *Context) SwitchCurrent(info *wechat.Account) {
	c.SwitchHistory(info.Name)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Current = info
	c.Refresh()

}
func (c *Context) Refresh() {
	if c.Current != nil {
		c.Account = c.Current.Name
		c.Platform = c.Current.Platform
		c.Version = c.Current.Version
		c.FullVersion = c.Current.FullVersion
		c.PID = int(c.Current.PID)
		c.ExePath = c.Current.ExePath
		c.Status = c.Current.Status
		if c.Current.Key != "" && c.Current.Key != c.DataKey {
			c.DataKey = c.Current.Key
		}
		if c.Current.DataDir != "" && c.Current.DataDir != c.DataDir {
			c.DataDir = c.Current.DataDir
		}
	}
	if c.DataUsage == "" && c.DataDir != "" {
		go func() {
			c.DataUsage = util.GetDirSize(c.DataDir)
		}()
	}
	if c.WorkUsage == "" && c.WorkDir != "" {
		go func() {
			c.WorkUsage = util.GetDirSize(c.WorkDir)
		}()
	}
}

func (c *Context) SetHTTPEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HTTPEnabled = enabled
	c.UpdateConfig()
}

func (c *Context) SetHTTPAddr(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HTTPAddr = addr
	c.UpdateConfig()
}

func (c *Context) SetWorkDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WorkDir = dir
	c.UpdateConfig()
	c.Refresh()
}

func (c *Context) SetDataDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DataDir = dir
	c.UpdateConfig()
	c.Refresh()
}

func (c *Context) SetAutoDecrypt(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AutoDecrypt = enabled
	c.UpdateConfig()
}

// 更新配置
func (c *Context) UpdateConfig() {
	pconf := conf.ProcessConfig{
		Type:        "wechat",
		Account:     c.Account,
		Platform:    c.Platform,
		Version:     c.Version,
		FullVersion: c.FullVersion,
		DataDir:     c.DataDir,
		DataKey:     c.DataKey,
		WorkDir:     c.WorkDir,
		HTTPEnabled: c.HTTPEnabled,
		HTTPAddr:    c.HTTPAddr,
	}
	conf := c.conf.GetConfig()
	conf.UpdateHistory(c.Account, pconf)
}
