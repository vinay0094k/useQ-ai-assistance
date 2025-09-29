package app

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"

	"github.com/yourusername/useq-ai-assistant/storage"
)

// SessionManager handles user sessions, history, and learning
type SessionManager struct {
	storage        *storage.SQLiteDB
	activeSessions map[string]*Session
	mu             sync.RWMutex
	config         SessionConfig
}

// Session represents an active user session
type Session struct {
	ID              string             `json:"id"`
	StartTime       time.Time          `json:"start_time"`
	LastActivity    time.Time          `json:"last_activity"`
	QueryCount      int                `json:"query_count"`
	TotalTokens     int                `json:"total_tokens"`
	TotalCost       float64            `json:"total_cost"`
	QueryHistory    []QueryResponse    `json:"query_history"`
	UserPreferences UserPreferences    `json:"user_preferences"`
	LearningContext LearningContext    `json:"learning_context"`
	Performance     SessionPerformance `json:"performance"`
	mu              sync.RWMutex       `json:"-"`
}

// QueryResponse pairs a query with its response for session history
type QueryResponse struct {
	Query          *models.Query    `json:"query"`
	Response       *models.Response `json:"response"`
	UserFeedback   *UserFeedback    `json:"user_feedback,omitempty"`
	Timestamp      time.Time        `json:"timestamp"`
	ProcessingTime time.Duration    `json:"processing_time"`
	Success        bool             `json:"success"`
}

// UserFeedback represents user feedback on responses
type UserFeedback struct {
	Rating       int          `json:"rating"` // 1-5 stars
	Helpful      bool         `json:"helpful"`
	Accurate     bool         `json:"accurate"`
	Complete     bool         `json:"complete"`
	Comments     string       `json:"comments,omitempty"`
	Corrections  []Correction `json:"corrections,omitempty"`
	Timestamp    time.Time    `json:"timestamp"`
	FeedbackType FeedbackType `json:"feedback_type"`
}

// Correction represents a user correction to improve future responses
type Correction struct {
	Original   string         `json:"original"`
	Corrected  string         `json:"corrected"`
	Context    string         `json:"context"`
	Type       CorrectionType `json:"type"`
	Confidence float64        `json:"confidence"`
}

// FeedbackType represents different types of user feedback
type FeedbackType string

const (
	FeedbackTypeRating     FeedbackType = "rating"
	FeedbackTypeCorrection FeedbackType = "correction"
	FeedbackTypeFlag       FeedbackType = "flag"
	FeedbackTypeAccept     FeedbackType = "accept"
	FeedbackTypeReject     FeedbackType = "reject"
)

// CorrectionType represents different types of corrections
type CorrectionType string

const (
	CorrectionTypeCode        CorrectionType = "code"
	CorrectionTypeExplanation CorrectionType = "explanation"
	CorrectionTypeSearch      CorrectionType = "search"
	CorrectionTypeFact        CorrectionType = "fact"
)

// UserPreferences stores user preferences and patterns
type UserPreferences struct {
	PreferredLanguage  string           `json:"preferred_language"`
	CodeStyle          string           `json:"code_style"`
	VerbosityLevel     int              `json:"verbosity_level"` // 1-5
	ShowLineNumbers    bool             `json:"show_line_numbers"`
	PreferredProviders []string         `json:"preferred_providers"`
	CustomKeywords     []string         `json:"custom_keywords"`
	ProjectPatterns    []ProjectPattern `json:"project_patterns"`
	LastUpdated        time.Time        `json:"last_updated"`
}

// ProjectPattern represents learned patterns from the user's project
type ProjectPattern struct {
	Pattern    string    `json:"pattern"`
	Context    string    `json:"context"`
	Frequency  int       `json:"frequency"`
	LastSeen   time.Time `json:"last_seen"`
	Confidence float64   `json:"confidence"`
}

// LearningContext tracks what the system has learned about the user
type LearningContext struct {
	CommonQueryTypes   map[models.QueryType]int `json:"common_query_types"`
	FrequentKeywords   map[string]int           `json:"frequent_keywords"`
	PreferredAgents    map[string]int           `json:"preferred_agents"`
	SuccessfulPatterns []LearnedPattern         `json:"successful_patterns"`
	CorrectionPatterns []LearnedPattern         `json:"correction_patterns"`
	LastLearningUpdate time.Time                `json:"last_learning_update"`
}

// LearnedPattern represents a pattern the system has learned
type LearnedPattern struct {
	Input          string    `json:"input"`
	ExpectedOutput string    `json:"expected_output"`
	Context        string    `json:"context"`
	Confidence     float64   `json:"confidence"`
	UsageCount     int       `json:"usage_count"`
	LastUsed       time.Time `json:"last_used"`
	Success        bool      `json:"success"`
}

