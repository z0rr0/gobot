package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
)

const (
	// dbPath is the path of temporary database file.
	dbPath = "/tmp/gobot_db_test.sqlite"
)

func compareMap(m1, m2 map[string]struct{}) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k := range m1 {
		if _, ok := m2[k]; !ok {
			return false
		}
	}
	return true
}

func open() (*sql.DB, error) {
	return sql.Open("sqlite3", dbPath)
}

func TestGet(t *testing.T) {
	const chatID = "TestGet"
	db, err := open()
	if err != nil {
		t.Fatalf("failed to open database: %s", err)
	}
	defer func() {
		if e := db.Close(); e != nil {
			t.Errorf("failed to close database: %s", e)
		}
	}()
	ctx := context.Background()
	now := time.Now().UTC()
	chat := Chat{
		ID:      chatID,
		Active:  true,
		Exclude: "[\"user1\",\"user2\"]",
		URL:     "https://github.com/",
		Created: now,
		Updated: now,
	}
	if err = chat.Upsert(ctx, db); err != nil {
		t.Fatalf("failed to upsert chat: %s", err)
	}
	gottenChat, err := Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if gottenChat == nil {
		t.Fatal("got nil chat")
	}
	if !chat.Equal(gottenChat) {
		t.Errorf("got chat %+v, want %+v", gottenChat, chat)
	}
}

func TestUpsertActive(t *testing.T) {
	const chatID = "TestUpsertActive"
	db, err := open()
	if err != nil {
		t.Fatalf("failed to open database: %s", err)
	}
	defer func() {
		if e := db.Close(); e != nil {
			t.Errorf("failed to close database: %s", e)
		}
	}()
	ctx := context.Background()
	if err = UpsertActive(db, chatID, true, time.Second); err != nil {
		t.Fatalf("failed to upsert active chat: %s", err)
	}
	chat, err := Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if (chat.ID != chatID) || !chat.Active {
		t.Errorf("got chat %+v, want %+v", chat, Chat{ID: chatID, Active: true})
	}
	// reset active
	if err = UpsertActive(db, chatID, false, time.Second); err != nil {
		t.Fatalf("failed to upsert not active chat: %s", err)
	}
	chat, err = Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if (chat.ID != chatID) || chat.Active {
		t.Errorf("got chat %+v, want %+v", chat, Chat{ID: chatID, Active: true})
	}
}

func TestChat_Update(t *testing.T) {
	const chatID = "TestChat_Update"
	db, err := open()
	if err != nil {
		t.Fatalf("failed to open database: %s", err)
	}
	defer func() {
		if e := db.Close(); e != nil {
			t.Errorf("failed to close database: %s", e)
		}
	}()
	ctx := context.Background()
	now := time.Now().UTC()
	chat := Chat{
		ID:      chatID,
		Active:  true,
		Exclude: "[\"user1\",\"user2\"]",
		URL:     "https://github.com/",
		Created: now,
		Updated: now,
	}
	if err = chat.Upsert(ctx, db); err != nil {
		t.Fatalf("failed to upsert chat: %s", err)
	}
	dbChat, err := Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if !chat.Equal(dbChat) {
		t.Errorf("got chat\n%+v, want\n%+v", dbChat, chat)
	}
	// change and update
	chat.Active = false
	chat.Created = time.Now().UTC()
	chat.Exclude = "[\"user3\",\"user4\"]"
	chat.URL = "https://gitlab.com/"
	if err = chat.Update(ctx, db); err != nil {
		t.Fatalf("failed to update chat: %s", err)
	}
	dbChat, err = Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if !chat.Equal(dbChat) {
		t.Errorf("got chat\n%+v\n want\n%+v", dbChat, chat)
	}
}

func TestExclude(t *testing.T) {
	const chatID = "TestExclude"
	db, err := open()
	if err != nil {
		t.Fatalf("failed to open database: %s", err)
	}
	defer func() {
		if e := db.Close(); e != nil {
			t.Errorf("failed to close database: %s", e)
		}
	}()
	now := time.Now().UTC()
	chat := Chat{
		ID:      chatID,
		Active:  true,
		Exclude: "[\"user1\",\"user2\"]",
		URL:     "https://github.com/",
		Created: now,
		Updated: now,
	}
	if err = chat.ExcludeToMap(); err != nil {
		t.Fatalf("failed to load exlclude: %v", err)
	}
	expected := map[string]struct{}{"user1": {}, "user2": {}}
	if !compareMap(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	chat.AddExclude(map[string]struct{}{"user0": {}})
	expected = map[string]struct{}{"user0": {}, "user1": {}, "user2": {}}
	if !compareMap(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	if err = chat.ExcludeToString(); err != nil {
		t.Errorf("failed convert exclude to string: %v", err)
	}
	expectedStr := "[\"user0\",\"user1\",\"user2\"]"
	if chat.Exclude != expectedStr {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chat.Exclude, expectedStr)
	}
	// delete
	chat.DelExclude("user2")
	expected = map[string]struct{}{"user0": {}, "user1": {}}
	if !compareMap(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	if err = chat.ExcludeToString(); err != nil {
		t.Errorf("failed convert exclude to string: %v", err)
	}
	expectedStr = "[\"user0\",\"user1\"]"
	if chat.Exclude != expectedStr {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chat.Exclude, expectedStr)
	}
}
