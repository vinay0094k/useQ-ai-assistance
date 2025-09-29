package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/logger"
	"github.com/yourusername/useq-ai-assistant/models"

	_ "github.com/mattn/go-sqlite3"
)

// =============================================================================
// SQLITE MCP SERVER - INTELLIGENT DATABASE OPERATIONS
// =============================================================================

// SQLiteServer provides AI access to SQLite database operations
type SQLiteServer struct {
	db                *sql.DB
	dbPath            string
	logger            *logger.StepLogger
	maxQueryTime      time.Duration
	maxRows           int
	allowedOperations []string
	blockedOperations []string
	readOnly          bool
}

// SQLiteConfig holds configuration for the SQLite server
type SQLiteConfig struct {
	DatabasePath      string        `json:"database_path"`
	MaxQueryTime      time.Duration `json:"max_query_time"`
	MaxRows           int           `json:"max_rows"`
	AllowedOperations []string      `json:"allowed_operations"`
	BlockedOperations []string      `json:"blocked_operations"`
	ReadOnly          bool          `json:"read_only"`
}

// NewSQLiteServer creates a new SQLite MCP server
func NewSQLiteServer(config SQLiteConfig, logger *logger.StepLogger) (*SQLiteServer, error) {
	// Set defaults
	if config.MaxQueryTime == 0 {
		config.MaxQueryTime = 30 * time.Second
	}
	if config.MaxRows == 0 {
		config.MaxRows = 1000
	}
	if len(config.AllowedOperations) == 0 {
		config.AllowedOperations = []string{"SELECT", "INSERT", "UPDATE", "CREATE", "PRAGMA"}
	}
	if len(config.BlockedOperations) == 0 {
		config.BlockedOperations = []string{"DELETE", "DROP", "ALTER"}
	}

	// Open database connection
	db, err := sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	server := &SQLiteServer{
		db:                db,
		dbPath:            config.DatabasePath,
		logger:            logger,
		maxQueryTime:      config.MaxQueryTime,
		maxRows:           config.MaxRows,
		allowedOperations: config.AllowedOperations,
		blockedOperations: config.BlockedOperations,
		readOnly:          config.ReadOnly,
	}

	logger.LogInfo("MCP", "SQLite MCP server initialized", map[string]interface{}{
		"database": config.DatabasePath,
		"readonly": config.ReadOnly,
	})
	return server, nil
}

// Close closes the database connection
func (ss *SQLiteServer) Close() error {
	if ss.db != nil {
		return ss.db.Close()
	}
	return nil
}

// =============================================================================
// INTELLIGENT COMMAND EXECUTION
// =============================================================================

// ExecuteCommand intelligently executes SQLite commands based on user intent
func (ss *SQLiteServer) ExecuteCommand(ctx context.Context, operation *models.MCPOperation) (*models.MCPResult, error) {
	ss.logger.LogInfo("MCP", "Executing SQLite command", map[string]interface{}{
		"type":   operation.Type,
		"params": operation.Parameters,
	})

	switch operation.Type {
	case models.MCPOperationSQLiteQuery:
		return ss.handleQuery(ctx, operation)
	case models.MCPOperationSQLiteSchema:
		return ss.handleSchemaInfo(ctx, operation)
	case models.MCPOperationSQLiteTables:
		return ss.handleTablesInfo(ctx, operation)
	default:
		return ss.createErrorResult(operation.ID, fmt.Errorf("unsupported operation: %s", operation.Type))
	}
}

// =============================================================================
// INTELLIGENT QUERY EXECUTION
// =============================================================================

