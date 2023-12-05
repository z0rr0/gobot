package cmd

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	botgolang "github.com/mail-ru-im/bot-golang"

	"github.com/z0rr0/gobot/config"
	"github.com/z0rr0/gobot/db"
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
	defaultCtx = context.Background()
)

func TestStart(t *testing.T) {
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
	chat := &db.Chat{ID: "TestStart"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Start(defaultCtx, e); err != nil {
		t.Errorf("Start: %v", err)
	}
	if !chat.Saved {
		t.Error("chat.Saved = false")
	}
	if !chat.Active {
		t.Error("chat.Active = false")
	}
	expected := "started"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	if err = Start(defaultCtx, e); err != nil {
		t.Errorf("Start: %v", err)
	}
	expected = "already started"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
}

func TestStop(t *testing.T) {
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
	chat := &db.Chat{ID: "TestStop"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	// stop for not saved chat
	if err = Stop(defaultCtx, e); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if chat.Saved {
		t.Error("chat.Saved = true")
	}
	if chat.Active {
		t.Error("chat.Active = true")
	}
	expected := "already stopped"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// create active db record and stop it
	chat.Active = true
	if err = e.Chat.Update(defaultCtx, c.DB); err != nil {
		t.Errorf("e.Chat.Update: %v", err)
	}
	if err = Stop(defaultCtx, e); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if chat.Active {
		t.Error("chat.Active = true")
	}
	expected = "stopped"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
}

func TestVersion(t *testing.T) {
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
	chat := &db.Chat{ID: "TestVersion"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Version(defaultCtx, e); err != nil {
		t.Errorf("Version: %v", err)
	}
	expected := fmt.Sprintf(
		"%v %v\nRevision: %v\nGo version: %v\nBuild time: %v",
		buildInfo.Name, buildInfo.Hash, buildInfo.Revision, buildInfo.GoVersion, buildInfo.Date,
	)
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// build info without URL
	e.Cfg.BuildInfo = &config.BuildInfo{
		Name:      buildInfo.Name,
		Hash:      buildInfo.Hash,
		Revision:  buildInfo.Revision,
		GoVersion: buildInfo.GoVersion,
		Date:      buildInfo.Date,
	}
	if err = Version(defaultCtx, e); err != nil {
		t.Errorf("Version: %v", err)
	}
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
}

func TestGPT(t *testing.T) {
	botServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		_, err := fmt.Fprint(w, response)
		if err != nil {
			t.Error(err)
		}
	}))
	defer botServer.Close()

	gptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"id":"test","object":"chat.completion","created":1677652288,` +
			`"choices":[{"index":0,"message":{"content":"Hi, it is ChatGPT!"},` +
			`"finish_reason":"stop"}],"usage":{"prompt_tokens":35,"completion_tokens":13,"total_tokens":48}}`

		if _, err := fmt.Fprint(w, response); err != nil {
			panic(err)
		}
	}))
	defer gptServer.Close()

	c, err := config.New(configPath, buildInfo, botServer)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	defer func() {
		if errCfg := c.Close(); errCfg != nil {
			t.Error(errCfg)
		}
	}()
	c.G.Bearer = "test"
	c.G.URL = gptServer.URL
	c.G.Client = gptServer.Client()

	chat := &db.Chat{ID: "TestGPT", GPT: true}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, Arguments: "request", debug: true}
	if err = GPT(defaultCtx, e); err != nil {
		t.Errorf("GPT: %v", err)
	}

	if msg := e.buffer.String(); msg != "Hi, it is ChatGPT!" {
		t.Errorf("failed bot response=%q", msg)
	}
}

func TestYandexGPT(t *testing.T) {
	botServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		_, err := fmt.Fprint(w, response)
		if err != nil {
			t.Error(err)
		}
	}))
	defer botServer.Close()

	gptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"result":{"message":{"role":"Ассистент","text":"Меня зовут Алиса"},"num_tokens":"20"}}`

		if _, err := fmt.Fprint(w, response); err != nil {
			panic(err)
		}
	}))
	defer gptServer.Close()

	c, err := config.New(configPath, buildInfo, botServer)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	defer func() {
		if errCfg := c.Close(); errCfg != nil {
			t.Error(errCfg)
		}
	}()
	c.Y.APIKey = "test"
	c.Y.URL = gptServer.URL
	c.Y.Client = gptServer.Client()

	chat := &db.Chat{ID: "TestYandexGPT", GPT: true}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, Arguments: "request", debug: true}
	if err = YandexGPT(defaultCtx, e); err != nil {
		t.Errorf("Yandex GPT: %v", err)
	}

	if msg := e.buffer.String(); msg != "Меня зовут Алиса" {
		t.Errorf("failed bot response=%q", msg)
	}
}

