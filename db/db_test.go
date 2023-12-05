package db

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
)

const (
	// dbPath is the path of temporary database file.
	dbPath = "/tmp/gobot_db_test.sqlite"
)

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
		Skip:    "[\"user3\",\"user4\"]",
		Days:    "{\"2\":[\"user1\"]",
		URL:     "https://github.com/",
		URLText: "GitHub",
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

func TestCleanSkip(t *testing.T) {
	const chatID = "TestCleanSkip"

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

	chat, err := GetOrCreate(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get or create chat: %s", err)
	}

	err = chat.Upsert(ctx, db)
	if err != nil {
		t.Fatalf("failed to upsert chat: %s", err)
	}

	chat.AddSkip("test1")
	chat.AddSkip("test2")

	if err = chat.Update(ctx, db); err != nil {
		t.Fatalf("failed to update chat: %s", err)
	}

	expected := "[\"test1\",\"test2\"]"
	if chat.Skip != expected {
		t.Errorf("failed compare skip string, current '%v' expected '%v'", chat.Skip, expected)
	}

	if err = CleanSkip(ctx, db); err != nil {
		t.Fatalf("failed to clean skip: %s", err)
	}

	chat, err = Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}

	if chat.Skip != "" {
		t.Errorf("failed compare skip string, current '%v' expected ''", chat.Skip)
	}

	if chat.SkipUsers != nil {
		t.Errorf("failed compare skip users, current '%v' expected nil", chat.SkipUsers)
	}
}

func TestGetOrCreate(t *testing.T) {
	const (
		chatID          = "TestGetOrCreate"
		chatIDNotExists = "TestGetOrCreateNotExists"
	)
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
		ID:           chatID,
		Active:       true,
		ExcludeUsers: map[string]struct{}{"user1": {}, "user2": {}},
		SkipUsers:    map[string]struct{}{"user3": {}, "user4": {}},
		WeekDays:     map[time.Weekday]map[string]struct{}{time.Tuesday: {"user1": {}}},
		URL:          "https://github.com/",
		URLText:      "GitHub",
		Created:      now,
		Updated:      now,
	}
	if err = chat.Upsert(ctx, db); err != nil {
		t.Fatalf("failed to upsert chat: %s", err)
	}
	chatNew, err := GetOrCreate(ctx, db, chatIDNotExists)
	if err != nil {
		t.Fatalf("failed to get or create chat: %s", err)
	}
	if chatNew.Active {
		t.Error("got active chat")
	}
	_, err = Get(ctx, db, chatIDNotExists)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("got chat want ErrNoRows")
	}

	chatNew, err = GetOrCreate(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get or create chat: %s", err)
	}
	if chatNew.ID != chat.ID {
		t.Errorf("got chat %+v, want %+v", chatNew, chat)
	}

	expected := "[\"user1\",\"user2\"]"
	if chatNew.Exclude != expected {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chatNew.Exclude, expected)
	}

	expected = "[\"user3\",\"user4\"]"
	if chatNew.Skip != expected {
		t.Errorf("failed compare skip string, current '%v' expected '%v'", chatNew.Skip, expected)
	}

	expected = "{\"2\":[\"user1\"]}"
	if chatNew.Days != expected {
		t.Errorf("failed compare days string, current %q expected %q", chatNew.Days, expected)
	}

	if !maps.Equal(chatNew.ExcludeUsers, chat.ExcludeUsers) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chatNew.ExcludeUsers, chat.ExcludeUsers)
	}

	if !maps.Equal(chatNew.SkipUsers, chat.SkipUsers) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chatNew.SkipUsers, chat.SkipUsers)
	}
}

