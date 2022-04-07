package serve

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/cmd"
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
	nullLogger = log.New(os.Stdout, "TEST  ", log.LstdFlags)
)

func patchHandlers(name string) *[]string {
	var (
		mu sync.Mutex
		b  = make([]string, 0)
	)
	f := func(ctx context.Context, event *cmd.Event) error {
		time.Sleep(time.Millisecond * 100) // simulate  command handling
		mu.Lock()
		b = append(b, event.Arguments)
		mu.Unlock()
		return nil
	}
	allowedCommands[name] = f
	notStoppedCommands[name] = f
	return &b
}

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
	arguments := []string{"a", "b", "c", "d"}
	b := patchHandlers("TestNew")
	p, stop := New(2)
	for _, a := range arguments {
		event := Payload{
			Cfg: c,
			Event: &botgolang.Event{
				Type: botgolang.NEW_MESSAGE,
				Payload: botgolang.EventPayload{
					BaseEventPayload: botgolang.BaseEventPayload{
						Text:  "TestNew " + a,
						MsgID: a,
						Chat:  botgolang.Chat{ID: a},
					},
				},
			},
			LogInfo:  nullLogger,
			LogError: nullLogger,
		}
		p <- event
	}
	// failed event type
	p <- Payload{
		Cfg: c,
		Event: &botgolang.Event{
			Type: botgolang.DELETED_MESSAGE,
			Payload: botgolang.EventPayload{
				BaseEventPayload: botgolang.BaseEventPayload{
					Text:  "TestNew x",
					MsgID: "delete",
					Chat:  botgolang.Chat{ID: "delete"},
				},
			},
		},
		LogInfo:  nullLogger,
		LogError: nullLogger,
	}
	// unknown command
	p <- Payload{
		Cfg: c,
		Event: &botgolang.Event{
			Type: botgolang.NEW_MESSAGE,
			Payload: botgolang.EventPayload{
				BaseEventPayload: botgolang.BaseEventPayload{
					Text:  "BadCmd y",
					MsgID: "BadCmd",
					Chat:  botgolang.Chat{ID: "BadCmd"},
				},
			},
		},
		LogInfo:  nullLogger,
		LogError: nullLogger,
	}
	// all done, stop
	close(p)
	<-stop

	sort.Strings(*b)
	if result := strings.Join(*b, ";"); result != "a;b;c;d" {
		t.Errorf("unexpected result: %s", result)
	}
}
