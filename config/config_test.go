package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	// configPath is the path of temporary configuration file.
	configPath = "/tmp/gobot_config_test.toml"
)

var (
	buildInfo = &BuildInfo{Name: "config_test"}
)

func TestNew(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := "{\"about\": \"Бот для командных чатов\", \"firstName\": \"goBot\", " +
			"\"nick\": \"goBot\", \"userId\": \"123\", \"ok\": true}"
		_, err := fmt.Fprint(w, response)
		if err != nil {
			t.Error(err)
		}
	})
	s := httptest.NewServer(handler)
	defer s.Close()

	_, err := New("/bad_name.toml", buildInfo, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
	c, err := New(configPath, buildInfo, s)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		e := c.Close()
		if e != nil {
			t.Error(e)
		}
	}()
	if c.Bot.Info.ID != "123" {
		t.Errorf("c.Bot.Info.ID = %v, want %v", c.Bot.Info.ID, 123)
	}
}
