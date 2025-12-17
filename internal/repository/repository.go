package repository

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // SQLite driver for database/sql

	"ldapmerge/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Repository handles database operations.
type Repository struct {
	db     *sql.DB
	dbPath string
}

// New creates a new repository with the given database path.
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.ExecContext(context.Background(), "PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	repo := &Repository{db: db, dbPath: dbPath}

	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations.
func (r *Repository) migrate() error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	return goose.Up(r.db, "migrations")
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// DBInfo contains database information.
type DBInfo struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	SizeHuman    string `json:"size_human"`
	Version      string `json:"version"`
	Tables       int    `json:"tables"`
	WALMode      bool   `json:"wal_mode"`
	HistoryCount int64  `json:"history_count"`
	ConfigCount  int64  `json:"config_count"`
}

// GetDBInfo returns database information
func (r *Repository) GetDBInfo(ctx context.Context) (*DBInfo, error) {
	info := &DBInfo{
		Path: r.dbPath,
	}

	// Get SQLite version
	row := r.db.QueryRowContext(ctx, "SELECT sqlite_version()")
	if err := row.Scan(&info.Version); err != nil {
		info.Version = "unknown"
	}

	// Get journal mode (WAL or not)
	var journalMode string
	row = r.db.QueryRowContext(ctx, "PRAGMA journal_mode")
	if err := row.Scan(&journalMode); err == nil {
		info.WALMode = journalMode == "wal"
	}

	// Get table count
	row = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'goose_%'")
	if err := row.Scan(&info.Tables); err != nil {
		info.Tables = 0
	}

	// Get history count
	row = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM history")
	if err := row.Scan(&info.HistoryCount); err != nil {
		info.HistoryCount = 0
	}

	// Get config count
	row = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nsx_configs")
	if err := row.Scan(&info.ConfigCount); err != nil {
		info.ConfigCount = 0
	}

	// Get file size
	if fileInfo, err := os.Stat(r.dbPath); err == nil {
		info.Size = fileInfo.Size()
		info.SizeHuman = formatBytes(info.Size)
	}

	return info, nil
}

// formatBytes converts bytes to human readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// SaveHistory saves a merge operation to history
func (r *Repository) SaveHistory(ctx context.Context, initial []models.Domain, response models.CertificateResponse, result []models.Domain) (*models.HistoryEntry, error) {
	initialJSON, err := json.Marshal(initial)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial: %w", err)
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO history (initial, response, result) VALUES (?, ?, ?)`,
		string(initialJSON), string(responseJSON), string(resultJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert history: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return r.GetHistory(ctx, id)
}

// GetHistory retrieves a history entry by ID
func (r *Repository) GetHistory(ctx context.Context, id int64) (*models.HistoryEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, created_at, initial, response, result FROM history WHERE id = ?`, id)

	var entry models.HistoryEntry
	var initialStr, responseStr, resultStr string
	var createdAt string

	err := row.Scan(&entry.ID, &createdAt, &initialStr, &responseStr, &resultStr)
	if err != nil {
		return nil, err
	}

	entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	if err := json.Unmarshal([]byte(initialStr), &entry.Initial.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initial: %w", err)
	}
	if err := json.Unmarshal([]byte(responseStr), &entry.Response.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if err := json.Unmarshal([]byte(resultStr), &entry.Result.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &entry, nil
}

// ListHistory retrieves all history entries
func (r *Repository) ListHistory(ctx context.Context) ([]models.HistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, created_at, initial, response, result FROM history ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HistoryEntry
	for rows.Next() {
		var entry models.HistoryEntry
		var initialStr, responseStr, resultStr string
		var createdAt string

		err := rows.Scan(&entry.ID, &createdAt, &initialStr, &responseStr, &resultStr)
		if err != nil {
			return nil, err
		}

		entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		if err := json.Unmarshal([]byte(initialStr), &entry.Initial.Data); err != nil {
			continue
		}
		if err := json.Unmarshal([]byte(responseStr), &entry.Response.Data); err != nil {
			continue
		}
		if err := json.Unmarshal([]byte(resultStr), &entry.Result.Data); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// SaveConfig saves or updates an NSX configuration
func (r *Repository) SaveConfig(ctx context.Context, config *models.NSXConfig) (*models.NSXConfig, error) {
	now := time.Now()

	if config.ID == 0 {
		// Insert new config
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO nsx_configs (name, description, host, username, password, insecure, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			config.Name, config.Description, config.Host, config.Username, config.Password, config.Insecure, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert config: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}

		return r.GetConfig(ctx, id)
	}

	// Update existing config
	_, err := r.db.ExecContext(ctx,
		`UPDATE nsx_configs SET name=?, description=?, host=?, username=?, password=?, insecure=?, updated_at=? WHERE id=?`,
		config.Name, config.Description, config.Host, config.Username, config.Password, config.Insecure, now, config.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return r.GetConfig(ctx, config.ID)
}

// GetConfig retrieves an NSX configuration by ID
func (r *Repository) GetConfig(ctx context.Context, id int64) (*models.NSXConfig, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, host, username, password, insecure, created_at, updated_at
		 FROM nsx_configs WHERE id = ?`, id)

	var config models.NSXConfig
	var createdAt, updatedAt string
	var description, password sql.NullString

	err := row.Scan(&config.ID, &config.Name, &description, &config.Host, &config.Username, &password, &config.Insecure, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	config.Description = description.String
	config.Password = password.String
	config.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	config.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &config, nil
}

// ListConfigs retrieves all NSX configurations
func (r *Repository) ListConfigs(ctx context.Context) ([]models.NSXConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, host, username, insecure, created_at, updated_at
		 FROM nsx_configs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.NSXConfig
	for rows.Next() {
		var config models.NSXConfig
		var createdAt, updatedAt string
		var description sql.NullString

		err := rows.Scan(&config.ID, &config.Name, &description, &config.Host, &config.Username, &config.Insecure, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		config.Description = description.String
		// Don't return password in list
		config.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		config.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		configs = append(configs, config)
	}

	return configs, rows.Err()
}

// DeleteConfig deletes an NSX configuration by ID
func (r *Repository) DeleteConfig(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM nsx_configs WHERE id = ?`, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetConfigByName retrieves an NSX configuration by name
func (r *Repository) GetConfigByName(ctx context.Context, name string) (*models.NSXConfig, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, host, username, password, insecure, created_at, updated_at
		 FROM nsx_configs WHERE name = ?`, name)

	var config models.NSXConfig
	var createdAt, updatedAt string
	var description, password sql.NullString

	err := row.Scan(&config.ID, &config.Name, &description, &config.Host, &config.Username, &password, &config.Insecure, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	config.Description = description.String
	config.Password = password.String
	config.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	config.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &config, nil
}