// handleQuery executes intelligent database queries
func (ss *SQLiteServer) handleQuery(ctx context.Context, operation *models.MCPOperation) (*models.MCPResult, error) {
	startTime := time.Now()

	// Extract query parameters
	query, ok := operation.Parameters["query"].(string)
	if !ok {
		// Check if we have intent-based parameters instead of raw SQL
		intent, hasIntent := operation.Parameters["intent"].(string)
		keywords, _ := operation.Parameters["keywords"].([]interface{})

		if hasIntent {
			// Generate intelligent query based on intent
			generatedQuery, err := ss.generateIntelligentQuery(intent, interfaceToStringSlice(keywords))
			if err != nil {
				return ss.createErrorResult(operation.ID, fmt.Errorf("failed to generate query: %w", err))
			}
			query = generatedQuery
			ss.logger.LogInfo("MCP", "Generated intelligent query", map[string]interface{}{
				"intent": intent,
				"query":  query,
			})
		} else {
			return ss.createErrorResult(operation.ID, fmt.Errorf("query or intent parameter required"))
		}
	}

	// Security validation
	if err := ss.validateQuery(query); err != nil {
		return ss.createErrorResult(operation.ID, fmt.Errorf("query validation failed: %w", err))
	}

	ss.logger.LogInfo("MCP", "Executing database query", map[string]interface{}{"query": query})

	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, ss.maxQueryTime)
	defer cancel()

	rows, err := ss.db.QueryContext(queryCtx, query)
	if err != nil {
		return ss.createErrorResult(operation.ID, fmt.Errorf("query execution failed: %w", err))
	}
	defer rows.Close()

	// Process results intelligently
	results, err := ss.processQueryResults(rows)
	if err != nil {
		return ss.createErrorResult(operation.ID, fmt.Errorf("result processing failed: %w", err))
	}

	executionTime := time.Since(startTime)
	ss.logger.LogInfo("MCP", "Query executed successfully", map[string]interface{}{
		"rows_returned":  len(results),
		"execution_time": executionTime,
	})

	return &models.MCPResult{
		OperationID: operation.ID,
		Success:     true,
		Data: map[string]interface{}{
			"query":          query,
			"results":        results,
			"row_count":      len(results),
			"execution_time": executionTime.String(),
		},
		Duration: executionTime,
		Metadata: map[string]interface{}{
			"server":        "sqlite",
			"type":          operation.Type,
			"database_path": ss.dbPath,
			"query_type":    ss.detectQueryType(query),
		},
	}, nil
}

// generateIntelligentQuery generates SQL queries based on natural language intent
func (ss *SQLiteServer) generateIntelligentQuery(intent string, keywords []string) (string, error) {
	intentLower := strings.ToLower(intent)

	// Get table information for context
	tables, err := ss.getTableNames()
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	switch {
	case strings.Contains(intentLower, "find") || strings.Contains(intentLower, "search"):
		return ss.generateSearchQuery(keywords, tables)
	case strings.Contains(intentLower, "count"):
		return ss.generateCountQuery(keywords, tables)
	case strings.Contains(intentLower, "recent") || strings.Contains(intentLower, "latest"):
		return ss.generateRecentQuery(keywords, tables)
	case strings.Contains(intentLower, "popular") || strings.Contains(intentLower, "frequent"):
		return ss.generatePopularQuery(keywords, tables)
	case strings.Contains(intentLower, "schema") || strings.Contains(intentLower, "structure"):
		return ss.generateSchemaQuery(keywords, tables)
	default:
		return ss.generateGenericQuery(intent, keywords, tables)
	}
}

// generateSearchQuery generates intelligent search queries
func (ss *SQLiteServer) generateSearchQuery(keywords []string, tables []string) (string, error) {
	if len(keywords) == 0 {
		return "", fmt.Errorf("keywords required for search query")
	}

	// Find the most relevant table
	table := ss.findBestMatchingTable(keywords, tables)
	if table == "" {
		table = tables[0] // Fallback to first table
	}

	// Get table columns
	columns, err := ss.getTableColumns(table)
	if err != nil {
		return "", fmt.Errorf("failed to get table columns: %w", err)
	}

	// Build WHERE clause based on keywords and column types
	whereClauses := make([]string, 0)
	for _, keyword := range keywords {
		for _, column := range columns {
			if ss.isTextColumn(column.Type) {
				whereClauses = append(whereClauses, fmt.Sprintf("%s LIKE '%%%s%%'", column.Name, keyword))
			}
		}
	}

	if len(whereClauses) == 0 {
		return fmt.Sprintf("SELECT * FROM %s LIMIT %d", table, ss.maxRows), nil
	}

	whereClause := strings.Join(whereClauses, " OR ")
	return fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT %d", table, whereClause, ss.maxRows), nil
}

