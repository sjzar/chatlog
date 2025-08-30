package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/chatlog/conf"
	"github.com/sjzar/chatlog/internal/wechatdb"
)

type Config interface {
	GetWebhook() *conf.Webhook
}

type Webhook interface {
	Do(event fsnotify.Event)
}

type Service struct {
	config *conf.Webhook
	hooks  map[string][]*conf.WebhookItem
}

func New(config Config) *Service {
	s := &Service{
		config: config.GetWebhook(),
	}

	if s.config == nil {
		return s
	}

	hooks := make(map[string][]*conf.WebhookItem)
	for _, item := range s.config.Items {
		if item.Disabled {
			continue
		}
		if item.Type == "" {
			item.Type = "message"
		}
		switch item.Type {
		case "message":
			if hooks["message"] == nil {
				hooks["message"] = make([]*conf.WebhookItem, 0)
			}
			hooks["message"] = append(hooks["message"], item)
		default:
			log.Error().Msgf("unknown webhook type: %s", item.Type)
		}
	}
	s.hooks = hooks

	return s
}

func (s *Service) GetHooks(ctx context.Context, db *wechatdb.DB) []*Group {

	if len(s.hooks) == 0 {
		return nil
	}

	groups := make([]*Group, 0)
	for group, items := range s.hooks {
		hooks := make([]Webhook, 0)
		for _, item := range items {
			hooks = append(hooks, NewMessageWebhook(item, db, s.config.Host))
		}
		groups = append(groups, NewGroup(ctx, group, hooks, s.config.DelayMs))
	}

	return groups
}

type Group struct {
	ctx     context.Context
	group   string
	hooks   []Webhook
	delayMs int64
	ch      chan fsnotify.Event
}

func NewGroup(ctx context.Context, group string, hooks []Webhook, delayMs int64) *Group {
	g := &Group{
		group:   group,
		hooks:   hooks,
		delayMs: delayMs,
		ctx:     ctx,
		ch:      make(chan fsnotify.Event, 1),
	}
	go g.loop()
	return g
}

func (g *Group) Callback(event fsnotify.Event) error {
	// skip remove event
	if !event.Op.Has(fsnotify.Create) {
		return nil
	}

	select {
	case g.ch <- event:
	default:
	}
	return nil
}

func (g *Group) Group() string {
	return g.group
}

func (g *Group) loop() {
	for {
		select {
		case event, ok := <-g.ch:
			if !ok {
				return
			}
			if g.delayMs > 0 {
				time.Sleep(time.Duration(g.delayMs) * time.Millisecond)
			}
			g.do(event)
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Group) do(event fsnotify.Event) {
	for _, hook := range g.hooks {
		go hook.Do(event)
	}
}

type MessageWebhook struct {
	host     string
	conf     *conf.WebhookItem
	client   *http.Client
	db       *wechatdb.DB
	lastTime time.Time
}

func NewMessageWebhook(conf *conf.WebhookItem, db *wechatdb.DB, host string) *MessageWebhook {
	m := &MessageWebhook{
		host:     host,
		conf:     conf,
		client:   &http.Client{Timeout: time.Second * 10},
		db:       db,
		lastTime: time.Now(),
	}
	return m
}

func (m *MessageWebhook) Do(event fsnotify.Event) {
	messages, err := m.db.GetMessages(m.lastTime, time.Now().Add(time.Minute*10), m.conf.Talker, m.conf.Sender, m.conf.Keyword, 0, 0)
	if err != nil {
		log.Error().Err(err).Msgf("get messages failed")
		return
	}

	if len(messages) == 0 {
		return
	}

	m.lastTime = messages[len(messages)-1].Time.Add(time.Second)

	for _, message := range messages {
		message.SetContent("host", m.host)
		message.Content = message.PlainTextContent()
	}

	ret := map[string]any{
		"talker":   m.conf.Talker,
		"sender":   m.conf.Sender,
		"keyword":  m.conf.Keyword,
		"lastTime": m.lastTime.Format(time.DateTime),
		"length":   len(messages),
		"messages": messages,
	}
	body, _ := json.Marshal(ret)
	req, _ := http.NewRequest("POST", m.conf.URL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	log.Info().Msgf("post messages to %s, body: %s", m.conf.URL, string(body))
	resp, err := m.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("post messages failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("post messages failed, status code: %d", resp.StatusCode)
	}
}
