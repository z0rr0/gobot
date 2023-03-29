package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func compareSets(a, b map[string]struct{}) bool {
	n, m := len(a), len(b)
	if n != m {
		return false
	}
	for v := range a {
		if _, ok := b[v]; !ok {
			return false
		}
	}
	return true
}

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
	if !compareSets(expectedExcluded, chat.ExcludeUsers) {
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
	if !compareSets(expectedExcluded, chat.ExcludeUsers) {
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
			if !compareSets(result, c.expected) {
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