// generateCountQuery generates count-based queries
func (ss *SQLiteServer) generateCountQuery(keywords []string, tables []string) (string, error) {
	table := ss.findBestMatchingTable(keywords, tables)
	if table == "" {
		table = tables[0]
	}

	if len(keywords) == 0 {
		return fmt.Sprintf("SELECT COUNT(*) as total_count FROM %s", table), nil
	}

	// Generate count with conditions
	columns, err := ss.getTableColumns(table)
	if err != nil {
		return fmt.Sprintf("SELECT COUNT(*) as total_count FROM %s", table), nil
	}

	whereClauses := make([]string, 0)
	for _, keyword := range keywords {
		for _, column := range columns {
			if ss.isTextColumn(column.Type) {
				whereClauses = append(whereClauses, fmt.Sprintf("%s LIKE '%%%s%%'", column.Name, keyword))
			}
		}
	}

	if len(whereClauses) > 0 {
		whereClause := strings.Join(whereClauses, " OR ")
		return fmt.Sprintf("SELECT COUNT(*) as total_count FROM %s WHERE %s", table, whereClause), nil
	}

	return fmt.Sprintf("SELECT COUNT(*) as total_count FROM %s", table), nil
}

// generateRecentQuery generates queries for recent data
func (ss *SQLiteServer) generateRecentQuery(keywords []string, tables []string) (string, error) {
	table := ss.findBestMatchingTable(keywords, tables)
	if table == "" {
		table = tables[0]
	}

	columns, err := ss.getTableColumns(table)
	if err != nil {
		return "", fmt.Errorf("failed to get table columns: %w", err)
	}

	// Find timestamp/date columns
	var timeColumn string
	for _, column := range columns {
		if ss.isTimeColumn(column.Name) {
			timeColumn = column.Name
			break
		}
	}

	baseQuery := fmt.Sprintf("SELECT * FROM %s", table)

	if timeColumn != "" {
		baseQuery += fmt.Sprintf(" ORDER BY %s DESC", timeColumn)
	} else {
		// Fallback: assume ROWID for ordering
		baseQuery += " ORDER BY ROWID DESC"
	}

	baseQuery += fmt.Sprintf(" LIMIT %d", min(ss.maxRows, 20)) // Recent queries typically need fewer results

	return baseQuery, nil
}

// generatePopularQuery generates queries for popular/frequent data
func (ss *SQLiteServer) generatePopularQuery(keywords []string, tables []string) (string, error) {
	table := ss.findBestMatchingTable(keywords, tables)
	if table == "" {
		table = tables[0]
	}

	columns, err := ss.getTableColumns(table)
	if err != nil {
		return "", fmt.Errorf("failed to get table columns: %w", err)
	}

	// Look for count-related columns or use generic grouping
	var groupColumn string
	for _, column := range columns {
		if ss.isTextColumn(column.Type) && !ss.isTimeColumn(column.Name) {
			groupColumn = column.Name
			break
		}
	}

	if groupColumn != "" {
		return fmt.Sprintf("SELECT %s, COUNT(*) as frequency FROM %s GROUP BY %s ORDER BY frequency DESC LIMIT %d",
			groupColumn, table, groupColumn, min(ss.maxRows, 50)), nil
	}

	// Fallback to recent data
	return ss.generateRecentQuery(keywords, tables)
}

// generateSchemaQuery generates schema information queries
func (ss *SQLiteServer) generateSchemaQuery(keywords []string, tables []string) (string, error) {
	if len(keywords) > 0 {
		table := ss.findBestMatchingTable(keywords, tables)
		if table != "" {
			return fmt.Sprintf("PRAGMA table_info(%s)", table), nil
		}
	}

	// Return schema for all tables
	return "SELECT name, sql FROM sqlite_master WHERE type='table' ORDER BY name", nil
}

// generateGenericQuery generates a generic query based on intent
func (ss *SQLiteServer) generateGenericQuery(intent string, keywords []string, tables []string) (string, error) {
	table := tables[0] // Default to first table
	if len(keywords) > 0 {
		if bestMatch := ss.findBestMatchingTable(keywords, tables); bestMatch != "" {
			table = bestMatch
		}
	}

	return fmt.Sprintf("SELECT * FROM %s LIMIT %d", table, min(ss.maxRows, 100)), nil
}

// =============================================================================
// SCHEMA INFORMATION OPERATIONS
// =============================================================================

