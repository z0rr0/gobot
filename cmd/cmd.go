package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

var (
	// botIDRegexp is a regexp detect UserID as a bot identifier.
	botIDRegexp  = regexp.MustCompile(`^\d+$`)
	userIDRegexp = regexp.MustCompile(`@\[([A-Za-z@.]+)]`)
)

// Connector is interface for bot actions.
type Connector interface {
	IsChat() bool
	IsActive() bool
	ChatMembers() ([]botgolang.ChatMember, error)
	IgnoredMembers() map[string]struct{}
	SendMessage(msg string) error
	SendURLMessage(msg, txt, url string) error
	GoURL() string
	Version() *config.BuildInfo
	Args() string
	Ignore(map[string]struct{}) error
	Start() error
	Stop() error
}

// BotConnector is implementation of Connector interface.
type BotConnector struct {
	Cfg       *config.Config
	Event     *botgolang.Event
	Chat      *db.Chat
	Arguments string
}

// IsChat returns true if event is chat event.
func (bc *BotConnector) IsChat() bool {
	return bc.Event.Payload.Chat.ID != bc.Event.Payload.From.ID
}

// IsActive returns true if event is active event.
func (bc *BotConnector) IsActive() bool {
	return bc.Chat != nil && bc.Chat.Active
}

// ChatMembers returns list of chat members.
func (bc *BotConnector) ChatMembers() ([]botgolang.ChatMember, error) {
	return bc.Cfg.Bot.GetChatMembers(bc.Chat.ID)
}

// IgnoredMembers returns list of ignored users for the chat.
func (bc *BotConnector) IgnoredMembers() map[string]struct{} {
	return bc.Chat.ExcludeUsers
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
	if !bc.IsActive() {
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

// Args returns arguments for command.
func (bc *BotConnector) Args() string {
	return bc.Arguments
}

// Ignore handles "ignore" command.
func (bc *BotConnector) Ignore(users map[string]struct{}) error {
	bc.Chat.AddExclude(users)
	return bc.Cfg.SaveChat(bc.Chat)
}

// Start starts bot.
func Start(_ context.Context, c Connector) error {
	return c.Start()
}

// Stop stops bot.
func Stop(_ context.Context, c Connector) error {
	return c.Stop()
}

// Go returns a list of chat members in random order.
func Go(_ context.Context, c Connector) error {
	if !c.IsChat() {
		return c.SendMessage("sorry, this command is available only for chat")
	}
	members, err := c.ChatMembers()
	if err != nil {
		return fmt.Errorf("can't get chat members: %v", err)
	}
	ignored := c.IgnoredMembers()
	names := make([]string, 0, len(members)-1)
	for _, m := range members {
		if _, ok := ignored[m.User.ID]; ok {
			continue
		}
		if !botIDRegexp.MatchString(m.User.ID) {
			names = append(names, fmt.Sprintf("@[%s]", m.User.ID))
		}
	}
	if len(names) == 0 {
		return c.SendMessage("no users :(")
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
func Version(_ context.Context, c Connector) error {
	v := c.Version()
	msg := fmt.Sprintf("%v %v\n%v, %v, %v UTC", v.Name, v.Hash, v.Revision, v.GoVersion, v.Date)
	if v.URL == "" {
		return c.SendMessage(msg)
	}
	return c.SendURLMessage(msg, "sources", v.URL)
}

// Exclude adds users to ignore list.
func Exclude(_ context.Context, c Connector) error {
	if !c.IsActive() {
		return c.SendMessage("sorry, this command is available only for active chats")
	}
	found := userIDRegexp.FindAllStringSubmatch(c.Args(), -1)
	users := make(map[string]struct{}, len(found))
	for _, userInfo := range found {
		if len(userInfo) != 2 {
			continue
		}
		users[userInfo[1]] = struct{}{}
	}
	err := c.Ignore(users)
	if err != nil {
		return fmt.Errorf("can't handle exclude command: %v", err)
	}
	return c.SendMessage("success")
}

//func SetLink(c Connector) error {}
//func UnIgnore(c Connector) error {}