func TestGo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var url = strings.TrimRight(r.URL.Path, " /")
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		if url == "/chats/getMembers" {
			response = "{\"members\": [{\"userId\": \"1001\"}, {\"creator\": true, \"userId\": \"user1@my.team\"}, " +
				"{\"userId\": \"1001\"}, {\"creator\": false, \"userId\": \"user2@my.team\"}], \"ok\": true}"
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
	chat := &db.Chat{ID: "TestGo", Active: true}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Go(defaultCtx, e); err != nil {
		t.Errorf("Go: %v", err)
	}
	// no users order guarantee, example "@[user1@my.team]\n@[user2@my.team]"
	respMsg := e.buffer.String()
	if !(len(respMsg) == 33 && strings.HasPrefix(respMsg, "@[user") && strings.HasSuffix(respMsg, "@my.team]")) {
		t.Errorf("failed bot response='%s'", respMsg)
	}
	e.buffer.Reset()
	// with exclude
	chat.ExcludeUsers = map[string]struct{}{"user1@my.team": {}}
	if err = Go(defaultCtx, e); err != nil {
		t.Errorf("Go: %v", err)
	}
	expected := "@[user2@my.team]"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// with url
	chat.URL = "https://github.com/z0rr0/gobot"
	if err = Go(defaultCtx, e); err != nil {
		t.Errorf("Go: %v", err)
	}
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// all users excluded
	chat.ExcludeUsers = map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}}
	if err = Go(defaultCtx, e); err != nil {
		t.Errorf("Go: %v", err)
	}
	expected = "no users :("
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
}

func TestExclude(t *testing.T) {
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
	chat := &db.Chat{ID: "TestExclude"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Exclude(defaultCtx, e); err != nil {
		t.Errorf("Exclude: %v", err)
	}
	expected := "no excluded users"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// show excluded
	chat.ExcludeUsers = map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}}
	if err = Exclude(defaultCtx, e); err != nil {
		t.Errorf("Exclude: %v", err)
	}
	expected = "@[user1@my.team]\n@[user2@my.team]"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// set incorrect value
	e.Arguments = "set user user3@my.team ok?"
	if err = Exclude(defaultCtx, e); err != nil {
		t.Errorf("Exclude: %v", err)
	}
	expected = "no user IDs in arguments"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// set excluded user
	e.Arguments = "set user @[user3@my.team], ok?"
	if err = Exclude(defaultCtx, e); err != nil {
		t.Errorf("Exclude: %v", err)
	}
	expected = "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed bot response='%s', want='%s'", msg, expected)
	}
	expectedExcluded := map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}, "user3@my.team": {}}
	if !maps.Equal(expectedExcluded, chat.ExcludeUsers) {
		t.Error("failed compare excluded users")
	}
	expected = "[\"user1@my.team\",\"user2@my.team\",\"user3@my.team\"]"
	if chat.Exclude != expected {
		t.Errorf("failed chat.Exclude='%s', want='%s'", chat.Exclude, expected)
	}
}

