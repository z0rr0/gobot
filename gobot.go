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
	"github.com/z0rr0/gobot/db"
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
	allowedCommands = map[string]func(context.Context, cmd.Event) error{
		"/start":   cmd.Start,
		"/stop":    cmd.Stop,
		"/version": cmd.Version,
		"/go":      cmd.Go,
		"/shuffle": cmd.Go, // alias for "/go"
		"/exclude": cmd.Exclude,
	}
	notStoppedCommands = map[string]func(context.Context, cmd.Event) error{
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
	argsStr := strings.SplitN(event.Payload.Text, " ", 2)
	msg := strings.Trim(argsStr[0], " ")
	handler, ok := allowedCommands[msg]
	if !ok {
		return false, nil
	}
	ctx, cancel := c.Context()
	defer cancel()

	chat, err := db.GetOrCreate(ctx, c.Db, event.Payload.Chat.ID)
	if err != nil {
		return false, err
	}
	if !chat.Active {
		handler, ok = notStoppedCommands[msg]
	}
	if !ok {
		return false, nil
	}
	args := ""
	if len(argsStr) > 1 {
		args = argsStr[1] // argsStr length is 1 on 2
	}
	bc := cmd.Event{Cfg: c, ChatEvent: event, Chat: chat, Arguments: args}
	return true, handler(ctx, bc)
}
