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

// Event is implementation of Connector interface.
type Event struct {
	Cfg       *config.Config
	ChatEvent *botgolang.Event
	Chat      *db.Chat
	Arguments string
}

// IsChat returns true if event is chat event.
func (e *Event) IsChat() bool {
	return e.ChatEvent.Payload.Chat.ID != e.ChatEvent.Payload.From.ID
}

// SendMessage sends message to chat.
func (e *Event) SendMessage(msg string) error {
	message := e.Cfg.Bot.NewTextMessage(e.ChatEvent.Payload.Chat.ID, msg)
	return e.Cfg.Bot.SendMessage(message)
}

// SendURLMessage sends message to chat with URL link.
func (e *Event) SendURLMessage(msg, txt, url string) error {
	message := e.Cfg.Bot.NewTextMessage(e.ChatEvent.Payload.Chat.ID, msg)
	btn := botgolang.NewURLButton(txt, url)
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(btn)
	message.AttachInlineKeyboard(keyboard)
	return e.Cfg.Bot.SendMessage(message)
}

// Start starts bot.
func Start(ctx context.Context, e Event) error {
	if e.Chat.Active {
		return e.SendMessage("already started")
	}
	err := db.UpsertActive(ctx, e.Cfg.Db, e.ChatEvent.Payload.Chat.ID, true)
	if err != nil {
		return err
	}
	return e.SendMessage("started")
}

// Stop stops bot.
func Stop(ctx context.Context, e Event) error {
	if !e.Chat.Active {
		return e.SendMessage("already stopped")
	}
	err := db.UpsertActive(ctx, e.Cfg.Db, e.ChatEvent.Payload.Chat.ID, false)
	if err != nil {
		return err
	}
	return e.SendMessage("stopped")
}

// Go returns a list of chat members in random order.
func Go(_ context.Context, e Event) error {
	if !e.IsChat() {
		return e.SendMessage("sorry, this command is available only for chats")
	}
	members, err := e.Cfg.Bot.GetChatMembers(e.Chat.ID)
	if err != nil {
		return fmt.Errorf("can't get chat members: %v", err)
	}
	names := make([]string, 0, len(members)-1)
	for _, m := range members {
		if _, ok := e.Chat.ExcludeUsers[m.User.ID]; ok {
			continue
		}
		if !botIDRegexp.MatchString(m.User.ID) {
			names = append(names, fmt.Sprintf("@[%s]", m.User.ID))
		}
	}
	if len(names) == 0 {
		return e.SendMessage("no users :(")
	}
	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})
	msg := strings.Join(names, "\n")
	if e.Chat.URL != "" {
		return e.SendURLMessage(msg, "📞 call", e.Chat.URL)
	}
	return e.SendMessage(msg)
}

// Version returns bot version.
func Version(_ context.Context, e Event) error {
	v := e.Cfg.BuildInfo
	msg := fmt.Sprintf("%v %v\n%v, %v, %v UTC", v.Name, v.Hash, v.Revision, v.GoVersion, v.Date)
	if v.URL == "" {
		return e.SendMessage(msg)
	}
	return e.SendURLMessage(msg, "sources", v.URL)
}

// getExclude returns exclude users from chat.
func (e *Event) getExclude() string {
	if e.Chat.ExcludeUsers == nil {
		return ""
	}
	exclude := make([]string, 0, len(e.Chat.ExcludeUsers))
	for userID := range e.Chat.ExcludeUsers {
		exclude = append(exclude, fmt.Sprintf("@[%s]", userID))
	}
	return strings.Join(exclude, "\n")
}

// setExclude sets exclude users from chat.
func (e *Event) setExclude(ctx context.Context) error {
	found := userIDRegexp.FindAllStringSubmatch(e.Arguments, -1)
	n := len(found)
	if n < 1 {
		return e.SendMessage("no users found in arguments")
	}
	users := make(map[string]struct{}, n)
	for _, userInfo := range found {
		if len(userInfo) != 2 {
			continue
		}
		users[userInfo[1]] = struct{}{}
	}
	e.Chat.AddExclude(users)
	if err := e.Chat.Update(ctx, e.Cfg.Db); err != nil {
		return fmt.Errorf("can't handle exclude command: %v", err)
	}
	return e.SendMessage("success")
}

// Exclude adds users to ignore list or returns them
func Exclude(ctx context.Context, e Event) error {
	if !e.IsChat() {
		return e.SendMessage("sorry, this command is available only for chats")
	}
	if e.Arguments == "" {
		if excluded := e.getExclude(); excluded == "" {
			return e.SendMessage("no excluded users")
		} else {
			return e.SendMessage(excluded)
		}
	}
	return e.setExclude(ctx)
}

//func SetLink(c Connector) error {}
//func UnIgnore(c Connector) error {}