func TestInclude(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var url = strings.TrimRight(r.URL.Path, " /")
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true}"
		if url == "/chats/getMembers" {
			response = "{\"members\": [{\"userId\": \"1001\"}, {\"creator\": true, \"userId\": \"user1@my.team\"}, " +
				"{\"userId\": \"1001\"}, {\"creator\": false, \"userId\": \"user2@my.team\"}], \"ok\": true}"
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
	chat := &db.Chat{ID: "TestInclude"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Include(defaultCtx, e); err != nil {
		t.Errorf("Exclude: %v", err)
	}
	// no users order guarantee, example "@[user1@my.team]\n@[user2@my.team]"
	respMsg := e.buffer.String()
	if !(len(respMsg) == 33 && strings.HasPrefix(respMsg, "@[user") && strings.HasSuffix(respMsg, "@my.team]")) {
		t.Errorf("failed bot response='%s'", respMsg)
	}
	e.buffer.Reset()
	// no excluded users
	e.Arguments = "some value"
	if err = Include(defaultCtx, e); err != nil {
		t.Errorf("Include: %v", err)
	}
	expected := "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// there are excluded users, but failed argument
	chat.ExcludeUsers = map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}}
	e.Arguments = "restore user2, ok?"
	if err = Include(defaultCtx, e); err != nil {
		t.Errorf("Include: %v", err)
	}
	expected = "no user IDs in arguments"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// delete from excluded users
	chat.ExcludeUsers = map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}}
	e.Arguments = "restore @[user2@my.team], ok?"
	if err = Include(defaultCtx, e); err != nil {
		t.Errorf("Include: %v", err)
	}
	expected = "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	expectedExcluded := map[string]struct{}{"user1@my.team": {}}
	if !maps.Equal(expectedExcluded, chat.ExcludeUsers) {
		t.Error("failed compare excluded users")
	}
	expected = "[\"user1@my.team\"]"
	if chat.Exclude != expected {
		t.Errorf("failed chat.Exclude='%s', want='%s'", chat.Exclude, expected)
	}
}

func TestLink(t *testing.T) {
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
	chat := &db.Chat{ID: "TestLink"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected := "no calling URL for this chat"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// chat has URL
	chat.URL = "https://github.com/z0rr0/gobot"
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected = "https://github.com/z0rr0/gobot"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// failed new URL
	e.Arguments = "invalid url value"
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected = "incorrect URL"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// set valid new URL
	e.Arguments = "https://github.com/z0rr0/gobot"
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected = "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// set valid new URL with bad text
	textURL := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa " +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb " +
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	e.Arguments = "https://github.com/z0rr0/gobot " + textURL
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected = "text is too long (max 255 characters)"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// set valid new URL with text
	textURL = "my call text"
	e.Arguments = "https://github.com/z0rr0/gobot " + textURL
	if err = Link(defaultCtx, e); err != nil {
		t.Errorf("Link: %v", err)
	}
	expected = "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	if chat.URLText != textURL {
		t.Errorf("failed chat.URLText='%s', want='%s'", chat.URLText, textURL)
	}
	e.buffer.Reset()
}

func TestResetLink(t *testing.T) {
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
	chat := &db.Chat{ID: "TestResetLink"}
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = ResetLink(defaultCtx, e); err != nil {
		t.Errorf("ResetLink: %v", err)
	}
	expected := "no calling URL for this chat"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()
	// chat has URL
	chat.URL = "https://github.com/z0rr0/gobot"
	chat.URLText = "my call text"
	if err = ResetLink(defaultCtx, e); err != nil {
		t.Errorf("ResetLink: %v", err)
	}
	expected = "success"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	if chat.URL != "" {
		t.Errorf("failed chat.URL='%s', want empty", chat.URL)
	}
	if chat.URLText != "call" {
		t.Errorf("failed chat.URLText='%s', want='call'", chat.URLText)
	}
}

