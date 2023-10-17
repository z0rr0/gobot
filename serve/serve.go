package serve

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/cmd"
	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

var (
	// allowedEvents are bot events for handling
	allowedEvents = map[botgolang.EventType]bool{
		botgolang.NEW_MESSAGE:    true,
		botgolang.EDITED_MESSAGE: true,
	}
	// allowedCommands is commands for handling bots actions
	allowedCommands = map[string]func(context.Context, *cmd.Event) error{
		"/start":    cmd.Start,
		"/stop":     cmd.Stop,
		"/version":  cmd.Version,
		"/go":       cmd.Go,
		"/shuffle":  cmd.Go, // alias for "/go"
		"/exclude":  cmd.Exclude,
		"/include":  cmd.Include,
		"/link":     cmd.Link,
		"/reset":    cmd.ResetLink,
		"/vacation": cmd.Vacation,
		"/gpt":      cmd.GPT,
		"/ygpt":     cmd.YandexGPT,
	}
	// notSupportedCommands is commands which can't be stopped
	notStoppedCommands = map[string]bool{"/start": true}
	// onlyChatCommands is commands which can be used only for chats
	onlyChatCommands = map[string]bool{
		"/go":       true,
		"/shuffle":  true,
		"/exclude":  true,
		"/include":  true,
		"/link":     true,
		"/reset":    true,
		"/vacation": true,
		"/gpt":      true,
	}
)

// Payload is a struct for events payload.
type Payload struct {
	Cfg      *config.Config
	Event    *botgolang.Event
	LogInfo  *log.Logger
	LogError *log.Logger
}

// ID returns message ID.
func (p *Payload) ID() string {
	return p.Event.Payload.MsgID
}

// handle is common handler for bot events.
// The first boolean returned is true if the event was handled.
func handle(p Payload) (bool, error) {
	if !allowedEvents[p.Event.Type] {
		return false, nil
	}
	argsStr := strings.SplitN(p.Event.Payload.Text, " ", 2)
	cmdName := strings.Trim(argsStr[0], " ")
	handler, ok := allowedCommands[cmdName]
	if !ok {
		return false, nil
	}
	ctx, cancel := p.Cfg.Context()
	defer cancel()

	chat, err := db.GetOrCreate(ctx, p.Cfg.DB, p.Event.Payload.Chat.ID)
	if err != nil {
		return false, err
	}
	if !chat.Active && !notStoppedCommands[cmdName] {
		return false, nil
	}

	args := ""
	if len(argsStr) > 1 {
		args = argsStr[1] // argsStr length is 1 on 2
	}

	e := &cmd.Event{
		Cfg:       p.Cfg,
		ChatEvent: p.Event,
		Chat:      chat,
		Arguments: args,
		OnlyChat:  onlyChatCommands[cmdName],
	}
	p.LogInfo.Printf("[%s] %q handling command --> %v", p.ID(), chat.ID, cmdName)

	if e.Unavailable() {
		return false, e.SendMessage("sorry, this command is available only for chats")
	}

	if err = handler(ctx, e); err != nil {
		p.LogError.Printf("[%s] %q error handling command: %v", p.ID(), chat.ID, err)
		return false, e.SendMessage("sorry, some error occurred")
	}

	return true, nil
}

// worker is a worker function for events handling.
// It listens for the queue channel and handles incoming items.
func worker(wg *sync.WaitGroup, queue <-chan Payload) {
	var (
		msgID    string
		start    time.Time
		duration time.Duration
	)
	defer wg.Done()

	for p := range queue {
		msgID, start = p.ID(), time.Now()

		if handled, err := handle(p); err != nil {
			p.LogError.Printf("[%s] error handling event: %v", msgID, err)
		} else {
			duration = time.Since(start).Truncate(10 * time.Millisecond)
			p.LogInfo.Printf("[%s] handled event in %v (handled=%v)", msgID, duration, handled)
		}
	}
}

// New creates new channels for events queue and stopping any handling.
// A caller must close queue channel and waits stop one closing.
func New(n int) (chan<- Payload, <-chan struct{}) {
	var (
		wg    sync.WaitGroup
		stop  = make(chan struct{})
		queue = make(chan Payload)
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go worker(&wg, queue)
	}

	go func() {
		wg.Wait()
		close(stop)
	}()

	return queue, stop
}

// Run starts main service process.
func Run(c *config.Config, p chan<- Payload, sigint <-chan os.Signal, logInfo, logError *log.Logger) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		events      = c.Bt.GetUpdatesChannel(ctx)
	)
	defer func() {
		close(p)
		cancel()
	}()
	for {
		select {
		case s := <-sigint:
			logInfo.Printf("taken signal %v", s)
			return
		case e := <-events:
			logInfo.Printf("[%s] got event type=%v for chat=%v", e.Payload.MsgID, e.Type, e.Payload.Chat.ID)
			p <- Payload{Cfg: c, Event: &e, LogInfo: logInfo, LogError: logError}
		}
	}
}
