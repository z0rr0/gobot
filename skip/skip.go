package skip

import (
	"fmt"
	"log"
	"time"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

func New(c *config.Config, stopService <-chan struct{}, logInfo, logError *log.Logger) <-chan struct{} {
	stopSkip := make(chan struct{})
	go run(c, stopService, stopSkip, logInfo, logError)
	return stopSkip
}

func run(c *config.Config, stopService <-chan struct{}, stopSkip chan<- struct{}, logInfo, logError *log.Logger) {
	var duration = nextDayDuration(time.Now().In(c.Timezone))

	for {
		select {
		case <-stopService:
			logInfo.Println("stop skip cron")
			close(stopSkip)
		case <-time.After(duration):
			logInfo.Printf("skip tick after %v", duration)

			if err := cleanSkip(c); err != nil {
				logError.Printf("failed update: %v", err)
				duration = time.Minute
			} else {
				duration = nextDayDuration(time.Now().In(c.Timezone))
			}
		}
	}
}

func nextDayDuration(ts time.Time) time.Duration {
	return time.Date(ts.Year(), ts.Month(), ts.Day()+1, 0, 0, 1, 0, ts.Location()).Sub(ts)
}

func cleanSkip(c *config.Config) error {
	ctx, cancel := c.Context()
	defer cancel()

	err := db.CleanSkip(ctx, c.DB)
	if err != nil {
		return fmt.Errorf("clean skip: %w", err)
	}

	return nil
}
