package skip

import (
	"log"
	"time"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

// Handler is a skip handler.
type Handler struct {
	StopSkip   chan struct{}
	forceClean chan struct{}
}

func New(c *config.Config, stopService <-chan struct{}, logInfo, logError *log.Logger) *Handler {
	handler := &Handler{
		StopSkip:   make(chan struct{}),
		forceClean: make(chan struct{}),
	}

	go handler.run(c, stopService, logInfo, logError)
	return handler
}

func (h *Handler) close() {
	close(h.forceClean)
	close(h.StopSkip)
}

func (h *Handler) run(c *config.Config, stopService <-chan struct{}, logInfo, logError *log.Logger) {
	var (
		now      = time.Now().In(c.Timezone)
		duration = nextTimeout(now)
	)

	defer h.close()
	logInfo.Printf("skip-daemon [%v] now=%v, duration=%v", c.Timezone, now.Truncate(time.Second), duration)

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-stopService:
			logInfo.Println("stop skip-cron")
			return
		case <-timer.C:
			logInfo.Printf("tick after %v", duration)
			timer.Reset(cleanSkip(c, logError))
		case <-h.forceClean:
			if !timer.Stop() {
				<-timer.C
			}

			duration = cleanSkip(c, logError)
			timer.Reset(duration)

			logInfo.Printf("force clean skip, new duration=%v", duration)
		}
	}
}

func nextTimeout(ts time.Time) time.Duration {
	return time.Date(ts.Year(), ts.Month(), ts.Day()+1, 0, 0, 1, 0, ts.Location()).Sub(ts)
}

func cleanSkip(c *config.Config, logError *log.Logger) time.Duration {
	ctx, cancel := c.Context()
	defer cancel()

	err := db.CleanSkip(ctx, c.DB)
	if err != nil {
		logError.Printf("failed clean skip: %v", err)
		return time.Minute // try again in a minute
	}

	return nextTimeout(time.Now().In(c.Timezone))
}
