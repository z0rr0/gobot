package cmd

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
)

var (
	botIDRegexp = regexp.MustCompile(`^\d+$`)
)

// Connector is interface for bot actions.
type Connector interface {
	ChatMembers() ([]botgolang.ChatMember, error)
	SendMessage(msg string) error
	SendURLMessage(msg, txt, url string) error
	Version() *config.BuildInfo
	Start() error
	Stop() error
}

// BotConnector is implementation of Connector interface.
type BotConnector struct {
	Cfg   *config.Config
	Event *botgolang.Event
}

// ChatMembers returns list of chat members.
func (bc *BotConnector) ChatMembers() ([]botgolang.ChatMember, error) {
	return bc.Cfg.Bot.GetChatMembers(bc.Event.Payload.Chat.ID)
}

// SendMessage sends message to chat.
func (bc *BotConnector) SendMessage(msg string) error {
	message := bc.Cfg.Bot.NewTextMessage(bc.Event.Payload.Chat.ID, msg)
	return bc.Cfg.Bot.SendMessage(message)
}

// SendURLMessage sends message to chat with URL link.
func (bc *BotConnector) SendURLMessage(msg, txt, url string) error {
	message := bc.Cfg.Bot.NewTextMessage(bc.Event.Payload.Chat.ID, msg)
	btn := botgolang.NewURLButton(txt, url)
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(btn)
	message.AttachInlineKeyboard(keyboard)
	return bc.Cfg.Bot.SendMessage(message)
}

// Version returns bot version.
func (bc *BotConnector) Version() *config.BuildInfo {
	return bc.Cfg.BuildInfo
}

// Start starts bot.
func (bc *BotConnector) Start() error {
	err := bc.Cfg.StartBot(bc.Event)
	if err != nil {
		return fmt.Errorf("can't start bot: %v", err)
	}
	return bc.SendMessage("success")
}

// Stop stops bot.
func (bc *BotConnector) Stop() error {
	err := bc.Cfg.StopBot(bc.Event)
	if err != nil {
		return fmt.Errorf("can't stop bot: %v", err)
	}
	return bc.SendMessage("success")
}

// Start starts bot.
func Start(c Connector) error {
	return c.Start()
}

// Stop stops bot.
func Stop(c Connector) error {
	return c.Start()
}

// Go returns a list of chat members in random order.
func Go(c Connector) error {
	members, err := c.ChatMembers()
	if err != nil {
		return fmt.Errorf("can't get chat members: %v", err)
	}
	names := make([]string, 0, len(members)-1)
	for _, m := range members {
		if !botIDRegexp.MatchString(m.User.ID) {
			names = append(names, fmt.Sprintf("@[%s]", m.User.ID))
		}
	}
	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})
	return c.SendMessage(strings.Join(names, "\n"))
}

// Version returns bot version.
func Version(c Connector) error {
	v := c.Version()
	msg := fmt.Sprintf("%v %v\n%v, %v, %v UTC", v.Name, v.Hash, v.Revision, v.GoVersion, v.Date)
	if v.URL == "" {
		return c.SendMessage(msg)
	}
	return c.SendURLMessage(msg, "sources", v.URL)
}
