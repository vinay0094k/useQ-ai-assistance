package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/yourusername/useq-ai-assistant/models"
)

// SQLiteDB represents a SQLite database connection
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// CodeFile represents a code file in the database
type CodeFile struct {
	ID           int64     `json:"id"`
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Extension    string    `json:"extension"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	Language     string    `json:"language"`
	Content      string    `json:"content"`
	LastModified time.Time `json:"last_modified"`
	LastIndexed  time.Time `json:"last_indexed"`
	Metadata     string    `json:"metadata"` // JSON
}

// CodeFunction represents a function in the database
type CodeFunction struct {
	ID          int64     `json:"id"`
	FileID      int64     `json:"file_id"`
	Name        string    `json:"name"`
	Signature   string    `json:"signature"`
	StartLine   int       `json:"start_line"`
	EndLine     int       `json:"end_line"`
	Visibility  string    `json:"visibility"`
	Type        string    `json:"type"`       // function, method, constructor
	Parameters  string    `json:"parameters"` // JSON
	ReturnType  string    `json:"return_type"`
	DocString   string    `json:"doc_string"`
	Complexity  int       `json:"complexity"`
	LastIndexed time.Time `json:"last_indexed"`
}

// CodeType represents a type/struct/interface in the database
type CodeType struct {
	ID          int64     `json:"id"`
	FileID      int64     `json:"file_id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"` // struct, interface, type
	StartLine   int       `json:"start_line"`
	EndLine     int       `json:"end_line"`
	Fields      string    `json:"fields"`  // JSON
	Methods     string    `json:"methods"` // JSON
	DocString   string    `json:"doc_string"`
	LastIndexed time.Time `json:"last_indexed"`
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	sqliteDB := &SQLiteDB{
		db:   db,
		path: dbPath,
	}

	// Initialize schema
	if err := sqliteDB.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return sqliteDB, nil
}

// initSchema creates the database schema
func (db *SQLiteDB) initSchema() error {
	schema := `
    -- Files table
    CREATE TABLE IF NOT EXISTS files (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        path TEXT UNIQUE NOT NULL,
        name TEXT NOT NULL,
        extension TEXT NOT NULL,
        size INTEGER NOT NULL,
        hash TEXT NOT NULL,
        language TEXT NOT NULL,
        content TEXT,
        last_modified DATETIME NOT NULL,
        last_indexed DATETIME NOT NULL,
        metadata TEXT DEFAULT '{}',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    -- Functions table
    CREATE TABLE IF NOT EXISTS functions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        file_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        signature TEXT NOT NULL,
        start_line INTEGER NOT NULL,
        end_line INTEGER NOT NULL,
        visibility TEXT DEFAULT 'public',
        type TEXT DEFAULT 'function',
        parameters TEXT DEFAULT '[]',
        return_type TEXT DEFAULT '',
        doc_string TEXT DEFAULT '',
        complexity INTEGER DEFAULT 0,
        last_indexed DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
    );

    -- Types table
    CREATE TABLE IF NOT EXISTS types (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        file_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        kind TEXT NOT NULL,
        start_line INTEGER NOT NULL,
        end_line INTEGER NOT NULL,
        fields TEXT DEFAULT '[]',
        methods TEXT DEFAULT '[]',
        doc_string TEXT DEFAULT '',
        last_indexed DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
    );

    -- Sessions table
    CREATE TABLE IF NOT EXISTS sessions (
        id TEXT PRIMARY KEY,
        data TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    -- Query history table
    CREATE TABLE IF NOT EXISTS query_history (
        id TEXT PRIMARY KEY,
        session_id TEXT NOT NULL,
        query_data TEXT NOT NULL,
        response_data TEXT NOT NULL,
        tokens_used INTEGER NOT NULL,
        cost REAL NOT NULL,
        provider TEXT NOT NULL,
        agent TEXT NOT NULL,
        success BOOLEAN NOT NULL,
        duration_ms INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );

    -- Queries table (for models.Query)
    CREATE TABLE IF NOT EXISTS queries (
        id TEXT PRIMARY KEY,
        user_input TEXT NOT NULL,
        language TEXT,
        context TEXT,
        timestamp DATETIME NOT NULL,
        session_id TEXT
    );

    -- Responses table (for models.Response)
    CREATE TABLE IF NOT EXISTS responses (
        id TEXT PRIMARY KEY,
        query_id TEXT NOT NULL,
        type TEXT NOT NULL,
        content TEXT NOT NULL,
        metadata TEXT,
        agent_used TEXT,
        timestamp DATETIME NOT NULL,
        token_usage TEXT,
        cost TEXT,
        FOREIGN KEY (query_id) REFERENCES queries(id)
    );

    -- Code files table (for vector indexing)
    CREATE TABLE IF NOT EXISTS code_files (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        path TEXT UNIQUE NOT NULL,
        name TEXT NOT NULL,
        extension TEXT,
        size INTEGER,
        hash TEXT,
        language TEXT,
        indexed_at DATETIME,
        last_modified DATETIME
    );

    -- User feedback table
    CREATE TABLE IF NOT EXISTS feedback (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        query_id TEXT NOT NULL,
        session_id TEXT NOT NULL,
        rating INTEGER,
        helpful BOOLEAN,
        accurate BOOLEAN,
        complete BOOLEAN,
        comments TEXT,
        corrections TEXT, -- JSON
        feedback_type TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (query_id) REFERENCES query_history(id) ON DELETE CASCADE
    );

    -- Token usage table
    CREATE TABLE IF NOT EXISTS token_usage (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL,
        query_id TEXT NOT NULL,
        provider TEXT NOT NULL,
        model TEXT NOT NULL,
        input_tokens INTEGER NOT NULL,
        output_tokens INTEGER NOT NULL,
        total_tokens INTEGER NOT NULL,
        input_cost REAL NOT NULL,
        output_cost REAL NOT NULL,
        total_cost REAL NOT NULL,
        timestamp DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    -- Learning patterns table
    CREATE TABLE IF NOT EXISTS learning_patterns (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL,
        pattern_type TEXT NOT NULL, -- successful, correction
        input_pattern TEXT NOT NULL,
        expected_output TEXT NOT NULL,
        context TEXT NOT NULL,
        confidence REAL NOT NULL,
        usage_count INTEGER DEFAULT 1,
        success_rate REAL DEFAULT 0.0,
        last_used DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    -- Create indexes for better performance
    CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
    CREATE INDEX IF NOT EXISTS idx_files_extension ON files(extension);
    CREATE INDEX IF NOT EXISTS idx_files_last_modified ON files(last_modified);
    CREATE INDEX IF NOT EXISTS idx_functions_file_id ON functions(file_id);
    CREATE INDEX IF NOT EXISTS idx_functions_name ON functions(name);
    CREATE INDEX IF NOT EXISTS idx_types_file_id ON types(file_id);
    CREATE INDEX IF NOT EXISTS idx_types_name ON types(name);
    CREATE INDEX IF NOT EXISTS idx_query_history_session_id ON query_history(session_id);
    CREATE INDEX IF NOT EXISTS idx_token_usage_session_id ON token_usage(session_id);
    CREATE INDEX IF NOT EXISTS idx_learning_patterns_session_id ON learning_patterns(session_id);
    CREATE INDEX IF NOT EXISTS idx_feedback_query_id ON feedback(query_id);

    -- Create triggers for updated_at
    CREATE TRIGGER IF NOT EXISTS update_files_updated_at
        AFTER UPDATE ON files
        BEGIN
            UPDATE files SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;

    CREATE TRIGGER IF NOT EXISTS update_functions_updated_at
        AFTER UPDATE ON functions
        BEGIN
            UPDATE functions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;

    CREATE TRIGGER IF NOT EXISTS update_types_updated_at
        AFTER UPDATE ON types
        BEGIN
            UPDATE types SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;

    CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at
        AFTER UPDATE ON sessions
        BEGIN
            UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;
    `

	_, err := db.db.Exec(schema)
	return err
}

// File operations

// SaveFile saves or updates a code file
func (db *SQLiteDB) SaveFile(file *CodeFile) error {
	query := `
    INSERT OR REPLACE INTO files 
    (path, name, extension, size, hash, language, content, last_modified, last_indexed, metadata)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		file.Path, file.Name, file.Extension, file.Size, file.Hash,
		file.Language, file.Content, file.LastModified, file.LastIndexed, file.Metadata)

	return err
}

// GetFile retrieves a file by path
func (db *SQLiteDB) GetFile(path string) (*CodeFile, error) {
	query := `SELECT id, path, name, extension, size, hash, language, content, 
              last_modified, last_indexed, metadata FROM files WHERE path = ?`

	var file CodeFile
	err := db.db.QueryRow(query, path).Scan(
		&file.ID, &file.Path, &file.Name, &file.Extension, &file.Size,
		&file.Hash, &file.Language, &file.Content, &file.LastModified,
		&file.LastIndexed, &file.Metadata)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetFilesByExtension retrieves files by extension
func (db *SQLiteDB) GetFilesByExtension(extension string) ([]*CodeFile, error) {
	query := `SELECT id, path, name, extension, size, hash, language, content,
              last_modified, last_indexed, metadata FROM files WHERE extension = ?`

	rows, err := db.db.Query(query, extension)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*CodeFile
	for rows.Next() {
		var file CodeFile
		err := rows.Scan(
			&file.ID, &file.Path, &file.Name, &file.Extension, &file.Size,
			&file.Hash, &file.Language, &file.Content, &file.LastModified,
			&file.LastIndexed, &file.Metadata)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}

	return files, nil
}

// Function operations

// SaveFunction saves or updates a function
func (db *SQLiteDB) SaveFunction(function *CodeFunction) error {
	// This method expects FileID to be already set correctly
	query := `
    INSERT OR REPLACE INTO functions 
    (file_id, name, signature, start_line, end_line, visibility, type, parameters, return_type, doc_string, complexity, last_indexed)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(query,
		function.FileID, function.Name, function.Signature, function.StartLine, function.EndLine,
		function.Visibility, function.Type, function.Parameters, function.ReturnType,
		function.DocString, function.Complexity, time.Now())

	return err
}

// SaveFunctionForFile saves a function with file path resolution
func (db *SQLiteDB) SaveFunctionForFile(function *CodeFunction, filePath string) error {
	// First get the file ID
	fileID, err := db.getFileIDByPath(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file ID for %s: %w", filePath, err)
	}

	function.FileID = fileID
	return db.SaveFunction(function)
}

// GetFunctionsByFile retrieves functions for a specific file
func (db *SQLiteDB) GetFunctionsByFile(filePath string) ([]*CodeFunction, error) {
	query := `
    SELECT f.id, f.file_id, f.name, f.signature, f.start_line, f.end_line,
           f.visibility, f.type, f.parameters, f.return_type, f.doc_string, f.complexity, f.last_indexed
    FROM functions f
    JOIN files fl ON f.file_id = fl.id
    WHERE fl.path = ?`

	rows, err := db.db.Query(query, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*CodeFunction
	for rows.Next() {
		var function CodeFunction
		err := rows.Scan(
			&function.ID, &function.FileID, &function.Name, &function.Signature,
			&function.StartLine, &function.EndLine, &function.Visibility, &function.Type,
			&function.Parameters, &function.ReturnType, &function.DocString,
			&function.Complexity, &function.LastIndexed)
		if err != nil {
			return nil, err
		}
		functions = append(functions, &function)
	}

	return functions, nil
}

// SearchFunctions searches for functions by name pattern
func (db *SQLiteDB) SearchFunctions(namePattern string) ([]*CodeFunction, error) {
	query := `
    SELECT f.id, f.file_id, f.name, f.signature, f.start_line, f.end_line,
           f.visibility, f.type, f.parameters, f.return_type, f.doc_string, f.complexity, f.last_indexed
    FROM functions f
    WHERE f.name LIKE ?
    ORDER BY f.name`

	pattern := "%" + namePattern + "%"
	rows, err := db.db.Query(query, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*CodeFunction
	for rows.Next() {
		var function CodeFunction
		err := rows.Scan(
			&function.ID, &function.FileID, &function.Name, &function.Signature,
			&function.StartLine, &function.EndLine, &function.Visibility, &function.Type,
			&function.Parameters, &function.ReturnType, &function.DocString,
			&function.Complexity, &function.LastIndexed)
		if err != nil {
			return nil, err
		}
		functions = append(functions, &function)
	}

	return functions, nil
}

// Session operations

// SaveSession saves session data
func (db *SQLiteDB) SaveSession(sessionID string, data []byte) error {
	query := `INSERT OR REPLACE INTO sessions (id, data, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := db.db.Exec(query, sessionID, string(data))
	return err
}

// LoadSession loads session data
func (db *SQLiteDB) LoadSession(sessionID string) ([]byte, error) {
	query := `SELECT data FROM sessions WHERE id = ?`
	var data string
	err := db.db.QueryRow(query, sessionID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	return []byte(data), nil
}

// SaveQuery saves a query to history
func (db *SQLiteDB) SaveQuery(query *models.Query, response *models.Response) error {
	queryData, _ := json.Marshal(query)
	responseData, _ := json.Marshal(response)

	sqlQuery := `
    INSERT INTO query_history 
    (id, session_id, query_data, response_data, tokens_used, cost, provider, agent, success, duration_ms)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.db.Exec(sqlQuery,
		query.ID, query.SessionID, string(queryData), string(responseData),
		response.TokenUsage.TotalTokens, response.Cost.TotalCost,
		response.Provider, response.AgentUsed,
		response.Type != models.ResponseTypeError,
		response.Metadata.GenerationTime.Milliseconds())

	return err
}

// SaveTokenUsage saves token usage data
func (db *SQLiteDB) SaveTokenUsage(usage *models.TokenUsage) error {
	query := `
    INSERT INTO token_usage 
    (session_id, query_id, provider, model, input_tokens, output_tokens, total_tokens, 
     input_cost, output_cost, total_cost, timestamp)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Note: SessionID and QueryID would need to be added to TokenUsage model
	_, err := db.db.Exec(query,
		"", "", usage.Provider, usage.Model,
		usage.InputTokens, usage.OutputTokens, usage.TotalTokens,
		0.0, 0.0, 0.0, usage.Timestamp) // Cost calculation would need to be added

	return err
}

// GetTokenUsageStats returns token usage statistics
func (db *SQLiteDB) GetTokenUsageStats(sessionID string, period time.Duration) (*models.TokenMetrics, error) {
	query := `
    SELECT 
        COUNT(*) as query_count,
        SUM(total_tokens) as total_tokens,
        SUM(total_cost) as total_cost,
        AVG(total_tokens) as avg_tokens,
        AVG(total_cost) as avg_cost
    FROM token_usage 
    WHERE session_id = ? AND timestamp >= ?`

	since := time.Now().Add(-period)

	var metrics models.TokenMetrics
	err := db.db.QueryRow(query, sessionID, since).Scan(
		&metrics.TotalQueries, &metrics.TotalTokens, &metrics.TotalCost,
		&metrics.AverageTokensPerQuery, &metrics.AverageCostPerQuery)

	if err != nil {
		return nil, err
	}

	metrics.Period = models.PeriodDaily // Set based on period parameter
	metrics.StartDate = since
	metrics.EndDate = time.Now()

	return &metrics, nil
}

// Cleanup operations

// DeleteOldSessions removes sessions older than the specified duration
func (db *SQLiteDB) DeleteOldSessions(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM sessions WHERE updated_at < ?`
	_, err := db.db.Exec(query, cutoff)
	return err
}

// DeleteFile removes a file and all associated data
func (db *SQLiteDB) DeleteFile(path string) error {
	query := `DELETE FROM files WHERE path = ?`
	_, err := db.db.Exec(query, path)
	return err
}

// Helper methods

// getFileIDByPath gets the file ID for a given path
func (db *SQLiteDB) getFileIDByPath(pathOrID interface{}) (int64, error) {
	// If it's already an int64, return it
	if id, ok := pathOrID.(int64); ok {
		return id, nil
	}

	// Otherwise, treat as path and look it up
	path, ok := pathOrID.(string)
	if !ok {
		return 0, fmt.Errorf("invalid file identifier")
	}

	query := `SELECT id FROM files WHERE path = ?`
	var id int64
	err := db.db.QueryRow(query, path).Scan(&id)
	return id, err
}

// GetBasicStats returns basic database statistics
func (db *SQLiteDB) GetBasicStats() (map[string]int, error) {
	stats := make(map[string]int)

	tables := []string{"files", "functions", "types", "sessions", "query_history"}

	for _, table := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		var count int
		if err := db.db.QueryRow(query).Scan(&count); err != nil {
			return nil, err
		}
		stats[table] = count
	}

	return stats, nil
}

// GetIndexedFiles returns list of all indexed files
func (db *SQLiteDB) GetIndexedFiles() ([]string, error) {
	fmt.Printf("📁 [DB] GetIndexedFiles called\n")
	query := `SELECT path FROM files ORDER BY path`
	rows, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		files = append(files, path)
	}

	fmt.Printf("📁 [DB] Found %d indexed files\n", len(files))
	return files, nil
}

// Close closes the database connection
func (db *SQLiteDB) Close() error {
	return db.db.Close()
}

// Vacuum optimizes the database
func (db *SQLiteDB) Vacuum() error {
	_, err := db.db.Exec("VACUUM")
	return err
}

// SaveCodeChunk saves a code chunk (alias for SaveFile for compatibility)
func (db *SQLiteDB) SaveCodeChunk(chunk *CodeFile) error {
	return db.SaveFile(chunk)
}

// SaveCodeFunction saves a code function (alias for SaveFunction for compatibility)  
func (db *SQLiteDB) SaveCodeFunction(function *CodeFunction) error {
	return db.SaveFunction(function)
}

// StoreQuery stores a query and its metadata
func (db *SQLiteDB) StoreQuery(query *models.Query) error {
	contextJSON, _ := json.Marshal(query.Context)
	
	_, err := db.db.Exec(`
		INSERT INTO queries (id, user_input, language, context, timestamp, session_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, query.ID, query.UserInput, query.Language, string(contextJSON), query.Timestamp, query.SessionID)
	
	return err
}

// StoreResponse stores a response and its metadata
func (db *SQLiteDB) StoreResponse(response *models.Response) error {
	contentJSON, _ := json.Marshal(response.Content)
	metadataJSON, _ := json.Marshal(response.Metadata)
	tokenUsageJSON, _ := json.Marshal(response.TokenUsage)
	costJSON, _ := json.Marshal(response.Cost)
	
	_, err := db.db.Exec(`
		INSERT INTO responses (id, query_id, type, content, metadata, agent_used, timestamp, token_usage, cost)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, response.ID, response.QueryID, string(response.Type), string(contentJSON), 
		string(metadataJSON), response.AgentUsed, response.Timestamp, 
		string(tokenUsageJSON), string(costJSON))
	
	return err
}

// GetQueryHistory retrieves recent queries
func (db *SQLiteDB) GetQueryHistory(limit int) ([]*models.Query, error) {
	rows, err := db.db.Query(`
		SELECT id, user_input, language, context, timestamp, session_id
		FROM queries ORDER BY timestamp DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var queries []*models.Query
	for rows.Next() {
		query := &models.Query{}
		var contextJSON string
		err := rows.Scan(&query.ID, &query.UserInput, &query.Language, 
			&contextJSON, &query.Timestamp, &query.SessionID)
		if err != nil {
			return nil, err
		}
		
		if contextJSON != "" {
			json.Unmarshal([]byte(contextJSON), &query.Context)
		}
		
		queries = append(queries, query)
	}
	
	return queries, nil
}

// GetStats returns database statistics
func (db *SQLiteDB) GetStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}
	
	// Count files
	err := db.db.QueryRow("SELECT COUNT(*) FROM code_files").Scan(&stats.TotalFiles)
	if err != nil {
		return nil, err
	}
	
	// Count queries
	err = db.db.QueryRow("SELECT COUNT(*) FROM queries").Scan(&stats.TotalQueries)
	if err != nil {
		return nil, err
	}
	
	// Count responses
	err = db.db.QueryRow("SELECT COUNT(*) FROM responses").Scan(&stats.TotalResponses)
	if err != nil {
		return nil, err
	}
	
	// Get languages
	rows, err := db.db.Query("SELECT language, COUNT(*) FROM code_files GROUP BY language")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	stats.LanguageBreakdown = make(map[string]int)
	for rows.Next() {
		var language string
		var count int
		rows.Scan(&language, &count)
		stats.LanguageBreakdown[language] = count
	}
	
	return stats, nil
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	TotalFiles          int            `json:"total_files"`
	TotalQueries        int            `json:"total_queries"`
	TotalResponses      int            `json:"total_responses"`
	LanguageBreakdown   map[string]int `json:"language_breakdown"`
}
