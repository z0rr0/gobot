package serve

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
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
	testLogger = log.New(os.Stdout, "TEST  ", log.LstdFlags)
	cmdMutex   sync.Mutex
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
	cmdMutex.Lock()
	allowedCommands[name] = f
	notStoppedCommands[name] = true
	cmdMutex.Unlock()
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
	b := patchHandlers("TestNew")
	p, stop := New(2)
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
		LogInfo:  testLogger,
		LogError: testLogger,
	}
	// valid commands
	arguments := []string{"a", "b", "c", "d"}
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
			LogInfo:  testLogger,
			LogError: testLogger,
		}
		p <- event
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
		LogInfo:  testLogger,
		LogError: testLogger,
	}
	// all done, stop
	close(p)
	<-stop

	sort.Strings(*b)
	expected := strings.Join(arguments, ";")
	if result := strings.Join(*b, ";"); result != expected {
		t.Errorf("failed result=%s, expected=%s", result, expected)
	}
}

func TestRun(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var url = strings.TrimRight(r.URL.Path, " /")
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		if url == "/events/get" {
			response = "{\"events\": [{\"eventId\": 534, \"payload\": " +
				"{\"chat\": {\"chatId\": \"123@chat.agent\", \"title\": \"goBotTest\", \"type\": \"group\"}, " +
				"\"from\": {\"firstName\": \"firstName\", \"lastName\": \"lastName\", " +
				"\"userId\": \"user1@my.team\"}, " +
				"\"msgId\": \"7083840679717109811\", " +
				"\"text\": \"TestRun user-test-msg\", \"timestamp\": 1649335185}, \"type\": \"newMessage\"}], " +
				"\"ok\": true}"
		}
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
	b := patchHandlers("TestRun")
	p, stop := New(2)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	go func() {
		time.Sleep(200 * time.Millisecond) // take time to work
		close(sigint)
	}()

	go Run(c, p, sigint, testLogger, testLogger)
	// all done, stop
	<-stop

	result := strings.Join(*b, " ")
	if !strings.Contains(result, "user-test-msg") {
		t.Errorf("no expected value in the result: %s", result)
	}
}