// SessionPerformance tracks session performance metrics
type SessionPerformance struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	SuccessRate         float64       `json:"success_rate"`
	UserSatisfaction    float64       `json:"user_satisfaction"`
	TokenEfficiency     float64       `json:"token_efficiency"`
	CostEfficiency      float64       `json:"cost_efficiency"`
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	MaxHistorySize  int           `json:"max_history_size"`
	SessionTimeout  time.Duration `json:"session_timeout"`
	AutoSave        bool          `json:"auto_save"`
	LearningEnabled bool          `json:"learning_enabled"`
	FeedbackEnabled bool          `json:"feedback_enabled"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(storage *storage.SQLiteDB) *SessionManager {
	return &SessionManager{
		storage:        storage,
		activeSessions: make(map[string]*Session),
		config: SessionConfig{
			MaxHistorySize:  100,
			SessionTimeout:  30 * time.Minute,
			AutoSave:        true,
			LearningEnabled: true,
			FeedbackEnabled: true,
		},
	}
}

// GetOrCreateSession gets an existing session or creates a new one
func (sm *SessionManager) GetOrCreateSession(sessionID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if session exists in memory
	if session, exists := sm.activeSessions[sessionID]; exists {
		session.LastActivity = time.Now()
		return session
	}

	// Try to load from storage
	session := sm.loadSessionFromStorage(sessionID)
	if session != nil {
		session.LastActivity = time.Now()
		sm.activeSessions[sessionID] = session
		return session
	}

	// Create new session
	session = &Session{
		ID:           sessionID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		QueryHistory: make([]QueryResponse, 0),
		UserPreferences: UserPreferences{
			PreferredLanguage:  "go",
			VerbosityLevel:     3,
			ShowLineNumbers:    true,
			PreferredProviders: []string{"openai", "gemini"},
			LastUpdated:        time.Now(),
		},
		LearningContext: LearningContext{
			CommonQueryTypes:   make(map[models.QueryType]int),
			FrequentKeywords:   make(map[string]int),
			PreferredAgents:    make(map[string]int),
			LastLearningUpdate: time.Now(),
		},
	}

	sm.activeSessions[sessionID] = session

	if sm.config.AutoSave {
		go sm.saveSessionToStorage(session)
	}

	return session
}

// SaveQuery saves a query and its response to the session
func (sm *SessionManager) SaveQuery(query *models.Query, response *models.Response) error {
	session := sm.GetOrCreateSession(response.QueryID)

	session.mu.Lock()
	defer session.mu.Unlock()

	// Create query-response pair
	qr := QueryResponse{
		Query:          query,
		Response:       response,
		Timestamp:      time.Now(),
		ProcessingTime: response.Metadata.GenerationTime,
		Success:        response.Type != models.ResponseTypeError,
	}

	// Add to history
	session.QueryHistory = append(session.QueryHistory, qr)

	// Update session metrics
	session.QueryCount++
	session.TotalTokens += response.TokenUsage.TotalTokens
	session.TotalCost += response.Cost.TotalCost
	session.LastActivity = time.Now()

	// Trim history if too large
	if len(session.QueryHistory) > sm.config.MaxHistorySize {
		session.QueryHistory = session.QueryHistory[len(session.QueryHistory)-sm.config.MaxHistorySize:]
	}

	// Update learning context
	sm.updateLearningContext(session, query, response)

	// Auto-save if enabled
	if sm.config.AutoSave {
		go sm.saveSessionToStorage(session)
	}

	return nil
}

// AddUserFeedback adds user feedback to a query response
func (sm *SessionManager) AddUserFeedback(sessionID, queryID string, feedback *UserFeedback) error {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.Lock()
	defer session.mu.Unlock()

	// Find the query response and add feedback
	for i := range session.QueryHistory {
		if session.QueryHistory[i].Query.ID == queryID {
			session.QueryHistory[i].UserFeedback = feedback

			// Update learning context with feedback
			sm.processFeedbackLearning(session, &session.QueryHistory[i])

			// Auto-save if enabled
			if sm.config.AutoSave {
				go sm.saveSessionToStorage(session)
			}

			return nil
		}
	}

	return fmt.Errorf("query not found in session history")
}

// GetSessionHistory returns the query history for a session
func (sm *SessionManager) GetSessionHistory(sessionID string, limit int) ([]QueryResponse, error) {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.RLock()
	defer session.mu.RUnlock()

	history := session.QueryHistory
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:], nil
	}

	return history, nil
}

// GetUserPreferences returns user preferences for a session
func (sm *SessionManager) GetUserPreferences(sessionID string) UserPreferences {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.RLock()
	defer session.mu.RUnlock()

	return session.UserPreferences
}

// UpdateUserPreferences updates user preferences
func (sm *SessionManager) UpdateUserPreferences(sessionID string, preferences UserPreferences) {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.Lock()
	defer session.mu.Unlock()

	preferences.LastUpdated = time.Now()
	session.UserPreferences = preferences

	if sm.config.AutoSave {
		go sm.saveSessionToStorage(session)
	}
}

// GetLearningInsights returns learning insights for improving responses
func (sm *SessionManager) GetLearningInsights(sessionID string) *LearningContext {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.RLock()
	defer session.mu.RUnlock()

	return &session.LearningContext
}

// GetSessionStats returns statistics for a session
func (sm *SessionManager) GetSessionStats(sessionID string) SessionStats {
	session := sm.GetOrCreateSession(sessionID)

	session.mu.RLock()
	defer session.mu.RUnlock()

	return SessionStats{
		SessionID:    sessionID,
		Duration:     time.Since(session.StartTime),
		QueryCount:   session.QueryCount,
		TotalTokens:  session.TotalTokens,
		TotalCost:    session.TotalCost,
		LastActivity: session.LastActivity,
		Performance:  session.Performance,
	}
}

// SessionStats represents session statistics
type SessionStats struct {
	SessionID    string             `json:"session_id"`
	Duration     time.Duration      `json:"duration"`
	QueryCount   int                `json:"query_count"`
	TotalTokens  int                `json:"total_tokens"`
	TotalCost    float64            `json:"total_cost"`
	LastActivity time.Time          `json:"last_activity"`
	Performance  SessionPerformance `json:"performance"`
}

// CleanupExpiredSessions removes expired sessions from memory
func (sm *SessionManager) CleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for sessionID, session := range sm.activeSessions {
		if now.Sub(session.LastActivity) > sm.config.SessionTimeout {
			// Save before removing
			if sm.config.AutoSave {
				sm.saveSessionToStorage(session)
			}
			delete(sm.activeSessions, sessionID)
		}
	}
}

// Private helper methods

// updateLearningContext updates the learning context based on query/response
func (sm *SessionManager) updateLearningContext(session *Session, query *models.Query, response *models.Response) {
	if !sm.config.LearningEnabled {
		return
	}

	ctx := &session.LearningContext

	// Update query type frequency
	ctx.CommonQueryTypes[query.Type]++

	// Update keyword frequency
	if intent, err := ParseIntent(query.UserInput); err == nil {
		for _, keyword := range intent.Keywords {
			ctx.FrequentKeywords[keyword]++
		}
	}

	// Update preferred agents
	ctx.PreferredAgents[response.AgentUsed]++

	// Add successful patterns
	if response.Type != models.ResponseTypeError {
		pattern := LearnedPattern{
			Input:          query.UserInput,
			ExpectedOutput: response.Content.Text,
			Context:        fmt.Sprintf("type:%s,agent:%s", query.Type, response.AgentUsed),
			Confidence:     response.Quality.Accuracy,
			UsageCount:     1,
			LastUsed:       time.Now(),
			Success:        true,
		}
		ctx.SuccessfulPatterns = append(ctx.SuccessfulPatterns, pattern)
	}

	ctx.LastLearningUpdate = time.Now()
}

// processFeedbackLearning processes user feedback for learning
func (sm *SessionManager) processFeedbackLearning(session *Session, qr *QueryResponse) {
	if !sm.config.LearningEnabled || qr.UserFeedback == nil {
		return
	}

	feedback := qr.UserFeedback
	ctx := &session.LearningContext

	// Process corrections
	for _, correction := range feedback.Corrections {
		pattern := LearnedPattern{
			Input:          qr.Query.UserInput,
			ExpectedOutput: correction.Corrected,
			Context:        correction.Context,
			Confidence:     correction.Confidence,
			UsageCount:     1,
			LastUsed:       time.Now(),
			Success:        feedback.Rating >= 3,
		}
		ctx.CorrectionPatterns = append(ctx.CorrectionPatterns, pattern)
	}

	// Update performance metrics
	session.Performance.UserSatisfaction = sm.calculateSatisfactionScore(session)
}

// calculateSatisfactionScore calculates user satisfaction based on feedback
func (sm *SessionManager) calculateSatisfactionScore(session *Session) float64 {
	totalRatings := 0
	ratingSum := 0

	for _, qr := range session.QueryHistory {
		if qr.UserFeedback != nil && qr.UserFeedback.Rating > 0 {
			totalRatings++
			ratingSum += qr.UserFeedback.Rating
		}
	}

	if totalRatings == 0 {
		return 0.0
	}

	return float64(ratingSum) / float64(totalRatings) / 5.0 // Normalize to 0-1
}

// saveSessionToStorage saves a session to persistent storage
func (sm *SessionManager) saveSessionToStorage(session *Session) error {
	session.mu.RLock()
	defer session.mu.RUnlock()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return sm.storage.SaveSession(session.ID, data)
}

// loadSessionFromStorage loads a session from persistent storage
func (sm *SessionManager) loadSessionFromStorage(sessionID string) *Session {
	data, err := sm.storage.LoadSession(sessionID)
	if err != nil {
		return nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	return &session
}

// ParseIntent is a helper function to parse intent (references the PromptParser)
func ParseIntent(input string) (*models.QueryIntent, error) {
	parser := NewPromptParser()
	return parser.ParseIntent(input)
}
