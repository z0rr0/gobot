package config

import (
	"context"
	"database/sql"
	"fmt"
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
func New(fileName string, b *BuildInfo) (*Config, error) {
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
	bot, err := botgolang.NewBot(c.BotSettings.Token, botgolang.BotDebug(c.M.Debug), botgolang.BotApiURL(c.BotSettings.ULR))
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

func (c *Config) touchChat(chat *db.Chat) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return chat.Upsert(ctx, c.Db)
}

// StartBot starts bot.
func (c *Config) StartBot(event *botgolang.Event) error {
	chat := db.Chat{ID: event.Payload.Chat.ID, Active: true}
	return c.touchChat(&chat)
}

// StopBot stops bot.
func (c *Config) StopBot(event *botgolang.Event) error {
	chat := db.Chat{ID: event.Payload.Chat.ID, Active: false}
	return c.touchChat(&chat)
}
