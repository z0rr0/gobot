package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/cmd"
	"github.com/z0rr0/gobot/config"
)

const (
	// Name is a program name.
	Name = "goBot"
	// configFile is default configuration file name.
	configFile = "config.toml"
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
	allowedCommands = map[string]func(c cmd.Connector) error{
		"/go":      cmd.Go,
		"/shuffle": cmd.Go, // alias for "/go"
		"/version": cmd.Version,
		"/stop":    cmd.Stop,
		"/start":   cmd.Start,
		"/exclude": cmd.Exclude,
	}
	alwaysCommands = map[string]func(c cmd.Connector) error{
		"/start": cmd.Start,
	}
	logError = log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfo  = log.New(os.Stdout, "INFO  ", log.LstdFlags)
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
	buildInfo := &config.BuildInfo{Name: Name, Hash: Version, Revision: Revision, GoVersion: GoVersion, Date: BuildDate}
	c, err := config.New(*cfg, buildInfo, nil)
	if err != nil {
		panic(err)
	}
	logInfo.Printf("run %v", versionInfo)
	serve(c)
	if err = c.Close(); err != nil {
		logError.Printf("can't close config: %v", err)
	}
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
			if handled, err := handle(c, &e); err != nil {
				logError.Printf("[%s] error handling event: %v", e.Payload.MsgID, err)
			} else {
				logInfo.Printf("[%s] handled event in %v (known=%v)", e.Payload.MsgID, time.Since(start), handled)
			}
		}
	}
}

// handle is common handler for bot events.
// The first boolean returned is true if the event was handled.
func handle(c *config.Config, event *botgolang.Event) (bool, error) {
	if !allowedEvents[event.Type] {
		return false, nil
	}
	txt := strings.SplitN(event.Payload.Text, " ", 2)
	msg := strings.Trim(txt[0], " ")
	f, ok := allowedCommands[msg]
	if !ok {
		return false, nil
	}
	chat, err := c.Chat(event)
	if err != nil {
		return false, err
	}
	if !chat.Active {
		if f, ok = alwaysCommands[msg]; ok {
			logInfo.Printf("[%s] handling not ignored command --> %v", event.Payload.MsgID, msg)
			return true, f(&cmd.BotConnector{Cfg: c, Event: event, Arguments: txt[1:]})
		}
		return false, nil
	}
	logInfo.Printf("[%s] handling command --> %v", event.Payload.MsgID, msg)
	return true, f(&cmd.BotConnector{Cfg: c, Event: event, Chat: chat, Arguments: txt[1:]})
}
