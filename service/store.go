package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists links. Create returns (created, storedURL, err).
type Store interface {
	Create(ctx context.Context, code, url string) (bool, string, error)
	Get(ctx context.Context, code string) (string, bool, error)
}

type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS links (
		code TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT now())`)
	return err
}

func (s *PGStore) Create(ctx context.Context, code, url string) (bool, string, error) {
	tag, err := s.pool.Exec(ctx, `INSERT INTO links (code, url) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING`, code, url)
	if err != nil {
		return false, "", err
	}
	if tag.RowsAffected() == 1 {
		return true, url, nil
	}
	existing, ok, err := s.Get(ctx, code)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "", errors.New("row vanished after conflict")
	}
	return false, existing, nil
}

func (s *PGStore) Get(ctx context.Context, code string) (string, bool, error) {
	var url string
	err := s.pool.QueryRow(ctx, `SELECT url FROM links WHERE code = $1`, code).Scan(&url)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return url, true, nil
}
