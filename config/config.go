package config

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	"github.com/pelletier/go-toml"

	"github.com/z0rr0/gobot/db"
)

// Bot contains base API configuration parameters.
type Bot struct {
	ID    string `toml:"id"`
	Nick  string `toml:"nick"`
	Token string `toml:"token"`
	ULR   string `toml:"url"`
	Src   string `toml:"src"`
}

// Main is a basic configuration settings.
type Main struct {
	Debug   bool   `toml:"debug"`
	Storage string `toml:"storage"`
	Timeout uint64 `toml:"timeout"`
	Workers uint   `toml:"workers"`
}

// BuildInfo is a build information.
type BuildInfo struct {
	Name      string
	Hash      string
	Revision  string
	GoVersion string
	Date      string
	URL       string
}

// Config is common configuration struct.
type Config struct {
	sync.Mutex
	BotSettings Bot  `toml:"bot"`
	M           Main `toml:"main"`
	Bot         *botgolang.Bot
	Db          *sql.DB
	BuildInfo   *BuildInfo
	timeout     time.Duration
}

// New returns new configuration.
func New(fileName string, b *BuildInfo, server *httptest.Server) (*Config, error) {
	fullPath, err := filepath.Abs(strings.Trim(fileName, " "))
	if err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config existing: %w", err)
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}
	c := &Config{}
	if err = toml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("config parsing: %w", err)
	}
	client := http.DefaultClient
	if server != nil {
		client = server.Client()
		c.BotSettings.ULR = server.URL
	}
	bot, err := botgolang.NewBot(
		c.BotSettings.Token,
		botgolang.BotDebug(c.M.Debug),
		botgolang.BotApiURL(c.BotSettings.ULR),
		botgolang.BotHTTPClient(*client),
	)
	if err != nil {
		return nil, fmt.Errorf("can not init bot: %w", err)
	}
	database, err := sql.Open("sqlite3", c.M.Storage)
	if err != nil {
		return nil, fmt.Errorf("database file: %w", err)
	}
	b.URL = c.BotSettings.Src
	c.timeout = time.Duration(c.M.Timeout) * time.Second
	c.Db = database
	c.Bot = bot
	c.BuildInfo = b
	return c, nil
}

// Close free resources.
func (c *Config) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.Db.Close()
}

// Context returns context with timeout.
func (c *Config) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

// SaveChat saves chat info.
func (c *Config) SaveChat(chat *db.Chat) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return chat.Update(ctx, c.Db)
}

// StartBot starts bot.
func (c *Config) StartBot(event *botgolang.Event) error {
	return db.UpsertActive(c.Db, event.Payload.Chat.ID, true, c.timeout)
}

// StopBot stops bot.
func (c *Config) StopBot(event *botgolang.Event) error {
	return db.UpsertActive(c.Db, event.Payload.Chat.ID, false, c.timeout)
}

//// Chat returns chat by ID.
//func (c *Config) Chat(event *botgolang.Event) (*db.Chat, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
//	defer cancel()
//	chat, err := db.Get(ctx, c.Db, event.Payload.Chat.ID)
//	if err != nil {
//		if err == sql.ErrNoRows {
//			// unknown chat
//			return &db.Chat{ID: event.Payload.Chat.ID}, nil
//		}
//		return nil, fmt.Errorf("chat load: %w", err)
//	}
//	return chat, nil
//}
