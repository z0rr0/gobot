package cmd

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/url"
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
	authorRegexp = regexp.MustCompile(`^([0-9A-Za-z@.]+)`)
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
	message := e.Cfg.Bt.NewTextMessage(e.Chat.ID, msg)
	return e.Cfg.Bt.SendMessage(message)
}

// SendURLMessage sends message to chat with URL link.
func (e *Event) SendURLMessage(msg, txt, url string) error {
	if err := e.writeLog(msg); err != nil {
		return err
	}

	message := e.Cfg.Bt.NewTextMessage(e.Chat.ID, msg)
	btn := botgolang.NewURLButton(txt, url)

	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(btn)

	message.AttachInlineKeyboard(keyboard)
	return e.Cfg.Bt.SendMessage(message)
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
	members, err := e.Cfg.Bt.GetChatMembers(e.Chat.ID)
	if err != nil {
		return fmt.Errorf("can't get chat members: %v", err)
	}
	names := make([]string, 0, len(members)-1)
	for _, m := range members {
		if _, ok := e.Chat.ExcludeUsers[m.User.ID]; ok {
			continue
		}

		if _, ok := e.Chat.SkipUsers[m.User.ID]; ok {
			continue
		}

		if !botIDRegexp.MatchString(m.User.ID) {
			names = append(names, fmt.Sprintf("@[%s]", m.User.ID))
		}
	}
	if len(names) == 0 {
		return e.SendMessage("no users :(")
	}
	r := rand.New(e.Cfg.RandSource)
	r.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})
	msg := strings.Join(names, "\n")
	if e.Chat.URL != "" {
		return e.SendURLMessage(msg, "📞 "+e.Chat.URLText, e.Chat.URL)
	}
	return e.SendMessage(msg)
}

// Version returns bot version.
func Version(_ context.Context, e *Event) error {
	v := e.Cfg.BuildInfo
	msg := fmt.Sprintf(
		"%v %v\nRevision: %v\nGo version: %v\nBuild time: %v",
		v.Name, v.Hash, v.Revision, v.GoVersion, v.Date,
	)
	if v.URL == "" {
		return e.SendMessage(msg)
	}
	return e.SendURLMessage(msg, v.URL, v.URL)
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

// Link sets a link to call or returns current one.
func Link(ctx context.Context, e *Event) error {
	const maxTextLen = 255
	var textURL = "call"

	if e.Arguments == "" {
		if e.Chat.URL == "" {
			return e.SendMessage("no calling URL for this chat")
		}
		return e.SendMessage(e.Chat.URL)
	}
	params := strings.SplitN(e.Arguments, " ", 2)
	linkURL := params[0]

	u, err := url.Parse(linkURL)
	if err != nil {
		return err
	}
	if !u.IsAbs() {
		return e.SendMessage("incorrect URL")
	}

	if len(params) > 1 {
		textURL = params[1]
	}
	if len(textURL) > maxTextLen {
		return e.SendMessage(fmt.Sprintf("text is too long (max %d characters)", maxTextLen))
	}

	e.Chat.URL = u.String()
	e.Chat.URLText = textURL

	if err = e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return err
	}
	return e.SendMessage("success")
}

// ResetLink removes link from chat.
func ResetLink(ctx context.Context, e *Event) error {
	if e.Chat.URL == "" {
		return e.SendMessage("no calling URL for this chat")
	}
	e.Chat.URL = ""
	e.Chat.URLText = "call" // default value
	if err := e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return err
	}
	return e.SendMessage("success")
}

// Vacation adds or removes users from ignored list.
func Vacation(ctx context.Context, e *Event) error {
	var (
		msg        string
		authorUser = e.ChatEvent.Payload.From.User.ID
	)
	if !authorRegexp.MatchString(authorUser) {
		return e.SendMessage("no valid author user")
	}

	// only one user map, the author
	userMap := map[string]struct{}{authorUser: {}}

	if _, ok := e.Chat.ExcludeUsers[authorUser]; ok {
		// user is already in exclude list, so delete him from it
		e.Chat.DelExclude(userMap)
		msg = "you are back from vacation, welcome"
	} else {
		// user is not in exclude list, so add him to it
		e.Chat.AddExclude(userMap)
		msg = "you are on vacation, good luck"
	}

	if err := e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return fmt.Errorf("can't handle command: %v", err)
	}

	return e.SendMessage(fmt.Sprintf("@[%s] %s", authorUser, msg))
}

// GPT generates text using ChatGPT.
func GPT(ctx context.Context, e *Event) error {
	if e.Cfg.G.Client == nil {
		return e.SendMessage("gpt is not configured")
	}

	if !e.Chat.GPT {
		return e.SendMessage("gpt is not allowed for this chat")
	}

	content := strings.TrimSpace(e.Arguments)
	if content == "" {
		return e.SendMessage("no arguments")
	}

	result, err := e.Cfg.G.Response(ctx, content)
	if err != nil {
		return err
	}

	return e.SendMessage(result)
}

// YandexGPT generates text using Yandex GPT.
func YandexGPT(ctx context.Context, e *Event) error {
	if e.Cfg.Y.Client == nil {
		return e.SendMessage("yandex gpt is not configured")
	}

	if !e.Chat.GPT {
		return e.SendMessage("gpt is not allowed for this chat")
	}

	content := strings.TrimSpace(e.Arguments)
	if content == "" {
		return e.SendMessage("no arguments")
	}

	result, err := e.Cfg.Y.Response(ctx, content)
	if err != nil {
		return err
	}

	return e.SendMessage(result)
}

// Skip adds or removes users from skipped list.
func Skip(ctx context.Context, e *Event) error {
	var (
		msg        string
		authorUser = e.ChatEvent.Payload.From.User.ID
	)
	if !authorRegexp.MatchString(authorUser) {
		return e.SendMessage("no valid author user")
	}

	if _, ok := e.Chat.SkipUsers[authorUser]; ok {
		e.Chat.DelSkip(authorUser)
		msg = "ok, you are in the list again"
	} else {
		e.Chat.AddSkip(authorUser)
		msg = "ok, you will be skipped today"
	}

	if err := e.Chat.Update(ctx, e.Cfg.DB); err != nil {
		return fmt.Errorf("can't handle command: %v", err)
	}

	return e.SendMessage(fmt.Sprintf("@[%s] %s", authorUser, msg))
}