// handleSchemaInfo returns comprehensive database schema information
func (ss *SQLiteServer) handleSchemaInfo(ctx context.Context, operation *models.MCPOperation) (*models.MCPResult, error) {
	startTime := time.Now()

	ss.logger.LogInfo("MCP", "Retrieving database schema information", nil)

	// Get all tables
	tables, err := ss.getAllTablesInfo()
	if err != nil {
		return ss.createErrorResult(operation.ID, fmt.Errorf("failed to get tables info: %w", err))
	}

	// Get indexes
	indexes, err := ss.getIndexesInfo()
	if err != nil {
		ss.logger.LogError("Failed to get indexes info", "error", err)
		indexes = make([]IndexInfo, 0)
	}

	// Get views
	views, err := ss.getViewsInfo()
	if err != nil {
		ss.logger.LogError("Failed to get views info", "error", err)
		views = make([]ViewInfo, 0)
	}

	result := DatabaseSchema{
		DatabasePath: ss.dbPath,
		Tables:       tables,
		Indexes:      indexes,
		Views:        views,
		Statistics: SchemaStatistics{
			TableCount: len(tables),
			IndexCount: len(indexes),
			ViewCount:  len(views),
			TotalRows:  ss.calculateTotalRows(tables),
		},
		GeneratedAt: time.Now(),
	}

	executionTime := time.Since(startTime)
	ss.logger.LogInfo("MCP", "Schema information retrieved", map[string]interface{}{
		"tables":         len(tables),
		"execution_time": executionTime,
	})

	return &models.MCPResult{
		OperationID:   operation.ID,
		
		
		Success:       true,
		Data:          result,
		Duration: executionTime,
		Metadata: map[string]interface{}{
			"server": "sqlite",
			"type":   operation.Type,
		},
		
	}, nil
}

// handleTablesInfo returns information about database tables
func (ss *SQLiteServer) handleTablesInfo(ctx context.Context, operation *models.MCPOperation) (*models.MCPResult, error) {
	startTime := time.Now()

	tablePattern, _ := operation.Parameters["pattern"].(string)
	includeColumns, _ := operation.Parameters["include_columns"].(bool)

	tables, err := ss.getTablesInfo(tablePattern, includeColumns)
	if err != nil {
		return ss.createErrorResult(operation.ID, fmt.Errorf("failed to get tables info: %w", err))
	}

	result := map[string]interface{}{
		"tables":      tables,
		"table_count": len(tables),
		"pattern":     tablePattern,
	}

	executionTime := time.Since(startTime)

	return &models.MCPResult{
		OperationID:   operation.ID,
		
		
		Success:       true,
		Data:          result,
		Duration: executionTime,
		Metadata: map[string]interface{}{
			"server": "sqlite",
			"type":   operation.Type,
		},
		
	}, nil
}

// =============================================================================
// DATA STRUCTURES FOR SCHEMA INFORMATION
// =============================================================================

type DatabaseSchema struct {
	DatabasePath string           `json:"database_path"`
	Tables       []TableInfo      `json:"tables"`
	Indexes      []IndexInfo      `json:"indexes"`
	Views        []ViewInfo       `json:"views"`
	Statistics   SchemaStatistics `json:"statistics"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

type TableInfo struct {
	Name         string       `json:"name"`
	Columns      []ColumnInfo `json:"columns"`
	RowCount     int64        `json:"row_count"`
	CreateSQL    string       `json:"create_sql"`
	LastModified time.Time    `json:"last_modified,omitempty"`
}

type ColumnInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	NotNull      bool   `json:"not_null"`
	DefaultValue string `json:"default_value"`
	PrimaryKey   bool   `json:"primary_key"`
}

type IndexInfo struct {
	Name      string   `json:"name"`
	Table     string   `json:"table"`
	Columns   []string `json:"columns"`
	Unique    bool     `json:"unique"`
	CreateSQL string   `json:"create_sql"`
}

type ViewInfo struct {
	Name      string `json:"name"`
	CreateSQL string `json:"create_sql"`
}

type SchemaStatistics struct {
	TableCount int   `json:"table_count"`
	IndexCount int   `json:"index_count"`
	ViewCount  int   `json:"view_count"`
	TotalRows  int64 `json:"total_rows"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// validateQuery validates SQL queries for security
func (ss *SQLiteServer) validateQuery(query string) error {
	queryUpper := strings.ToUpper(strings.TrimSpace(query))

	// Check if read-only mode is enforced
	if ss.readOnly {
		if !strings.HasPrefix(queryUpper, "SELECT") &&
			!strings.HasPrefix(queryUpper, "PRAGMA") &&
			!strings.HasPrefix(queryUpper, "EXPLAIN") {
			return fmt.Errorf("only SELECT, PRAGMA, and EXPLAIN queries allowed in read-only mode")
		}
	}

	// Check blocked operations
	for _, blocked := range ss.blockedOperations {
		if strings.Contains(queryUpper, strings.ToUpper(blocked)) {
			return fmt.Errorf("operation '%s' is not allowed", blocked)
		}
	}

	// Check allowed operations (if specified)
	if len(ss.allowedOperations) > 0 {
		allowed := false
		for _, allowedOp := range ss.allowedOperations {
			if strings.HasPrefix(queryUpper, strings.ToUpper(allowedOp)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("operation not in allowed list: %v", ss.allowedOperations)
		}
	}

	// Prevent potentially dangerous patterns
	dangerousPatterns := []string{
		`(?i)\bATTACH\b`,
		`(?i)\bDETACH\b`,
		`(?i)\..*read`, // File system functions
		`(?i)\..*write`,
	}

	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			return fmt.Errorf("query contains potentially dangerous operations")
		}
	}

	return nil
}

