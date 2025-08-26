package conf

const (
	DefalutHTTPAddr = "0.0.0.0:5030"
)

type ServerConfig struct {
	Type        string   `mapstructure:"type"`
	Platform    string   `mapstructure:"platform"`
	Version     int      `mapstructure:"version"`
	FullVersion string   `mapstructure:"full_version"`
	DataDir     string   `mapstructure:"data_dir"`
	DataKey     string   `mapstructure:"data_key"`
	ImgKey      string   `mapstructure:"img_key"`
	WorkDir     string   `mapstructure:"work_dir"`
	HTTPAddr    string   `mapstructure:"http_addr"`
	AutoDecrypt bool     `mapstructure:"auto_decrypt"`
	Webhook     *Webhook `mapstructure:"webhook"`
}

var ServerDefaults = map[string]any{}

func (c *ServerConfig) GetDataDir() string {
	return c.DataDir
}

func (c *ServerConfig) GetWorkDir() string {
	return c.WorkDir
}

func (c *ServerConfig) GetPlatform() string {
	return c.Platform
}

func (c *ServerConfig) GetVersion() int {
	return c.Version
}

func (c *ServerConfig) GetDataKey() string {
	return c.DataKey
}

func (c *ServerConfig) GetImgKey() string {
	return c.ImgKey
}

func (c *ServerConfig) GetAutoDecrypt() bool {
	return c.AutoDecrypt
}

func (c *ServerConfig) GetHTTPAddr() string {
	if c.HTTPAddr == "" {
		c.HTTPAddr = DefalutHTTPAddr
	}
	return c.HTTPAddr
}

func (c *ServerConfig) GetWebhook() *Webhook {
	return c.Webhook
}
