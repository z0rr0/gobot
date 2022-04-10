package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/serve"
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
	logError  = log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfo   = log.New(os.Stdout, "INFO  ", log.LstdFlags)
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

	versionInfo := fmt.Sprintf("%v: %v %v %v %v", Name, Version, Revision, GoVersion, BuildDate)
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
	if c.L.Output != nil {
		// custom logging in a file
		logInfo.SetOutput(c.L.Output)
		logError.SetOutput(c.L.Output)
	}
	logInfo.Printf("start process\n%v\nPID file: %s\nLOG file: %s", versionInfo, c.L.PidFile, c.L.LogFile)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGQUIT))
	defer close(sigint)

	p, stop := serve.New(c.M.Workers)
	serve.Run(c, p, sigint, logInfo, logError)
	<-stop
	logInfo.Printf("stopped %s", Name)
	if err = c.Close(); err != nil {
		log.Fatalf("can't close config: %v", err)
	}
}
