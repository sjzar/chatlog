package conf

type Webhook struct {
	Host    string         `mapstructure:"host"`
	DelayMs int64          `mapstructure:"delay_ms"`
	Items   []*WebhookItem `mapstructure:"items"`
}

type WebhookItem struct {
	Type     string `mapstructure:"type"`
	URL      string `mapstructure:"url"`
	Talker   string `mapstructure:"talker"`
	Sender   string `mapstructure:"sender"`
	Keyword  string `mapstructure:"keyword"`
	Disabled bool   `mapstructure:"disabled"`
}
