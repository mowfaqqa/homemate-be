package waitlist

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	store := &PostgresStore{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS waitlist_entries (
			id BIGSERIAL PRIMARY KEY,
			first_name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			city TEXT NOT NULL,
			assistant_interest BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS waitlist_entries_created_at_idx
			ON waitlist_entries (created_at DESC);
		CREATE INDEX IF NOT EXISTS waitlist_entries_assistant_interest_idx
			ON waitlist_entries (assistant_interest);
	`

	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate waitlist database: %w", err)
	}
	return nil
}

func (s *PostgresStore) Add(ctx context.Context, entry Entry) error {
	const query = `
		INSERT INTO waitlist_entries (first_name, email, city, assistant_interest)
		VALUES ($1, $2, $3, $4)
	`
	_, err := s.pool.Exec(ctx, query, entry.FirstName, entry.Email, entry.City, entry.AssistantInterest)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicateEmail
	}
	return fmt.Errorf("insert waitlist entry: %w", err)
}

func (s *PostgresStore) List(ctx context.Context, filters Filters) ([]Entry, error) {
	query := `
		SELECT id, first_name, email, city, assistant_interest, created_at
		FROM waitlist_entries
		WHERE 1 = 1
	`
	args := make([]any, 0, 2)

	if search := strings.TrimSpace(filters.Search); search != "" {
		args = append(args, "%"+search+"%")
		query += fmt.Sprintf(" AND (first_name ILIKE $%d OR email ILIKE $%d OR city ILIKE $%d)", len(args), len(args), len(args))
	}
	if filters.AssistantInterest != nil {
		args = append(args, *filters.AssistantInterest)
		query += fmt.Sprintf(" AND assistant_interest = $%d", len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query waitlist entries: %w", err)
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(
			&entry.ID,
			&entry.FirstName,
			&entry.Email,
			&entry.City,
			&entry.AssistantInterest,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("read waitlist entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read waitlist entries: %w", err)
	}
	return entries, nil
}
