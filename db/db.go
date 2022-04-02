package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Chat is a struct for chat's info.
type Chat struct {
	ID      string    `db:"id"`
	Active  bool      `db:"active"`
	Exclude string    `db:"exclude"`
	URL     string    `db:"url"`
	Created time.Time `db:"created_at"`
	Updated time.Time `db:"updated_at"`
}

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
	const query = "SELECT `id`, `active`, `exclude`, `url` " +
		"FROM `chat` " +
		"WHERE `id`=? " +
		"LIMIT 1;"
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exist statement: %w", err)
	}
	item := &Chat{}
	err = stmt.QueryRowContext(ctx, id).Scan(&item.ID, &item.Active, &item.Exclude, &item.URL)
	if err != nil {
		return nil, err
	}
	err = stmt.Close()
	if err != nil {
		return nil, fmt.Errorf("close exist statement: %w", err)
	}
	return item, nil
}

// Upsert inserts or updates a chat, make it active.
func (chat *Chat) Upsert(ctx context.Context, db *sql.DB) error {
	const query = "INSERT INTO `chat` " +
		"(`id`, `active`, `exclude`, `url`, `created`, `updated`)  VALUES (?,?,?,?,?,?) " +
		"ON CONFLICT(id) DO UPDATE " +
		"SET `active`=?, `updated`=?;"
	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		now := time.Now().UTC()
		_, err = tx.StmtContext(ctx, stmt).ExecContext(
			ctx, chat.ID, chat.Active, chat.Exclude, chat.URL, now, now, chat.Active, now,
		)
		if err != nil {
			return fmt.Errorf("upsert exec: %w", err)
		}
		return nil
	})
}
