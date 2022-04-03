package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// InTransaction runs method `f` inside the database transaction and does commit or rollback.
func InTransaction(ctx context.Context, db *sql.DB, f func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed transaction begin: %w", err)
	}
	err = f(tx)
	if err != nil {
		err = fmt.Errorf("rollback transaction: %w", err)
		e := tx.Rollback()
		if e != nil {
			err = fmt.Errorf("failed rollback: %v: %w", err, e)
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed transaction commit: %w", err)
	}
	return nil
}

// Get returns a chat's pointer by its ID.
func Get(ctx context.Context, db *sql.DB, id string) (*Chat, error) {
	const query = "SELECT `id`, `active`, `exclude`, `url`, `created`, `updated` " +
		"FROM `chat` WHERE `id`=? LIMIT 1;"
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exist statement: %w", err)
	}
	chat := &Chat{}
	err = stmt.QueryRowContext(ctx, id).Scan(
		&chat.ID, &chat.Active, &chat.Exclude, &chat.URL, &chat.Created, &chat.Updated,
	)
	if err != nil {
		return nil, err
	}
	if err = stmt.Close(); err != nil {
		return nil, fmt.Errorf("close exist statement: %w", err)
	}
	if err = chat.ExcludeToMap(); err != nil {
		return nil, err
	}
	return chat, nil
}

// UpsertActive updates the chat's active status.
// It is used to create a new chat item.
func UpsertActive(db *sql.DB, id string, active bool, timeout time.Duration) error {
	var now = time.Now().UTC()
	chat := &Chat{ID: id, Active: active, Created: now, Updated: now}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return chat.Upsert(ctx, db)
}
