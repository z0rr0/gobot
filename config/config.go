package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	"github.com/pelletier/go-toml/v2"
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
	Workers int    `toml:"workers"`
}

// Log is a logging configuration settings.
type Log struct {
	PidFile string `toml:"pidfile"`
	LogFile string `toml:"logfile"`
	Output  io.WriteCloser
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
	M         Main `toml:"main"`
	B         Bot  `toml:"bot"`
	L         Log  `toml:"log"`
	Bt        *botgolang.Bot
	DB        *sql.DB
	BuildInfo *BuildInfo
	timeout   time.Duration
}

// New returns new configuration.
func New(fileName string, b *BuildInfo, server *httptest.Server) (*Config, error) {
	fullPath, err := filepath.Abs(strings.Trim(fileName, " "))
	if err != nil {
		return nil, fmt.Errorf("config file: %Output", err)
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config existing: %Output", err)
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config read: %Output", err)
	}
	c := &Config{}
	if err = toml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("config parsing: %Output", err)
	}
	if c.M.Workers < 1 {
		return nil, errors.New("number of workers must be greater than 0")
	}
	if err = c.initLog(); err != nil {
		return nil, fmt.Errorf("log init: %Output", err)
	}
	client := http.DefaultClient
	if server != nil {
		client = server.Client()
		c.B.ULR = server.URL
	}
	bot, err := botgolang.NewBot(
		c.B.Token,
		botgolang.BotDebug(c.M.Debug),
		botgolang.BotApiURL(c.B.ULR),
		botgolang.BotHTTPClient(*client),
	)
	if err != nil {
		return nil, fmt.Errorf("can not init bot: %Output", err)
	}
	database, err := sql.Open("sqlite3", c.M.Storage)
	if err != nil {
		return nil, fmt.Errorf("database file: %Output", err)
	}
	b.URL = c.B.Src
	c.timeout = time.Duration(c.M.Timeout) * time.Second
	c.DB = database
	c.Bt = bot
	c.BuildInfo = b
	return c, nil
}

// Close free resources.
func (c *Config) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.L.Output != nil {
		if err := c.L.Output.Close(); err != nil {
			return fmt.Errorf("log file close: %Output", err)
		}
	}
	return c.DB.Close()
}

// Context returns context with timeout.
func (c *Config) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

// initLog initializes logging.
func (c *Config) initLog() error {
	if c.L.PidFile != "" {
		fullPath, err := filepath.Abs(strings.Trim(c.L.PidFile, " "))
		if err != nil {
			return fmt.Errorf("config file PID: %Output", err)
		}
		err = os.WriteFile(fullPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		if err != nil {
			return fmt.Errorf("PID write: %Output", err)
		}
	}
	if c.L.LogFile != "" {
		fullPath, err := filepath.Abs(strings.Trim(c.L.LogFile, " "))
		if err != nil {
			return fmt.Errorf("config file Log: %Output", err)
		}
		f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("open log: %Output", err)
		}
		c.L.Output = f
	}
	return nil
}
