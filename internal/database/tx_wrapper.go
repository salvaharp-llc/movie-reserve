package database

import (
	"context"
	"database/sql"
	"fmt"
)

type DbStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *DbStore {
	return &DbStore{
		Queries: New(db),
		db:      db,
	}
}

func (store *DbStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := store.WithTx(tx)
	err = fn(q)

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
