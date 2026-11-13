package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Message represents a single message in a conversation
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	ThreadID  string    `json:"thread_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Thread represents a conversation thread
type Thread struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	MessageCount int
}

// Store manages conversation/message persistence with SQLite
type Store struct {
	db *sql.DB
}

// NewStore creates a new conversation store
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "conversation.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &Store{db: db}

	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables for conversations, messages, and threads
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		thread_id TEXT,
		meta_json TEXT
	);

	CREATE TABLE IF NOT EXISTS threads (
		id TEXT PRIMARY KEY,
		title TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		message_count INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
	CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);
	`

	_, err := s.db.Exec(schema)
	return err
}

// MessageWithMetadata represents a message with additional metadata for search
type MessageWithMetadata struct {
	Message
	ThreadTitle string
}

// CreateMessage creates a new message and returns its ID
func (s *Store) CreateMessage(ctx context.Context, msg *Message) (string, error) {
	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}

	msg.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (id, role, content, created_at, thread_id, meta_json)
		VALUES (?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.Role, msg.Content, msg.CreatedAt.Format(time.RFC3339), msg.ThreadID, s.encodeMeta(msg.Metadata))

	if err != nil {
		return "", fmt.Errorf("creating message: %w", err)
	}

	// Update thread message count
	if msg.ThreadID != "" {
		s.incrementThreadMessageCount(msg.ThreadID)
	}

	return msg.ID, nil
}

// GetMessage retrieves a message by ID
func (s *Store) GetMessage(ctx context.Context, id string) (*Message, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, role, content, created_at, thread_id, meta_json
		FROM messages WHERE id = ?
	`, id)

	var msg Message
	var metaJSON string
	err := row.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.CreatedAt, &msg.ThreadID, &metaJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if metaJSON != "" {
		if err := json.Unmarshal([]byte(metaJSON), &msg.Metadata); err != nil {
			msg.Metadata = make(map[string]string)
		}
	}

	return &msg, nil
}

// ListMessagesByThread retrieves all messages in a thread, ordered by creation time
func (s *Store) ListMessagesByThread(ctx context.Context, threadID string) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, content, created_at, thread_id, meta_json
		FROM messages WHERE thread_id = ?
		ORDER BY created_at ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var metaJSON string
		err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.CreatedAt, &msg.ThreadID, &metaJSON)
		if err != nil {
			return nil, err
		}

		if metaJSON != "" {
			if err := json.Unmarshal([]byte(metaJSON), &msg.Metadata); err != nil {
				msg.Metadata = make(map[string]string)
			}
		}
		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}

// ListThreads retrieves all threads
func (s *Store) ListThreads(ctx context.Context) ([]*Thread, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, created_at, updated_at, message_count
		FROM threads
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*Thread
	for rows.Next() {
		var t Thread
		var createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.Title, &createdAt, &updatedAt, &t.MessageCount); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		threads = append(threads, &t)
	}
	return threads, rows.Err()
}

// GetThread retrieves a thread by ID
func (s *Store) GetThread(ctx context.Context, id string) (*Thread, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, created_at, updated_at, message_count
		FROM threads WHERE id = ?
	`, id)

	var t Thread
	var createdAt, updatedAt string
	err := row.Scan(&t.ID, &t.Title, &createdAt, &updatedAt, &t.MessageCount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &t, nil
}

// SearchMessages searches messages by content, role, or thread
func (s *Store) SearchMessages(ctx context.Context, query string, threadID *string) ([]*MessageWithMetadata, error) {
	var sqlQuery string
	var args []interface{}

	if threadID != nil {
		sqlQuery = `SELECT m.id, m.role, m.content, m.created_at, m.thread_id, m.meta_json, t.title AS ThreadTitle
			FROM messages m
			LEFT JOIN threads t ON m.thread_id = t.id
			WHERE m.content LIKE ? AND m.thread_id = ?
		`
		args = append(args, "%"+query+"%")
		args = append(args, *threadID)
	} else {
		sqlQuery = `SELECT m.id, m.role, m.content, m.created_at, m.thread_id, m.meta_json, t.title AS ThreadTitle
			FROM messages m
			LEFT JOIN threads t ON m.thread_id = t.id
			WHERE m.content LIKE ?
		`
		args = append(args, "%"+query+"%")
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*MessageWithMetadata
	for rows.Next() {
		var id, role, content, createdAt, threadID, metaJSON, threadTitle string
		if err := rows.Scan(&id, &role, &content, &createdAt, &threadID, &metaJSON, &threadTitle); err != nil {
			return nil, err
		}

		var msg Message
		msg.ID = id
		msg.Role = role
		msg.Content = content

		if metaJSON != "" {
			if err := json.Unmarshal([]byte(metaJSON), &msg.Metadata); err != nil {
				msg.Metadata = make(map[string]string)
			}
		}

		msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		msg.ThreadID = threadID

		results = append(results, &MessageWithMetadata{
			Message:     msg,
			ThreadTitle: threadTitle,
		})
	}
	return results, rows.Err()
}

// DeleteMessage deletes a message by ID
func (s *Store) DeleteMessage(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		// We don't decrement thread count here to avoid complexity;
		// the count may become slightly stale but that's acceptable
	}

	return nil
}

// UpsertThread creates or updates a thread
func (s *Store) UpsertThread(ctx context.Context, thread *Thread) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO threads (id, title, created_at, updated_at, message_count)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = EXCLUDED.title,
			updated_at = EXCLUDED.updated_at,
			message_count = EXCLUDED.message_count
	`, thread.ID, thread.Title, now.Format(time.RFC3339), now.Format(time.RFC3339), thread.MessageCount)

	if err != nil {
		return fmt.Errorf("upserting thread: %w", err)
	}
	return nil
}

// encodeMeta encodes metadata map to JSON string
func (s *Store) encodeMeta(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	data, _ := json.Marshal(meta)
	return string(data)
}

// decodeMeta decodes JSON string to metadata map
func (s *Store) decodeMeta(metaJSON string) map[string]string {
	if metaJSON == "" {
		return make(map[string]string)
	}
	var meta map[string]string
	json.Unmarshal([]byte(metaJSON), &meta)
	return meta
}

// incrementThreadMessageCount increments the message count for a thread
func (s *Store) incrementThreadMessageCount(threadID string) {
	s.db.Exec(`UPDATE threads SET message_count = message_count + 1 WHERE id = ?`, threadID)
}