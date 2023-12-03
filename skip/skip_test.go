package skip

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/z0rr0/gobot/config"
)

const (
	// configPath is the path of temporary configuration file.
	configPath = "/tmp/gobot_config_test.toml"
)

var (
	buildInfo = &config.BuildInfo{
		Name:      "cmd_test",
		Hash:      "123",
		Revision:  "v0.0.1",
		GoVersion: "go1.18",
		Date:      "2022-03-28_06:21:50 UTC",
		URL:       "https://github.com/z0rr0/gobot",
	}
	//defaultCtx = context.Background()
	testLogger = log.New(os.Stdout, "TEST  ", log.LstdFlags)
)

func TestNew(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		_, err := fmt.Fprint(w, response)
		if err != nil {
			t.Error(err)
		}
	})
	s := httptest.NewServer(handler)
	defer s.Close()

	c, err := config.New(configPath, buildInfo, s)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}

	defer func() {
		if errCfg := c.Close(); errCfg != nil {
			t.Error(errCfg)
		}
	}()

	stopService := make(chan struct{})
	h := New(c, stopService, testLogger, testLogger)

	h.forceClean <- struct{}{}

	close(stopService)
	<-h.StopSkip
}

func TestNextTimeout(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	testCases := []struct {
		name string
		ts   time.Time
		want time.Duration
	}{
		{
			name: "UTC",
			ts:   time.Date(2023, 11, 23, 5, 4, 3, 0, time.UTC),
			want: time.Hour*18 + time.Minute*55 + time.Second*58,
		},
		{
			name: "day",
			ts:   time.Date(2023, 11, 23, 0, 0, 0, 0, time.FixedZone("UTC+1", 3600)),
			want: time.Hour*24 + time.Second,
		},
		{
			name: "winter",
			ts:   time.Date(2023, 11, 5, 0, 0, 1, 0, newYork),
			want: time.Hour * 25,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			if got := nextTimeout(tc.ts); got != tc.want {
				t.Errorf("failed compare durations, got %v want %v", got, tc.want)
			}
		})
	}
}
