package db

import (
	"context"
	"database/sql"
	"errors"
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
	const query = "SELECT `id`, `active`, `exclude`, `skip`, `days`, `url`, `url_text`, `created`, `updated`, `gpt` " +
		"FROM `chat` WHERE `id`=? LIMIT 1;"
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exist statement: %w", err)
	}
	chat := &Chat{}
	err = stmt.QueryRowContext(ctx, id).Scan(
		&chat.ID, &chat.Active, &chat.Exclude, &chat.Skip, &chat.Days,
		&chat.URL, &chat.URLText, &chat.Created, &chat.Updated, &chat.GPT,
	)

	if err != nil {
		return nil, fmt.Errorf("chat scan: %w", err)
	}

	if err = stmt.Close(); err != nil {
		return nil, fmt.Errorf("close exist statement: %w", err)
	}

	if err = chat.Unmarshal(); err != nil {
		return nil, err
	}

	return chat, nil
}

// GetOrCreate loads a chat by its ID or creates a new one but without saving it.
func GetOrCreate(ctx context.Context, db *sql.DB, id string) (*Chat, error) {
	chat, err := Get(ctx, db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// unknown chat
			now := time.Now().UTC()
			return &Chat{ID: id, Created: now, Updated: now}, nil
		}
		return nil, fmt.Errorf("chat load: %w", err)
	}
	chat.Saved = true
	return chat, nil
}

// CleanSkip removes all skip columns from all chats.
func CleanSkip(ctx context.Context, db *sql.DB) error {
	const query = "UPDATE `chat` SET `skip`='' WHERE `skip` != '';"
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("exist statement: %w", err)
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("clean skip exec: %w", err)
	}

	if err = stmt.Close(); err != nil {
		return fmt.Errorf("close exist statement: %w", err)
	}

	return nil
}