func TestVacation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true, " +
			"\"from\": {\"firstName\": \"A\", \"lastName\": \"B\", \"userId\": \"author@my.team\"}}"
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
	chat := &db.Chat{ID: "TestVacation"}
	if len(chat.ExcludeUsers) > 0 {
		t.Errorf("failed chat.ExcludeUsers='%v', want empty", chat.ExcludeUsers)
	}

	// no author
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Vacation(defaultCtx, e); err != nil {
		t.Errorf("Vacation: %v", err)
	}

	expected := "no valid author user"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if len(chat.ExcludeUsers) > 0 {
		t.Errorf("failed chat.ExcludeUsers='%v', want empty", chat.ExcludeUsers)
	}

	payLoad := botgolang.EventPayload{
		BaseEventPayload: botgolang.BaseEventPayload{
			From: botgolang.Contact{User: botgolang.User{ID: "author@my.team"}},
		},
	}
	e = &Event{Cfg: c, ChatEvent: &botgolang.Event{Payload: payLoad}, Chat: chat, debug: true}

	// add author to exclude users
	if err = Vacation(defaultCtx, e); err != nil {
		t.Errorf("Vacation: %v", err)
	}

	expected = "@[author@my.team] you are on vacation, good luck"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if _, ok := chat.ExcludeUsers["author@my.team"]; !ok {
		t.Errorf("not author in chat.ExcludeUsers: %v", chat.ExcludeUsers)
	}

	// remove author from exclude users
	if err = Vacation(defaultCtx, e); err != nil {
		t.Errorf("Vacation: %v", err)
	}

	expected = "@[author@my.team] you are back from vacation, welcome"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if len(chat.ExcludeUsers) > 0 {
		t.Errorf("failed chat.ExcludeUsers='%v', want empty", chat.ExcludeUsers)
	}
}

func TestSkip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true, " +
			"\"from\": {\"firstName\": \"A\", \"lastName\": \"B\", \"userId\": \"author@my.team\"}}"
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

	chat := &db.Chat{ID: "TestSkip"}

	// no author
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Skip(defaultCtx, e); err != nil {
		t.Errorf("Skip: %v", err)
	}

	expected := "no valid author user"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if len(chat.SkipUsers) > 0 {
		t.Errorf("failed chat.SkipUsers='%v', want empty", chat.SkipUsers)
	}

	payLoad := botgolang.EventPayload{
		BaseEventPayload: botgolang.BaseEventPayload{
			From: botgolang.Contact{User: botgolang.User{ID: "author@my.team"}},
		},
	}
	e = &Event{Cfg: c, ChatEvent: &botgolang.Event{Payload: payLoad}, Chat: chat, debug: true}

	// add author to skip users set
	if err = Skip(defaultCtx, e); err != nil {
		t.Errorf("Skip: %v", err)
	}

	expected = "@[author@my.team] ok, you will be skipped today"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if _, ok := chat.SkipUsers["author@my.team"]; !ok {
		t.Errorf("not author in chat.SkipUsers: %v", chat.SkipUsers)
	}

	// remove author from skip-set users
	if err = Skip(defaultCtx, e); err != nil {
		t.Errorf("Vacation: %v", err)
	}

	expected = "@[author@my.team] ok, you are in the list again"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if len(chat.SkipUsers) > 0 {
		t.Errorf("failed chat.SkipUsers='%v', want empty", chat.SkipUsers)
	}
}

