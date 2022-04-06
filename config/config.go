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
	DB          *sql.DB
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
	c.DB = database
	c.Bot = bot
	c.BuildInfo = b
	return c, nil
}

// Close free resources.
func (c *Config) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.DB.Close()
}

// Context returns context with timeout.
func (c *Config) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}
