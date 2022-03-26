package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	botgolang "github.com/mail-ru-im/bot-golang"
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
	// allowedBotEvents are bot events for handling
	allowedBotEvents = map[botgolang.EventType]bool{
		botgolang.NEW_MESSAGE:    true,
		botgolang.EDITED_MESSAGE: true,
	}
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "abnormal termination [%v]: %v\n%v", Version, r, string(debug.Stack()))
		}
	}()
	version := flag.Bool("version", false, "show version")
	cfg := flag.String("config", configFile, "configuration file")
	flag.Parse()

	if *version {
		fmt.Printf("%v: %v %v %v %v\n", Name, Version, Revision, GoVersion, BuildDate)
		flag.PrintDefaults()
		return
	}
	bi, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Printf("build=%#v\n", bi)
	}

	fmt.Println(*cfg)
}
