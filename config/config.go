package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	"github.com/pelletier/go-toml/v2"
	"github.com/z0rr0/aoapi"
	"github.com/z0rr0/tgtpgybot/ygpt"

	"github.com/z0rr0/gobot/random"
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
	Debug        bool   `toml:"debug"`
	Storage      string `toml:"storage"`
	Timeout      uint64 `toml:"timeout"`
	Workers      int    `toml:"workers"`
	SecureRandom bool   `toml:"secure_random"`
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

// GPT is a ChatGPT API configuration settings.
type GPT struct {
	Bearer       string       `toml:"bearer"`
	Organization string       `toml:"organization"`
	MaxTokens    uint         `toml:"max_tokens"`
	URL          string       `toml:"url"`
	Proxy        string       `toml:"proxy"`
	Temperature  float32      `toml:"temperature"`
	Client       *http.Client `toml:"-"`
}

// Response returns ChatGPT response.
func (gpt *GPT) Response(ctx context.Context, content string) (string, error) {
	if gpt.Client == nil {
		return "", fmt.Errorf("gpt client is not defined")
	}

	request := &aoapi.CompletionRequest{
		Model:       aoapi.ModelGPT35TurboK16,
		Messages:    []aoapi.Message{{Role: aoapi.RoleUser, Content: content}},
		MaxTokens:   gpt.MaxTokens,
		Temperature: &gpt.Temperature,
	}

	params := aoapi.Params{
		Bearer:       gpt.Bearer,
		Organization: gpt.Organization,
		URL:          gpt.URL,
		StopMarker:   "....",
	}

	resp, err := aoapi.Completion(ctx, gpt.Client, request, params)
	if err != nil {
		return "", fmt.Errorf("gpt completion error: %w", err)
	}

	return resp.String(), nil
}

// YandexGPT is a Yandex GPT API configuration settings.
type YandexGPT struct {
	APIKey string       `toml:"api_key"`
	URL    string       `toml:"url"`
	Proxy  string       `toml:"proxy"`
	Client *http.Client `toml:"-"`
}

// Response returns Yandex GPT response.
func (yt *YandexGPT) Response(ctx context.Context, content string) (string, error) {
	if yt.Client == nil {
		return "", fmt.Errorf("yandex gpt client is not defined")
	}

	request := &ygpt.ChatRequest{APIKey: yt.APIKey, URL: yt.URL, Text: content}

	resp, err := ygpt.GenerationChat(ctx, yt.Client, request)
	if err != nil {
		return "", fmt.Errorf("yandex gpt completion error: %w", err)
	}

	return resp.String(), nil
}

// Config is common configuration struct.
type Config struct {
	sync.Mutex
	M          Main      `toml:"main"`
	B          Bot       `toml:"bot"`
	G          GPT       `toml:"gpt"`
	Y          YandexGPT `toml:"yandex_gpt"`
	L          Log       `toml:"log"`
	Bt         *botgolang.Bot
	DB         *sql.DB
	BuildInfo  *BuildInfo
	RandSource rand.Source
	timeout    time.Duration
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

	if c.M.Workers < 1 {
		return nil, errors.New("number of workers must be greater than 0")
	}

	if err = c.initLog(); err != nil {
		return nil, fmt.Errorf("log init: %w", err)
	}

	if err = c.initGPT(); err != nil {
		return nil, fmt.Errorf("GPT init: %w", err)
	}

	if err = c.initYandexGPT(); err != nil {
		return nil, fmt.Errorf("yandex GPT init: %w", err)
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

	c.RandSource = random.New(c.M.SecureRandom, 0)
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

func gptInit(key, uri, proxy string) (*http.Client, error) {
	if (key == "") || (uri == "") {
		// no config settings
		return nil, nil
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}

		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}, nil
	}

	return &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}, nil
}

// initGPT initializes GPT client.
func (c *Config) initGPT() error {
	if c.G.Client != nil {
		return nil
	}

	client, err := gptInit(c.G.Bearer, c.G.URL, c.G.Proxy)
	if err != nil {
		return err
	}

	c.G.Client = client
	return nil
}

// initYandexGPT initializes Yandex GPT client.
func (c *Config) initYandexGPT() error {
	if c.Y.Client != nil {
		return nil
	}

	client, err := gptInit(c.Y.APIKey, c.Y.URL, c.Y.Proxy)
	if err != nil {
		return err
	}

	c.Y.Client = client
	return nil
}
