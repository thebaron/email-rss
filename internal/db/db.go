package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type ProcessedMessage struct {
	ID          int64
	Folder      string
	UID         uint32
	Subject     string
	From        string
	Date        time.Time
	ProcessedAt time.Time
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS processed_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		folder TEXT NOT NULL,
		uid INTEGER NOT NULL,
		subject TEXT,
		from_addr TEXT,
		date DATETIME,
		processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(folder, uid)
	);

	CREATE INDEX IF NOT EXISTS idx_folder_uid ON processed_messages(folder, uid);
	CREATE INDEX IF NOT EXISTS idx_folder_date ON processed_messages(folder, date);
	`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) IsMessageProcessed(folder string, uid uint32) (bool, error) {
	query := `SELECT COUNT(*) FROM processed_messages WHERE folder = ? AND uid = ?`

	var count int
	err := db.conn.QueryRow(query, folder, uid).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if message is processed: %v", err)
	}

	return count > 0, nil
}

func (db *DB) MarkMessageProcessed(folder string, uid uint32, subject, from string, date time.Time) error {
	query := `
	INSERT OR REPLACE INTO processed_messages (folder, uid, subject, from_addr, date)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, folder, uid, subject, from, date)
	if err != nil {
		return fmt.Errorf("failed to mark message as processed: %v", err)
	}

	return nil
}

func (db *DB) GetProcessedMessages(folder string, limit int) ([]ProcessedMessage, error) {
	query := `
	SELECT id, folder, uid, subject, from_addr, date, processed_at
	FROM processed_messages
	WHERE folder = ?
	ORDER BY date DESC
	LIMIT ?
	`

	rows, err := db.conn.Query(query, folder, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages: %v", err)
	}
	defer rows.Close()

	var messages []ProcessedMessage
	for rows.Next() {
		var msg ProcessedMessage
		err := rows.Scan(&msg.ID, &msg.Folder, &msg.UID, &msg.Subject, &msg.From, &msg.Date, &msg.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *DB) ClearFolderHistory(folder string) error {
	query := `DELETE FROM processed_messages WHERE folder = ?`

	_, err := db.conn.Exec(query, folder)
	if err != nil {
		return fmt.Errorf("failed to clear folder history: %v", err)
	}

	return nil
}

func (db *DB) GetLastProcessedDate(folder string) (time.Time, error) {
	query := `SELECT MAX(date) FROM processed_messages WHERE folder = ?`

	var lastDateStr sql.NullString
	err := db.conn.QueryRow(query, folder).Scan(&lastDateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last processed date: %v", err)
	}

	if !lastDateStr.Valid || lastDateStr.String == "" {
		return time.Time{}, nil
	}

	lastDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", lastDateStr.String)
	if err != nil {
		lastDate, err = time.Parse(time.RFC3339, lastDateStr.String)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse last processed date: %v", err)
		}
	}

	return lastDate, nil
}
