package skip

import (
	"log"
	"time"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
)

// Handler is a skip handler.
type Handler struct {
	Stop  chan struct{}
	clean chan struct{}
}

// New creates a new skip handler and starts it.
func New(c *config.Config, stopService <-chan struct{}, logInfo, logError *log.Logger) *Handler {
	handler := &Handler{
		Stop:  make(chan struct{}),
		clean: make(chan struct{}),
	}

	go handler.start(c, stopService, logInfo, logError)
	return handler
}

// stop stops skip daemon. It closes flag channels as a signal for stop for external services.
func (h *Handler) stop() {
	close(h.clean)
	close(h.Stop)
}

// start runs skip daemon.
func (h *Handler) start(c *config.Config, stopService <-chan struct{}, logInfo, logError *log.Logger) {
	var (
		now     = time.Now().In(c.Timezone)
		timeout = nextTimeout(now)
	)
	logInfo.Printf("start skip-daemon [%v] now=%v, timeout=%v", c.Timezone, now.Truncate(time.Second), timeout)

	defer func() {
		h.stop()
		logInfo.Println("stop skip-daemon")
	}()

	timer := time.NewTimer(timeout)
	defer func() {
		if !timer.Stop() {
			// drain timer channel, if it has already expired or been stopped
			<-timer.C
		}
	}()

	for {
		select {
		case <-stopService:
			return
		case <-timer.C:
			logInfo.Printf("tick after %v", timeout)
			timer.Reset(clean(c, logError))
		case <-h.clean:
			if !timer.Stop() {
				<-timer.C
			}

			timeout = clean(c, logError)
			timer.Reset(timeout)

			logInfo.Printf("force clean skip, new timeout=%v", timeout)
		}
	}
}

// nextTimeout returns next timeout for clean.
func nextTimeout(ts time.Time) time.Duration {
	return time.Date(ts.Year(), ts.Month(), ts.Day()+1, 0, 0, 1, 0, ts.Location()).Sub(ts)
}

// clean removes all skip users from chats.
func clean(c *config.Config, logError *log.Logger) time.Duration {
	ctx, cancel := c.Context()
	defer cancel()

	err := db.CleanSkip(ctx, c.DB)
	if err != nil {
		logError.Printf("failed clean skip: %v", err)
		return 5 * time.Minute // retry again in 5 minutes
	}

	return nextTimeout(time.Now().In(c.Timezone))
}