func TestNoDays(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"msgId\": \"7083436385855602743\", \"ok\": true, " +
			"\"from\": {\"firstName\": \"A\", \"lastName\": \"B\", \"userId\": \"author@my.team\"}}"
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

	chat := &db.Chat{ID: "TestNoDays"}

	// no author
	e := &Event{Cfg: c, ChatEvent: &botgolang.Event{}, Chat: chat, debug: true}
	if err = Skip(defaultCtx, e); err != nil {
		t.Errorf("Skip: %v", err)
	}

	expected := "no valid author user"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	if len(chat.WeekDays) > 0 {
		t.Errorf("failed chat.WeekDays='%v', want empty", chat.WeekDays)
	}

	payLoad := botgolang.EventPayload{
		BaseEventPayload: botgolang.BaseEventPayload{
			From: botgolang.Contact{User: botgolang.User{ID: "author@my.team"}},
		},
	}
	e = &Event{Cfg: c, ChatEvent: &botgolang.Event{Payload: payLoad}, Chat: chat, Arguments: "2 5", debug: true}

	// add noDays for author
	if err = NoDays(defaultCtx, e); err != nil {
		t.Errorf("NoDays: %v", err)
	}

	expected = "@[author@my.team] days are set: Tuesday, Friday"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	expectedMap := map[time.Weekday]map[string]struct{}{
		time.Tuesday: {"author@my.team": {}},
		time.Friday:  {"author@my.team": {}},
	}

	if len(chat.WeekDays) != len(expectedMap) {
		t.Errorf("failed chat.WeekDays='%v', want='%v'", chat.WeekDays, expectedMap)
	}

	for day, m := range expectedMap {
		chatMap, ok := chat.WeekDays[day]

		if ok {
			if !maps.Equal(chatMap, m) {
				t.Errorf("failed chat.WeekDays[%v]='%v', want='%v'", day, chatMap, m)
			}
		} else {
			t.Errorf("failed chat.WeekDays[%v]", day)
		}
	}

	// update user's noDays
	e = &Event{Cfg: c, ChatEvent: &botgolang.Event{Payload: payLoad}, Chat: chat, Arguments: "3", debug: true}
	if err = NoDays(defaultCtx, e); err != nil {
		t.Errorf("NoDays: %v", err)
	}

	expected = "@[author@my.team] days are set: Wednesday"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	expected = "{\"3\":[\"author@my.team\"]}"
	if chat.Days != expected {
		t.Errorf("failed chat.Days='%s', want='%s'", chat.Days, expected)
	}

	// reset user's noDays
	e = &Event{Cfg: c, ChatEvent: &botgolang.Event{Payload: payLoad}, Chat: chat, debug: true}
	if err = NoDays(defaultCtx, e); err != nil {
		t.Errorf("NoDays: %v", err)
	}

	expected = "@[author@my.team] days are cleaned"
	if msg := e.buffer.String(); msg != expected {
		t.Errorf("failed msg='%s', want='%s'", msg, expected)
	}
	e.buffer.Reset()

	expected = ""
	if chat.Days != expected {
		t.Errorf("failed chat.Days='%s', want='%s'", chat.Days, expected)
	}

	if len(chat.WeekDays) != 0 {
		t.Errorf("failed chat.WeekDays='%v', want empty", chat.WeekDays)
	}
}

func TestEvent_ArgsUserIDs(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected map[string]struct{}
	}{
		{
			name:     "empty",
			input:    "",
			expected: nil,
		},
		{
			name:     "no users",
			input:    "some value",
			expected: nil,
		},
		{
			name:     "one user",
			input:    " @[user2@my.team] other ignored",
			expected: map[string]struct{}{"user2@my.team": {}},
		},
		{
			name:     "many users",
			input:    " @[user2@my.team], @[user1@my.team]; @[user2@my.team] @[user3@my.team]",
			expected: map[string]struct{}{"user1@my.team": {}, "user2@my.team": {}, "user3@my.team": {}},
		},
		{
			name:     "invalid names",
			input:    "user2@my.team other ignored",
			expected: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(tt *testing.T) {
			e := &Event{Arguments: c.input}
			result := e.ArgsUserIDs()
			if !maps.Equal(result, c.expected) {
				tt.Errorf("failed case='%s' result='%v', expected='%v'", c.name, result, c.expected)
			}
		})
	}
}

func FuzzEventArgsUserIDs(f *testing.F) {
	cases := []string{
		"",
		" ",
		"some value",
		" @[user2@my.team] other ignored",
		" @[user2@my.team], @[user2@my.team]; @[user2@my.team]",
		"user2@my.team other ignored",
	}
	for _, c := range cases {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, orig string) {
		e := &Event{Arguments: orig}
		result := e.ArgsUserIDs()
		for userID := range result {
			if len(userID) == 0 {
				t.Error("failed len")
			}
			userName := fmt.Sprintf("@[%s]", userID)
			if !strings.Contains(orig, userName) {
				t.Error("no username")
			}
		}
	})
}