// processQueryResults processes SQL query results into structured format
func (ss *SQLiteServer) processQueryResults(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	rowCount := 0

	for rows.Next() && rowCount < ss.maxRows {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create result row
		row := make(map[string]interface{})
		for i, col := range columns {
			// Handle different data types
			val := values[i]
			if val != nil {
				switch v := val.(type) {
				case []byte:
					row[col] = string(v)
				default:
					row[col] = v
				}
			} else {
				row[col] = nil
			}
		}

		results = append(results, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

// detectQueryType detects the type of SQL query
func (ss *SQLiteServer) detectQueryType(query string) string {
	queryUpper := strings.ToUpper(strings.TrimSpace(query))

	switch {
	case strings.HasPrefix(queryUpper, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(queryUpper, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(queryUpper, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(queryUpper, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(queryUpper, "CREATE"):
		return "CREATE"
	case strings.HasPrefix(queryUpper, "DROP"):
		return "DROP"
	case strings.HasPrefix(queryUpper, "ALTER"):
		return "ALTER"
	case strings.HasPrefix(queryUpper, "PRAGMA"):
		return "PRAGMA"
	default:
		return "OTHER"
	}
}

// getTableNames gets all table names in the database
func (ss *SQLiteServer) getTableNames() ([]string, error) {
	rows, err := ss.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// getTableColumns gets column information for a table
func (ss *SQLiteServer) getTableColumns(tableName string) ([]ColumnInfo, error) {
	rows, err := ss.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultVal sql.NullString

		err := rows.Scan(
			&col.ID,
			&col.Name,
			&col.Type,
			&col.NotNull,
			&defaultVal,
			&col.PrimaryKey,
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = defaultVal.String
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// getAllTablesInfo gets comprehensive information about all tables
func (ss *SQLiteServer) getAllTablesInfo() ([]TableInfo, error) {
	tableNames, err := ss.getTableNames()
	if err != nil {
		return nil, err
	}

	var tables []TableInfo
	for _, tableName := range tableNames {
		tableInfo, err := ss.getTableInfo(tableName)
		if err != nil {
			ss.logger.LogError("MCP", "Failed to get info for table", fmt.Errorf("table info error"), map[string]interface{}{
			"table": tableName,
		})
			continue
		}
		tables = append(tables, *tableInfo)
	}

	return tables, nil
}

// getTableInfo gets detailed information about a specific table
func (ss *SQLiteServer) getTableInfo(tableName string) (*TableInfo, error) {
	// Get columns
	columns, err := ss.getTableColumns(tableName)
	if err != nil {
		return nil, err
	}

	// Get row count
	var rowCount int64
	err = ss.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount)
	if err != nil {
		ss.logger.LogError("MCP", "Failed to get row count", err, map[string]interface{}{
			"table": tableName,
		})
		rowCount = 0
	}

	// Get CREATE statement
	var createSQL string
	err = ss.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL)
	if err != nil {
		ss.logger.LogError("MCP", "Failed to get CREATE statement", err, map[string]interface{}{
			"table": tableName,
		})
	}

	return &TableInfo{
		Name:      tableName,
		Columns:   columns,
		RowCount:  rowCount,
		CreateSQL: createSQL,
	}, nil
}

// getTablesInfo gets table information with optional filtering
func (ss *SQLiteServer) getTablesInfo(pattern string, includeColumns bool) ([]TableInfo, error) {
	tables, err := ss.getAllTablesInfo()
	if err != nil {
		return nil, err
	}

	// Filter by pattern if provided
	if pattern != "" {
		filtered := make([]TableInfo, 0)
		for _, table := range tables {
			if matched, _ := regexp.MatchString(pattern, table.Name); matched {
				filtered = append(filtered, table)
			}
		}
		tables = filtered
	}

	// Remove columns if not requested
	if !includeColumns {
		for i := range tables {
			tables[i].Columns = nil
		}
	}

	return tables, nil
}

// getIndexesInfo gets information about database indexes
func (ss *SQLiteServer) getIndexesInfo() ([]IndexInfo, error) {
	rows, err := ss.db.Query("SELECT name, tbl_name, sql FROM sqlite_master WHERE type='index' AND sql IS NOT NULL ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var index IndexInfo
		var sql sql.NullString

		if err := rows.Scan(&index.Name, &index.Table, &sql); err != nil {
			continue
		}

		if sql.Valid {
			index.CreateSQL = sql.String
			index.Unique = strings.Contains(strings.ToUpper(sql.String), "UNIQUE")
		}

		// Get index columns (simplified)
		indexColumns, err := ss.getIndexColumns(index.Name)
		if err == nil {
			index.Columns = indexColumns
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// getIndexColumns gets columns for a specific index
func (ss *SQLiteServer) getIndexColumns(indexName string) ([]string, error) {
	rows, err := ss.db.Query(fmt.Sprintf("PRAGMA index_info(%s)", indexName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqno int
		var cid int
		var name string

		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			continue
		}
		columns = append(columns, name)
	}

	return columns, nil
}

// getViewsInfo gets information about database views
func (ss *SQLiteServer) getViewsInfo() ([]ViewInfo, error) {
	rows, err := ss.db.Query("SELECT name, sql FROM sqlite_master WHERE type='view' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []ViewInfo
	for rows.Next() {
		var view ViewInfo
		if err := rows.Scan(&view.Name, &view.CreateSQL); err != nil {
			continue
		}
		views = append(views, view)
	}

	return views, nil
}

// findBestMatchingTable finds the table that best matches the given keywords
func (ss *SQLiteServer) findBestMatchingTable(keywords []string, tables []string) string {
	if len(keywords) == 0 || len(tables) == 0 {
		return ""
	}

	scores := make(map[string]int)

	for _, table := range tables {
		tableLower := strings.ToLower(table)
		score := 0

		for _, keyword := range keywords {
			keywordLower := strings.ToLower(keyword)
			if strings.Contains(tableLower, keywordLower) {
				score += 10
			}
			if strings.HasPrefix(tableLower, keywordLower) {
				score += 5
			}
			if strings.HasSuffix(tableLower, keywordLower) {
				score += 3
			}
		}

		scores[table] = score
	}

	// Find table with highest score
	var bestTable string
	var bestScore int
	for table, score := range scores {
		if score > bestScore {
			bestScore = score
			bestTable = table
		}
	}

	return bestTable
}

// isTextColumn checks if a column type is text-based
func (ss *SQLiteServer) isTextColumn(colType string) bool {
	textTypes := []string{"TEXT", "VARCHAR", "CHAR", "STRING"}
	colTypeUpper := strings.ToUpper(colType)

	for _, textType := range textTypes {
		if strings.Contains(colTypeUpper, textType) {
			return true
		}
	}
	return false
}

// isTimeColumn checks if a column name suggests it contains time data
func (ss *SQLiteServer) isTimeColumn(colName string) bool {
	timePatterns := []string{
		"created", "updated", "modified", "timestamp", "time", "date",
		"created_at", "updated_at", "modified_at", "datetime",
	}

	colNameLower := strings.ToLower(colName)
	for _, pattern := range timePatterns {
		if strings.Contains(colNameLower, pattern) {
			return true
		}
	}
	return false
}

// calculateTotalRows calculates total rows across all tables
func (ss *SQLiteServer) calculateTotalRows(tables []TableInfo) int64 {
	var total int64
	for _, table := range tables {
		total += table.RowCount
	}
	return total
}

// createErrorResult creates a standardized error result
func (ss *SQLiteServer) createErrorResult(operationID string, err error) (*models.MCPResult, error) {
	return &models.MCPResult{
		OperationID: operationID,
		Success:     false,
		Error:       err.Error(),
		Metadata: map[string]interface{}{
			"server": "sqlite",
		},
	}, nil
}

// interfaceToStringSlice converts interface{} to []string
func interfaceToStringSlice(v interface{}) []string {
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, len(slice))
		for i, item := range slice {
			if str, ok := item.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	if slice, ok := v.([]string); ok {
		return slice
	}
	return []string{}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
