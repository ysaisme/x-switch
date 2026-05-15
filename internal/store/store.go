package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type RequestLog struct {
	ID           int64     `json:"id"`
	RequestID    string    `json:"request_id"`
	Timestamp    time.Time `json:"timestamp"`
	SiteID       string    `json:"site_id"`
	Model        string    `json:"model"`
	Protocol     string    `json:"protocol"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	LatencyMs    int       `json:"latency_ms"`
	StatusCode   int       `json:"status_code"`
	IsStream     bool      `json:"is_stream"`
	Error        string    `json:"error,omitempty"`
	ClientIP     string    `json:"client_ip"`
	Cost         float64   `json:"cost"`
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	dbPath := filepath.Join(dir, "mswitch.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS request_logs (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id    TEXT,
			timestamp     DATETIME DEFAULT CURRENT_TIMESTAMP,
			site_id       TEXT,
			model         TEXT,
			protocol      TEXT,
			input_tokens  INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			latency_ms    INTEGER DEFAULT 0,
			status_code   INTEGER DEFAULT 0,
			is_stream     BOOLEAN DEFAULT FALSE,
			error         TEXT,
			client_ip     TEXT,
			cost          REAL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON request_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_logs_site_id ON request_logs(site_id);
		CREATE INDEX IF NOT EXISTS idx_logs_model ON request_logs(model);
	`)

	return err
}

func (s *Store) InsertLog(log *RequestLog) error {
	_, err := s.db.Exec(`
		INSERT INTO request_logs (request_id, timestamp, site_id, model, protocol,
			input_tokens, output_tokens, latency_ms, status_code, is_stream,
			error, client_ip, cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RequestID, log.Timestamp, log.SiteID, log.Model, log.Protocol,
		log.InputTokens, log.OutputTokens, log.LatencyMs, log.StatusCode,
		log.IsStream, log.Error, log.ClientIP, log.Cost,
	)
	return err
}

func (s *Store) QueryLogs(filter LogFilter) ([]RequestLog, error) {
	query := "SELECT id, request_id, timestamp, site_id, model, protocol, input_tokens, output_tokens, latency_ms, status_code, is_stream, error, client_ip, cost FROM request_logs WHERE 1=1"
	args := []interface{}{}

	if filter.SiteID != "" {
		query += " AND site_id = ?"
		args = append(args, filter.SiteID)
	}
	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.StatusCode > 0 {
		query += " AND status_code = ?"
		args = append(args, filter.StatusCode)
	}
	if filter.OnlyErrors {
		query += " AND error != ''"
	}
	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}

	query += " ORDER BY id DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []RequestLog{}
	for rows.Next() {
		var l RequestLog
		var ts string
		err := rows.Scan(&l.ID, &l.RequestID, &ts, &l.SiteID, &l.Model, &l.Protocol,
			&l.InputTokens, &l.OutputTokens, &l.LatencyMs, &l.StatusCode,
			&l.IsStream, &l.Error, &l.ClientIP, &l.Cost)
		if err != nil {
			continue
		}
		l.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		logs = append(logs, l)
	}

	return logs, nil
}

func (s *Store) CleanOldLogs(maxDays int) error {
	if maxDays <= 0 {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM request_logs WHERE timestamp < datetime('now', ?)",
		fmt.Sprintf("-%d days", maxDays))
	return err
}

func (s *Store) GetStats(since time.Time) (*Stats, error) {
	stats := &Stats{}

	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cost), 0)
		FROM request_logs WHERE timestamp >= ?`, since).Scan(
		&stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCost)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

type LogFilter struct {
	SiteID     string
	Model      string
	StatusCode int
	OnlyErrors bool
	Since      time.Time
	Limit      int
}

type Stats struct {
	TotalRequests    int64   `json:"total_requests"`
	TotalInputTokens int64   `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
	TotalCost        float64 `json:"total_cost"`
}
