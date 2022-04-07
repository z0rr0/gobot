package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"

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

	p, stop := serve.New(2)
	serve.Run(c, p, logInfo, logError)
	<-stop

	if err = c.Close(); err != nil {
		logError.Printf("can't close config: %v", err)
	}
	logInfo.Printf("stopped %s", Name)
}
