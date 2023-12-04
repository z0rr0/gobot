package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Chat is a struct for chat's info.
type Chat struct {
	ID           string    `db:"id"`
	Active       bool      `db:"active"`
	GPT          bool      `db:"gpt"`
	Exclude      string    `db:"exclude"`
	Skip         string    `db:"skip"`
	Days         string    `db:"days"`
	URL          string    `db:"url"`
	URLText      string    `db:"url_text"`
	Created      time.Time `db:"created_at"`
	Updated      time.Time `db:"updated_at"`
	ExcludeUsers map[string]struct{}
	SkipUsers    map[string]struct{}
	WeekDays     map[time.Weekday]map[string]struct{}
	Saved        bool
}

// Equal returns true if the two chats are equal.
func (chat *Chat) Equal(c *Chat) bool {
	value := chat.ID == c.ID && chat.Active == c.Active && chat.Exclude == c.Exclude && chat.Skip == c.Skip
	value = value && chat.URL == c.URL && chat.URLText == c.URLText
	return value && chat.Created.Equal(c.Created) // updated chan be change automatically
}

// AddExclude adds user to an exclude set.
func (chat *Chat) AddExclude(userIDs map[string]struct{}) {
	if chat.ExcludeUsers == nil {
		chat.ExcludeUsers = make(map[string]struct{})
	}
	for userID := range userIDs {
		chat.ExcludeUsers[userID] = struct{}{}
	}
}

// DelExclude removes user from an exclude set.
func (chat *Chat) DelExclude(userIDs map[string]struct{}) {
	if chat.ExcludeUsers == nil {
		return
	}
	for userID := range userIDs {
		delete(chat.ExcludeUsers, userID)
	}
}

// ExcludeToString returns a string of exclude users set.
func (chat *Chat) ExcludeToString() error {
	s, err := setToString(chat.ExcludeUsers)
	if err != nil {
		return err
	}

	chat.Exclude = s
	return nil
}

// ExcludeToMap loads exclude users set from a string.
func (chat *Chat) ExcludeToMap() error {
	m, err := stringToSet(chat.Exclude)
	if err != nil {
		return err
	}

	chat.ExcludeUsers = m
	return nil
}

// SkipToString returns a string of skipped users set.
func (chat *Chat) SkipToString() error {
	s, err := setToString(chat.SkipUsers)
	if err != nil {
		return err
	}

	chat.Skip = s
	return nil
}

// SkipToMap loads skipped users set from a string.
func (chat *Chat) SkipToMap() error {
	m, err := stringToSet(chat.Skip)
	if err != nil {
		return err
	}

	chat.SkipUsers = m
	return nil
}

// AddSkip adds user to an skip-set.
func (chat *Chat) AddSkip(userID string) {
	if chat.SkipUsers == nil {
		chat.SkipUsers = make(map[string]struct{})
	}
	chat.SkipUsers[userID] = struct{}{}
}

// DelSkip removes user from an skip-set.
func (chat *Chat) DelSkip(userID string) {
	delete(chat.SkipUsers, userID)
}

// MarshalDays converts week days to a string.
func (chat *Chat) MarshalDays() error {
	if len(chat.WeekDays) == 0 {
		chat.Days = ""
		return nil
	}

	data := make(map[int][]string, len(chat.WeekDays))

	for k, v := range chat.WeekDays {
		dayNumber := int(k)
		data[dayNumber] = make([]string, 0, len(v))

		for vk := range v {
			data[dayNumber] = append(data[dayNumber], vk)
		}
	}

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal days: %w", err)
	}

	chat.Days = string(b)
	return nil
}

// UnmarshalDays converts a string to week days.
func (chat *Chat) UnmarshalDays() error {
	if chat.Days == "" {
		chat.WeekDays = nil
		return nil
	}

	data := make(map[int][]string)

	err := json.Unmarshal([]byte(chat.Days), &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal days: %w", err)
	}

	chat.WeekDays = make(map[time.Weekday]map[string]struct{}, len(data))
	for k, v := range data {
		chat.WeekDays[time.Weekday(k)] = make(map[string]struct{}, len(v))
		for i := range v {
			chat.WeekDays[time.Weekday(k)][v[i]] = struct{}{}
		}
	}

	return nil
}

// Marshal converts exclude and skip sets to strings.
func (chat *Chat) Marshal() error {
	if err := chat.ExcludeToString(); err != nil {
		return err
	}

	if err := chat.SkipToString(); err != nil {
		return err
	}

	if err := chat.MarshalDays(); err != nil {
		return err
	}

	return nil
}

// Unmarshal converts exclude and skip strings to sets.
func (chat *Chat) Unmarshal() error {
	if err := chat.ExcludeToMap(); err != nil {
		return err
	}

	if err := chat.SkipToMap(); err != nil {
		return err
	}

	if err := chat.UnmarshalDays(); err != nil {
		return err
	}

	return nil
}

// Update saves chat's info.
func (chat *Chat) Update(ctx context.Context, db *sql.DB) error {
	const query = "UPDATE `chat` " +
		"SET `active`=?, `exclude`=?, `skip`=?, `days`=?, `url`=?, `url_text`=?, `created`=?, `updated`=? " +
		"WHERE `id`=?"
	if e := chat.Marshal(); e != nil {
		return e
	}

	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, stmt).ExecContext(
			ctx, chat.Active, chat.Exclude, chat.Skip, chat.Days,
			chat.URL, chat.URLText, chat.Created, time.Now().UTC(), chat.ID,
		)
		if err != nil {
			return fmt.Errorf("upsert exec: %w", err)
		}

		if err = stmt.Close(); err != nil {
			return fmt.Errorf("close exist statement: %w", err)
		}

		chat.Saved = true
		return nil
	})
}

// Upsert inserts or updates a chat, make it active.
func (chat *Chat) Upsert(ctx context.Context, db *sql.DB) error {
	const query = "INSERT INTO `chat` " +
		"(`id`, `active`, `exclude`, `skip`, `days`, `url`, `url_text`, `created`, `updated`)  VALUES (?,?,?,?,?,?,?,?,?) " +
		"ON CONFLICT(id) DO UPDATE SET `active`=?, `updated`=?;"

	if e := chat.Marshal(); e != nil {
		return e
	}

	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, stmt).ExecContext(
			ctx, chat.ID, chat.Active, chat.Exclude, chat.Skip, chat.Days, chat.URL, chat.URLText,
			chat.Created, chat.Updated, chat.Active, chat.Updated,
		)
		if err != nil {
			return fmt.Errorf("upsert exec: %w", err)
		}

		if err = stmt.Close(); err != nil {
			return fmt.Errorf("close exist statement: %w", err)
		}

		chat.Saved = true
		return nil
	})
}
