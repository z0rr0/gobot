package cmd

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

var (
	// botIDRegexp is a regexp to detect UserID as a bot identifier.
	botIDRegexp = regexp.MustCompile(`^\d+$`)
	// botIDRegexp is a regexp to find all UserIDs in arguments.
	userIDRegexp = regexp.MustCompile(`@\[([0-9A-Za-z@.]+)]`)
)

// Event is implementation of Connector interface.
type Event struct {
	Cfg       *config.Config
	ChatEvent *botgolang.Event
	Chat      *db.Chat
	OnlyChat  bool
	Arguments string
	// only for testing
	debug  bool
	buffer *bytes.Buffer
}

func (e *Event) writeLog(msg string) error {
	if !e.debug {
		return nil
	}
	if e.buffer == nil {
		e.buffer = bytes.NewBufferString(msg)
		return nil
	}
	_, err := e.buffer.WriteString(msg)
	return err
}

// IsChat returns true if event is chat event.
func (e *Event) IsChat() bool {
	return e.ChatEvent.Payload.Chat.ID != e.ChatEvent.Payload.From.ID
}

// Unavailable returns true if event is unavailable.
func (e *Event) Unavailable() bool {
	return e.OnlyChat && !e.IsChat()
}

// SendMessage sends message to chat.
func (e *Event) SendMessage(msg string) error {
	if err := e.writeLog(msg); err != nil {
		return err
	}
	message := e.Cfg.Bot.NewTextMessage(e.Chat.ID, msg)
	return e.Cfg.Bot.SendMessage(message)
}

// SendURLMessage sends message to chat with URL link.
func (e *Event) SendURLMessage(msg, txt, url string) error {
	if err := e.writeLog(msg); err != nil {
		return err
	}
	message := e.Cfg.Bot.NewTextMessage(e.Chat.ID, msg)
	btn := botgolang.NewURLButton(txt, url)
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(btn)
	message.AttachInlineKeyboard(keyboard)
	return e.Cfg.Bot.SendMessage(message)
}

// ArgsUserIDs returns all UserIDs from arguments.
func (e *Event) ArgsUserIDs() map[string]struct{} {
	found := userIDRegexp.FindAllStringSubmatch(e.Arguments, -1)
	n := len(found)
	if n == 0 {
		return nil
	}
	users := make(map[string]struct{}, n)
	for _, userInfo := range found {
		if len(userInfo) != 2 {
			continue
		}
		users[userInfo[1]] = struct{}{}
	}
	return users
}

// Start starts bot.
func Start(ctx context.Context, e *Event) error {
	if e.Chat.Active {
		return e.SendMessage("already started")
	}
	e.Chat.Active = true
	err := e.Chat.Upsert(ctx, e.Cfg.DB)
	if err != nil {
		return err
	}
	return e.SendMessage("started")
}

// Stop stops bot.
func Stop(ctx context.Context, e *Event) error {
	if !e.Chat.Active {
		return e.SendMessage("already stopped")
	}
	e.Chat.Active = false
	err := e.Chat.Upsert(ctx, e.Cfg.DB)
	if err != nil {
		return err
	}
	return e.SendMessage("stopped")
}

// Go returns a list of chat members in random order.
func Go(_ context.Context, e *Event) error {
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
func Version(_ context.Context, e *Event) error {
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
	sort.Strings(exclude)
	return strings.Join(exclude, "\n")
}

// setExclude sets exclude users from chat.
func (e *Event) setExclude(ctx context.Context) error {
	users := e.ArgsUserIDs()
	if len(users) == 0 {
		return e.SendMessage("no user IDs in arguments")
	}
	e.Chat.AddExclude(users)
	if err := e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return fmt.Errorf("can't handle exclude command: %v", err)
	}
	return e.SendMessage("success")
}

// Exclude adds users to ignore list or returns them
func Exclude(ctx context.Context, e *Event) error {
	if e.Arguments == "" {
		excluded := e.getExclude()
		if excluded == "" {
			return e.SendMessage("no excluded users")
		}
		return e.SendMessage(excluded)
	}
	return e.setExclude(ctx)
}

// Include removes users from ignore list or shows all included ones.
func Include(ctx context.Context, e *Event) error {
	if e.Arguments == "" {
		return Go(ctx, e)
	}
	if e.Chat.ExcludeUsers == nil {
		return e.SendMessage("success") // empty exclude list
	}
	users := e.ArgsUserIDs()
	if len(users) == 0 {
		return e.SendMessage("no user IDs in arguments")
	}
	e.Chat.DelExclude(users)
	if err := e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return fmt.Errorf("can't handle include command: %v", err)
	}
	return e.SendMessage("success")
}

//func Link(ctx context.Context, e Event) error {}
