package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	botgolang "github.com/mail-ru-im/bot-golang"
)

// Bot contains base API configuration parameters.
type Bot struct {
	ID    string `json:"id"`
	Nick  string `json:"nick"`
	Token string `json:"token"`
	ULR   string `json:"url"`
	Debug bool   `json:"debug"`
}

// Config is common configuration struct.
type Config struct {
	sync.Mutex
	BotSettings  Bot `json:"bot"`
	Bot          *botgolang.Bot
	StoppedChats map[string]bool
}

// New returns new configuration.
func New(fileName string) (*Config, error) {
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
	if err = json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("config parsing: %w", err)
	}
	bot, err := botgolang.NewBot(c.BotSettings.Token, botgolang.BotDebug(c.BotSettings.Debug), botgolang.BotApiURL(c.BotSettings.ULR))
	if err != nil {
		return nil, fmt.Errorf("can not init bot: %w", err)
	}
	c.StoppedChats = make(map[string]bool)
	c.Bot = bot
	return c, nil
}
