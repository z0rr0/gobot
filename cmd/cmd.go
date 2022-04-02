package cmd

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

var (
	botIDRegexp = regexp.MustCompile(`^\d+$`)
)

// Connector is interface for bot actions.
type Connector interface {
	IsChat() bool
	ChatMembers() ([]botgolang.ChatMember, error)
	SendMessage(msg string) error
	SendURLMessage(msg, txt, url string) error
	GoURL() string
	Version() *config.BuildInfo
	Start() error
	Stop() error
}

// BotConnector is implementation of Connector interface.
type BotConnector struct {
	Cfg   *config.Config
	Event *botgolang.Event
	Chat  *db.Chat
}

// IsChat returns true if event is chat event.
func (bc *BotConnector) IsChat() bool {
	return bc.Event.Payload.Chat.ID != bc.Event.Payload.From.ID
}

// ChatMembers returns list of chat members.
func (bc *BotConnector) ChatMembers() ([]botgolang.ChatMember, error) {
	return bc.Cfg.Bot.GetChatMembers(bc.Chat.ID)
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
	if bc.Chat != nil && bc.Chat.Active {
		return bc.SendMessage("already started")
	}
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

// GoURL returns URL for go command.
func (bc *BotConnector) GoURL() string {
	return bc.Chat.URL
}

// Start starts bot.
func Start(c Connector) error {
	return c.Start()
}

// Stop stops bot.
func Stop(c Connector) error {
	return c.Stop()
}

// Go returns a list of chat members in random order.
func Go(c Connector) error {
	if !c.IsChat() {
		return c.SendMessage("sorry, this command is available only for chat")
	}
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
	msg := strings.Join(names, "\n")
	if url := c.GoURL(); url != "" {
		return c.SendURLMessage(msg, "📞 call", url)
	}
	return c.SendMessage(msg)
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
