package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if c.Bt.Info.ID != "123" {
		t.Errorf("c.Bt.Info.ID = %v, want %v", c.Bt.Info.ID, 123)
	}
}

func TestCleanFileName(t *testing.T) {
	currentDir, wdErr := os.Getwd()
	if wdErr != nil {
		t.Fatalf("Failed to get current directory: %s", wdErr)
	}

	testCases := []struct {
		name         string
		fileName     string
		allowedPaths []string
		want         string
		wantErr      bool
	}{
		{
			name:     "simple file name",
			fileName: "test.txt",
			want:     filepath.Join(currentDir, "test.txt"),
		},
		{
			name:         "absolute path allowed",
			fileName:     "/usr/local/test.txt",
			allowedPaths: []string{"/usr/local/"},
			want:         "/usr/local/test.txt",
		},
		{
			name:         "several paths",
			fileName:     "/tmp/test.txt",
			allowedPaths: []string{"/data", "/tmp"},
			want:         "/tmp/test.txt",
		},
		{
			name:         "several paths deny",
			fileName:     "/tmp/test.txt",
			allowedPaths: []string{"/data", "/var"},
			wantErr:      true,
		},
		{
			name:     "absolute path not allowed",
			fileName: "/etc/passwd",
			wantErr:  true,
		},
		{
			name:     "trim spaces",
			fileName: "   testfile.txt  ",
			want:     filepath.Join(currentDir, "testfile.txt"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CleanFileName(tc.fileName, tc.allowedPaths...)

			if (err != nil) != tc.wantErr {
				t.Errorf("CleanFileName() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if got != tc.want {
				t.Errorf("CleanFileName() = %v, want %v", got, tc.want)
			}
		})
	}
}
