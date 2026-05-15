package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Context is compact memory sent to the language model.
type Context struct {
	Summary string
	Facts   map[string]string
	Recent  []Turn
}

// Turn is one conversation message.
type Turn struct {
	ID        int64
	SessionID string
	Role      string
	Language  string
	Text      string
	CreatedAt time.Time
}

// Store persists conversation memory in SQLite.
type Store struct {
	db *sql.DB
}

// Open initializes the SQLite store.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// LoadContext loads compact memory for the next assistant turn.
func (s *Store) LoadContext(ctx context.Context, _ string) (Context, error) {
	summary, err := s.latestSummary(ctx)
	if err != nil {
		return Context{}, err
	}
	facts, err := s.facts(ctx)
	if err != nil {
		return Context{}, err
	}
	recent, err := s.recentTurns(ctx, 12)
	if err != nil {
		return Context{}, err
	}
	return Context{Summary: summary, Facts: facts, Recent: recent}, nil
}

// SaveTurn stores a user or assistant message.
func (s *Store) SaveTurn(ctx context.Context, sessionID string, role string, language string, text string) error {
	if sessionID == "" {
		return errors.New("session id is required")
	}
	if role == "" {
		return errors.New("role is required")
	}
	if strings.TrimSpace(text) == "" {
		return errors.New("text is required")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO conversation_turns(session_id, role, language, text, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, role, language, text, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("save turn: %w", err)
	}
	return nil
}

// UpdateSummary keeps a small rolling summary for future prompts.
func (s *Store) UpdateSummary(ctx context.Context, sessionID string) error {
	turns, err := s.sessionTurns(ctx, sessionID, 20)
	if err != nil {
		return err
	}
	if len(turns) == 0 {
		return nil
	}

	var builder strings.Builder
	builder.WriteString("Recent conversation summary:\n")
	for _, turn := range turns {
		line := strings.TrimSpace(turn.Text)
		if len(line) > 220 {
			line = line[:220] + "..."
		}
		builder.WriteString("- ")
		builder.WriteString(turn.Role)
		builder.WriteString(": ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO conversation_summaries(summary, updated_at)
		VALUES (?, ?)
	`, builder.String(), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update summary: %w", err)
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS conversation_turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			language TEXT NOT NULL,
			text TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS conversation_turns_created_at_idx
			ON conversation_turns(created_at)`,
		`CREATE TABLE IF NOT EXISTS conversation_summaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			summary TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS assistant_facts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	return nil
}

func (s *Store) latestSummary(ctx context.Context) (string, error) {
	var summary string
	err := s.db.QueryRowContext(ctx, `
		SELECT summary FROM conversation_summaries
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`).Scan(&summary)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load summary: %w", err)
	}
	return summary, nil
}

func (s *Store) facts(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM assistant_facts ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("load facts: %w", err)
	}
	defer rows.Close()

	facts := map[string]string{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		facts[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate facts: %w", err)
	}
	return facts, nil
}

func (s *Store) recentTurns(ctx context.Context, limit int) ([]Turn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, role, language, text, created_at
		FROM conversation_turns
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("load recent turns: %w", err)
	}
	defer rows.Close()
	return scanTurns(rows)
}

func (s *Store) sessionTurns(ctx context.Context, sessionID string, limit int) ([]Turn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, role, language, text, created_at
		FROM conversation_turns
		WHERE session_id = ?
		ORDER BY created_at ASC, id ASC
		LIMIT ?
	`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("load session turns: %w", err)
	}
	defer rows.Close()
	return scanTurns(rows)
}

func scanTurns(rows *sql.Rows) ([]Turn, error) {
	var turns []Turn
	for rows.Next() {
		var turn Turn
		if err := rows.Scan(&turn.ID, &turn.SessionID, &turn.Role, &turn.Language, &turn.Text, &turn.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		turns = append(turns, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turns: %w", err)
	}
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	return turns, nil
}