func TestChat_Upsert(t *testing.T) {
	const chatID = "TestChat_Upsert"
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
	chat := &Chat{ID: chatID, Active: true}
	if err = chat.Upsert(ctx, db); err != nil {
		t.Fatalf("failed to upsert active chat: %s", err)
	}
	chat, err = Get(ctx, db, chatID)
	if err != nil {
		t.Fatalf("failed to get chat: %s", err)
	}
	if (chat.ID != chatID) || !chat.Active {
		t.Errorf("got chat %+v, want %+v", chat, Chat{ID: chatID, Active: true})
	}
	// reset active
	chat.Active = false
	if err = chat.Upsert(ctx, db); err != nil {
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
		ID:           chatID,
		Active:       true,
		ExcludeUsers: map[string]struct{}{"user1": {}, "user2": {}},
		SkipUsers:    map[string]struct{}{"user3": {}},
		WeekDays:     map[time.Weekday]map[string]struct{}{time.Tuesday: {"user1": {}}},
		URL:          "https://github.com/",
		URLText:      "GitHub",
		Created:      now,
		Updated:      now,
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
	chat.AddExclude(map[string]struct{}{"user5": {}})
	chat.AddSkip("user4")
	chat.URL = "https://gitlab.com/"
	chat.URLText = "GitLab"
	chat.WeekDays[time.Wednesday] = map[string]struct{}{"user2": {}}

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

func TestChat_ExcludeToMap(t *testing.T) {
	now := time.Now().UTC()
	chat := Chat{
		ID:      "TestChat_ExcludeToMap",
		Active:  true,
		Exclude: "[\"user1\",\"user2\"]",
		URL:     "https://github.com/",
		URLText: "GitHub",
		Created: now,
		Updated: now,
	}
	if err := chat.ExcludeToMap(); err != nil {
		t.Fatalf("failed to load exlclude: %v", err)
	}
	expected := map[string]struct{}{"user1": {}, "user2": {}}
	if !maps.Equal(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
}

func TestChat_ExcludeToString(t *testing.T) {
	now := time.Now().UTC()
	chat := Chat{
		ID:           "TestChat_ExcludeToMap",
		Active:       true,
		Exclude:      "[\"user1\",\"user2\"]",
		URL:          "https://github.com/",
		URLText:      "GitHub",
		Created:      now,
		Updated:      now,
		ExcludeUsers: map[string]struct{}{"user1": {}, "user2": {}},
	}
	chat.AddExclude(map[string]struct{}{"user0": {}})
	expected := map[string]struct{}{"user0": {}, "user1": {}, "user2": {}}
	if !maps.Equal(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	if err := chat.ExcludeToString(); err != nil {
		t.Errorf("failed convert exclude to string: %v", err)
	}
	expectedStr := "[\"user0\",\"user1\",\"user2\"]"
	if chat.Exclude != expectedStr {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chat.Exclude, expectedStr)
	}
}

func TestChat_AddExclude(t *testing.T) {
	now := time.Now().UTC()
	chat := Chat{
		ID:      "TestChat_ExcludeToMap",
		Active:  true,
		URL:     "https://github.com/",
		URLText: "GitHub",
		Created: now,
		Updated: now,
	}
	chat.AddExclude(map[string]struct{}{"user0": {}, "user1": {}})
	chat.AddExclude(map[string]struct{}{"user2": {}})
	expected := map[string]struct{}{"user0": {}, "user1": {}, "user2": {}}
	if !maps.Equal(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	if err := chat.ExcludeToString(); err != nil {
		t.Errorf("failed convert exclude to string: %v", err)
	}
	expectedStr := "[\"user0\",\"user1\",\"user2\"]"
	if chat.Exclude != expectedStr {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chat.Exclude, expectedStr)
	}
}

func TestChat_DelExclude(t *testing.T) {
	now := time.Now().UTC()
	chat := Chat{
		ID:      "TestChat_ExcludeToMap",
		Active:  true,
		URL:     "https://github.com/",
		URLText: "GitHub",
		Created: now,
		Updated: now,
	}
	chat.DelExclude(map[string]struct{}{"user2": {}})
	if chat.Exclude != "" {
		t.Errorf("failed compare exclude string, current '%v' expected ''", chat.Exclude)
	}
	if chat.ExcludeUsers != nil {
		t.Errorf("failed compare exclude users, current '%v' expected nil", chat.ExcludeUsers)
	}
	chat.Exclude = "[\"user0\",\"user1\",\"user2\"]"
	chat.ExcludeUsers = map[string]struct{}{"user0": {}, "user1": {}, "user2": {}}
	// delete some value
	chat.DelExclude(map[string]struct{}{"user2": {}})
	expected := map[string]struct{}{"user0": {}, "user1": {}}
	if !maps.Equal(chat.ExcludeUsers, expected) {
		t.Fatalf("failed compare maps, current:\n%+v\n want\n%+v", chat.ExcludeUsers, expected)
	}
	if err := chat.ExcludeToString(); err != nil {
		t.Errorf("failed convert exclude to string: %v", err)
	}
	expectedStr := "[\"user0\",\"user1\"]"
	if chat.Exclude != expectedStr {
		t.Errorf("failed compare exclude string, current '%v' expected '%v'", chat.Exclude, expectedStr)
	}
}
