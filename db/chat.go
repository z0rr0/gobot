package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Chat is a struct for chat's info.
type Chat struct {
	ID            string    `db:"id"`
	Active        bool      `db:"active"`
	Exclude       string    `db:"exclude"`
	URL           string    `db:"url"`
	URLText       string    `db:"url_text"`
	Positions     string    `db:"positions"`
	Created       time.Time `db:"created_at"`
	Updated       time.Time `db:"updated_at"`
	ExcludeUsers  map[string]struct{}
	UserPositions map[string]int
	Saved         bool
}

// Equal returns true if the two chats are equal.
func (chat *Chat) Equal(c *Chat) bool {
	value := chat.ID == c.ID && chat.Active == c.Active && chat.Exclude == c.Exclude && chat.URL == c.URL
	value = value && chat.URLText == c.URLText
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
	if len(chat.ExcludeUsers) == 0 {
		chat.Exclude = ""
		return nil
	}
	exclude := make([]string, 0, len(chat.ExcludeUsers))
	for userID := range chat.ExcludeUsers {
		exclude = append(exclude, userID)
	}
	sort.Strings(exclude)
	b, err := json.Marshal(&exclude)
	if err != nil {
		return fmt.Errorf("failed to marshal exclude: %w", err)
	}
	chat.Exclude = string(b)
	return nil
}

// ExcludeToMap loads exclude users set from a string.
func (chat *Chat) ExcludeToMap() error {
	if chat.Exclude == "" {
		chat.ExcludeUsers = nil
		return nil
	}
	var exclude []string
	err := json.Unmarshal([]byte(chat.Exclude), &exclude)
	if err != nil {
		return fmt.Errorf("failed to unmarshal exclude: %w", err)
	}
	chat.ExcludeUsers = make(map[string]struct{}, len(exclude))
	for _, userID := range exclude {
		chat.ExcludeUsers[userID] = struct{}{}
	}
	return nil
}

// AddPositions adds user to a position map.
func (chat *Chat) AddPositions(userPositions map[string]int) {
	if chat.UserPositions == nil {
		chat.UserPositions = make(map[string]int)
	}
	for userName, position := range userPositions {
		chat.UserPositions[userName] = position
	}
}

// DelPositions removes user from a position map.
func (chat *Chat) DelPositions(userPositions map[string]int) {
	if chat.UserPositions == nil {
		return
	}
	for userName := range userPositions {
		delete(chat.UserPositions, userName)
	}
}

// UserPositionsToString returns a string of user positions map.
func (chat *Chat) UserPositionsToString() error {
	if len(chat.UserPositions) == 0 {
		chat.Positions = ""
		return nil
	}
	b, err := json.Marshal(&chat.UserPositions)
	if err != nil {
		return fmt.Errorf("failed to marshal positions: %w", err)
	}
	chat.Positions = string(b)
	return nil
}

// PositionsToMap loads user positions map from a string.
func (chat *Chat) PositionsToMap() error {
	if chat.Positions == "" {
		chat.UserPositions = nil
		return nil
	}
	err := json.Unmarshal([]byte(chat.Positions), &chat.UserPositions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal postions: %w", err)
	}
	return nil
}

func (chat *Chat) beforeSave() error {
	err := chat.ExcludeToString()
	if err != nil {
		return fmt.Errorf("failed to convert exclude to string: %w", err)
	}
	err = chat.UserPositionsToString()
	if err != nil {
		return fmt.Errorf("failed to convert positions to string: %w", err)
	}
	return nil
}

func (chat *Chat) afterLoad() error {
	err := chat.ExcludeToMap()
	if err != nil {
		return fmt.Errorf("failed to convert exclude to map: %w", err)
	}
	err = chat.PositionsToMap()
	if err != nil {
		return fmt.Errorf("failed to convert positions to map: %w", err)
	}
	return nil
}

// Update saves chat's info.
func (chat *Chat) Update(ctx context.Context, db *sql.DB) error {
	const query = "UPDATE `chat` " +
		"SET `active`=?, `exclude`=?, `url`=?, `url_text`=?, `positions`=?, `created`=?, `updated`=? " +
		"WHERE `id`=?"
	if e := chat.beforeSave(); e != nil {
		return e
	}
	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, stmt).ExecContext(
			ctx, chat.Active, chat.Exclude, chat.URL, chat.URLText, chat.Positions,
			chat.Created, time.Now().UTC(), chat.ID,
		)
		if err != nil {
			return fmt.Errorf("upsert exec: %w", err)
		}
		chat.Saved = true
		return nil
	})
}

// Upsert inserts or updates a chat, make it active.
func (chat *Chat) Upsert(ctx context.Context, db *sql.DB) error {
	const query = "INSERT INTO `chat` " +
		"(`id`, `active`, `exclude`, `url`, `url_text`, `positions`, `created`, `updated`)  " +
		"VALUES (?,?,?,?,?,?,?,?) " +
		"ON CONFLICT(id) DO UPDATE SET `active`=?, `updated`=?;"
	if e := chat.beforeSave(); e != nil {
		return e
	}
	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, stmt).ExecContext(
			ctx, chat.ID, chat.Active, chat.Exclude, chat.URL, chat.URLText, chat.Positions,
			chat.Created, chat.Updated, chat.Active, chat.Updated,
		)
		if err != nil {
			return fmt.Errorf("upsert exec: %w", err)
		}
		chat.Saved = true
		return nil
	})
}
