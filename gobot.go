package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
)

const (
	// Name is a program name.
	Name = "goBot"
	// configFile is default configuration file name.
	configFile = "config.json"
)

var (
	// Version is git version
	Version = ""
	// Revision is revision number
	Revision = ""
	// BuildDate is build date
	BuildDate = ""
	// GoVersion is runtime Go language version
	GoVersion = runtime.Version()
	// allowedEvents are bot events for handling
	allowedEvents = map[botgolang.EventType]bool{
		botgolang.NEW_MESSAGE:    true,
		botgolang.EDITED_MESSAGE: true,
	}
	// allowedCommands is commands for handling bot's actions
	allowedCommands = map[string]func(*config.Config, *botgolang.Event) error{
		"/go":      cmdGo,
		"/version": cmdVersion,
	}
	botIDRegexp = regexp.MustCompile(`^\d+$`)
	logError    = log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfo     = log.New(os.Stdout, "INFO  ", log.LstdFlags)
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			logError.Printf("abnormal termination [%v]: %v\n%v", Version, r, string(debug.Stack()))
		}
	}()
	version := flag.Bool("version", false, "show version")
	cfg := flag.String("config", configFile, "configuration file")
	flag.Parse()

	versionInfo := fmt.Sprintf("%v: %v %v %v %v\n", Name, Version, Revision, GoVersion, BuildDate)
	if *version {
		fmt.Println(versionInfo)
		flag.PrintDefaults()
		return
	}
	c, err := config.New(*cfg)
	if err != nil {
		panic(err)
	}
	logInfo.Printf("run %v", versionInfo)
	serve(c)
	logInfo.Printf("stopped %s", Name)
}

// serve runs bot's event loop.
func serve(c *config.Config) {
	var (
		sigint      = make(chan os.Signal, 1)
		ctx, cancel = context.WithCancel(context.Background())
		events      = c.Bot.GetUpdatesChannel(ctx)
	)
	defer func() {
		close(sigint)
		cancel()
	}()
	signal.Notify(sigint, os.Interrupt, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGQUIT))
	for {
		select {
		case s := <-sigint:
			logInfo.Printf("taken signal %v", s)
			return
		case e := <-events:
			logInfo.Printf("[%s] got event type=%v for chat=%v", e.Payload.MsgID, e.Type, e.Payload.Chat.ID)
			start := time.Now()
			if skip, err := handle(c, &e); err != nil {
				logError.Printf("[%s] error handling event: %v", e.Payload.MsgID, err)
			} else {
				logInfo.Printf("[%s] handled event in %v (skip=%v)", e.Payload.MsgID, time.Since(start), skip)
			}
		}
	}
}

// handle is common handler for bot events.
// The second boolean returned is true if the event was skipped.
func handle(c *config.Config, event *botgolang.Event) (bool, error) {
	if !allowedEvents[event.Type] {
		return true, nil
	}
	msg := strings.Trim(event.Payload.Text, " ")
	f, ok := allowedCommands[msg]
	if !ok {
		return true, nil
	}
	logInfo.Printf("[%s] handling command --> %v", event.Payload.MsgID, msg)
	return false, f(c, event)
}

// cmdGo handles "/go" command.
func cmdGo(c *config.Config, event *botgolang.Event) error {
	members, err := c.Bot.GetChatMembers(event.Payload.Chat.ID)
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
	message := c.Bot.NewTextMessage(event.Payload.Chat.ID, strings.Join(names, "\n"))
	return c.Bot.SendMessage(message)
}

// cmdVersion handles "/version" command.
func cmdVersion(c *config.Config, event *botgolang.Event) error {
	txt := fmt.Sprintf("%v %v\n%v, %v, %v UTC", Name, Version, Revision, GoVersion, BuildDate)
	message := c.Bot.NewTextMessage(event.Payload.Chat.ID, txt)
	return c.Bot.SendMessage(message)
}
