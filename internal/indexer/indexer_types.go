package indexer

import (
	"sync"
	"time"
)

// ChunkType represents different types of code chunks
type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "function"
	ChunkTypeMethod    ChunkType = "method"
	ChunkTypeType      ChunkType = "type"
	ChunkTypeInterface ChunkType = "interface"
	ChunkTypePackage   ChunkType = "package"
	ChunkTypeImport    ChunkType = "import"
	ChunkTypeComment   ChunkType = "comment"
	ChunkTypeGeneric   ChunkType = "generic"
	ChunkTypeFile      ChunkType = "file"
	ChunkTypeStruct    ChunkType = "struct"
	ChunkTypeVariable  ChunkType = "variable"
	ChunkTypeConstant  ChunkType = "constant"
)

// CodeChunk represents a chunk of code for embedding
type CodeChunk struct {
	ID         string            `json:"id"`
	FileID     string            `json:"file_id"`
	FilePath   string            `json:"file_path"`
	ChunkIndex int               `json:"chunk_index"`
	Content    string            `json:"content"`
	StartLine  int               `json:"start_line"`
	EndLine    int               `json:"end_line"`
	Language   string            `json:"language"`
	Type       ChunkType         `json:"type"`
	Context    ChunkContext      `json:"context"`
	Metadata   map[string]string `json:"metadata"`
}

// ChunkContext provides context about the code chunk
type ChunkContext struct {
	PackageName   string   `json:"package_name"`
	FunctionName  string   `json:"function_name,omitempty"`
	TypeName      string   `json:"type_name,omitempty"`
	InterfaceName string   `json:"interface_name,omitempty"`
	Dependencies  []string `json:"dependencies"`
	Imports       []string `json:"imports"`
	Receiver      string   `json:"receiver,omitempty"`
}

// FileInfo represents information about an indexed file
type FileInfo struct {
	Path         string      `json:"path"`
	Hash         string      `json:"hash"`
	Size         int64       `json:"size"`
	LastModified time.Time   `json:"last_modified"`
	Language     string      `json:"language"`
	IndexedAt    time.Time   `json:"indexed_at"`
	ChunkCount   int         `json:"chunk_count"`
	ParsedData   *ParsedCode `json:"parsed_data,omitempty"`
}

// GraphNode represents a code entity in the knowledge graph
type GraphNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	FilePath string            `json:"file_path"`
	Package  string            `json:"package"`
	Metadata map[string]string `json:"metadata"`
}

// GraphEdge represents relationships between nodes
type GraphEdge struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	Relationship string  `json:"relationship"`
	Weight       float64 `json:"weight"`
}

// BatchJob represents a batch processing job
type BatchJob struct {
	ID       string      `json:"id"`
	Files    []string    `json:"files"`
	Status   JobStatus   `json:"status"`
	Progress int         `json:"progress"`
	Error    string      `json:"error,omitempty"`
	Started  time.Time   `json:"started"`
	Finished *time.Time  `json:"finished,omitempty"`
}

// JobStatus represents the status of a processing job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ChangeEvent represents a file system change
type ChangeEvent struct {
	Type     ChangeType `json:"type"`
	FilePath string     `json:"file_path"`
	Time     time.Time  `json:"time"`
}

// ChangeType represents the type of file system change
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeModify ChangeType = "modify"
	ChangeTypeDelete ChangeType = "delete"
	ChangeTypeRename ChangeType = "rename"
)

// ProcessingResult represents the result of processing a file
type ProcessingResult struct {
	FilePath string       `json:"file_path"`
	Chunks   []*CodeChunk `json:"chunks"`
	Error    error        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// IncrementalState tracks the state for incremental indexing
type IncrementalState struct {
	LastIndexTime time.Time         `json:"last_index_time"`
	FileHashes    map[string]string `json:"file_hashes"`
	mu            sync.RWMutex
}

// IndexerConfig holds indexer configuration
type IndexerConfig struct {
	MaxFileSize     int64         `json:"max_file_size"`
	BatchSize       int           `json:"batch_size"`
	MaxWorkers      int           `json:"max_workers"`
	ChunkSize       int           `json:"chunk_size"`
	ChunkOverlap    int           `json:"chunk_overlap"`
	IndexTimeout    time.Duration `json:"index_timeout"`
	EnableWatching  bool          `json:"enable_watching"`
	SkipBinaryFiles bool          `json:"skip_binary_files"`
	SkipTestFiles   bool          `json:"skip_test_files"`
	SkipVendor      bool          `json:"skip_vendor"`
}
