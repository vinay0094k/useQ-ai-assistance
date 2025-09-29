package display

import (
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// SyntaxHighlighter provides syntax highlighting for different programming languages
type SyntaxHighlighter struct {
	themes map[string]*HighlightTheme
	rules  map[string]*LanguageRules
}

// HighlightTheme defines colors for different syntax elements
type HighlightTheme struct {
	Name     string
	Keyword  *color.Color
	String   *color.Color
	Comment  *color.Color
	Function *color.Color
	Variable *color.Color
	Number   *color.Color
	Type     *color.Color
	Operator *color.Color
	Bracket  *color.Color
	Import   *color.Color
	Error    *color.Color
}

// LanguageRules defines syntax rules for a programming language
type LanguageRules struct {
	Language         string
	Keywords         []string
	Types            []string
	Functions        []string
	Operators        []string
	StringDelimiters []string
	CommentPatterns  []CommentPattern
	NumberPattern    *regexp.Regexp
	FunctionPattern  *regexp.Regexp
	VariablePattern  *regexp.Regexp
	ImportPattern    *regexp.Regexp
	BracketPairs     map[rune]rune
}

// CommentPattern defines comment patterns for a language
type CommentPattern struct {
	Type     CommentType
	Start    string
	End      string
	LineOnly bool
}

// CommentType represents different types of comments
type CommentType string

const (
	CommentTypeLine  CommentType = "line"
	CommentTypeBlock CommentType = "block"
)

// TokenType represents different types of tokens
type TokenType string

const (
	TokenKeyword  TokenType = "keyword"
	TokenString   TokenType = "string"
	TokenComment  TokenType = "comment"
	TokenFunction TokenType = "function"
	TokenVariable TokenType = "variable"
	TokenNumber   TokenType = "number"
	TokenTypeDecl TokenType = "type"
	TokenOperator TokenType = "operator"
	TokenBracket  TokenType = "bracket"
	TokenImport   TokenType = "import"
	TokenNormal   TokenType = "normal"
)

// Token represents a highlighted token
type Token struct {
	Type  TokenType
	Value string
	Color *color.Color
	Start int
	End   int
}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter() *SyntaxHighlighter {
	highlighter := &SyntaxHighlighter{
		themes: make(map[string]*HighlightTheme),
		rules:  make(map[string]*LanguageRules),
	}

	highlighter.initializeThemes()
	highlighter.initializeLanguageRules()

	return highlighter
}

// initializeThemes sets up color themes
func (sh *SyntaxHighlighter) initializeThemes() {
	// Dark theme (default)
	sh.themes["dark"] = &HighlightTheme{
		Name:     "dark",
		Keyword:  color.New(color.FgMagenta, color.Bold),            // Purple keywords
		String:   color.New(color.FgGreen),                          // Green strings
		Comment:  color.New(color.FgBlue),                           // Blue comments
		Function: color.New(color.FgYellow, color.Bold),             // Yellow functions
		Variable: color.New(color.FgCyan),                           // Cyan variables
		Number:   color.New(color.FgRed),                            // Red numbers
		Type:     color.New(color.FgGreen, color.Bold),              // Bold green types
		Operator: color.New(color.FgWhite, color.Bold),              // White operators
		Bracket:  color.New(color.FgYellow),                         // Yellow brackets
		Import:   color.New(color.FgMagenta),                        // Purple imports
		Error:    color.New(color.FgRed, color.Bold, color.BgWhite), // Red on white
	}

	// Light theme
	sh.themes["light"] = &HighlightTheme{
		Name:     "light",
		Keyword:  color.New(color.FgBlue, color.Bold),
		String:   color.New(color.FgGreen, color.Bold),
		Comment:  color.New(color.FgBlack, color.Faint),
		Function: color.New(color.FgMagenta, color.Bold),
		Variable: color.New(color.FgBlack),
		Number:   color.New(color.FgRed, color.Bold),
		Type:     color.New(color.FgBlue),
		Operator: color.New(color.FgBlack, color.Bold),
		Bracket:  color.New(color.FgMagenta),
		Import:   color.New(color.FgBlue),
		Error:    color.New(color.FgWhite, color.BgRed),
	}
}

// initializeLanguageRules sets up language-specific syntax rules
func (sh *SyntaxHighlighter) initializeLanguageRules() {
	// Go language rules
	sh.rules["go"] = &LanguageRules{
		Language: "go",
		Keywords: []string{
			"break", "case", "chan", "const", "continue", "default", "defer", "else",
			"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
			"map", "package", "range", "return", "select", "struct", "switch", "type",
			"var", "nil", "true", "false", "iota",
		},
		Types: []string{
			"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
			"int", "int8", "int16", "int32", "int64", "rune", "string",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		},
		Functions: []string{
			"make", "len", "cap", "new", "append", "copy", "delete", "close",
			"panic", "recover", "print", "println",
		},
		Operators: []string{
			"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>", "&^",
			"==", "!=", "<", "<=", ">", ">=", "&&", "||", "<-",
			"=", ":=", "++", "--", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^=",
		},
		StringDelimiters: []string{`"`, "`", `'`},
		CommentPatterns: []CommentPattern{
			{Type: CommentTypeLine, Start: "//", LineOnly: true},
			{Type: CommentTypeBlock, Start: "/*", End: "*/"},
		},
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?(e[+-]?\d+)?[i]?\b`),
		FunctionPattern: regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`),
		ImportPattern:   regexp.MustCompile(`"[^"]+"`),
		BracketPairs:    map[rune]rune{'(': ')', '[': ']', '{': '}'},
	}

	// JavaScript rules
	sh.rules["javascript"] = &LanguageRules{
		Language: "javascript",
		Keywords: []string{
			"break", "case", "catch", "class", "const", "continue", "debugger", "default",
			"delete", "do", "else", "export", "extends", "finally", "for", "function",
			"if", "import", "in", "instanceof", "let", "new", "return", "super", "switch",
			"this", "throw", "try", "typeof", "var", "void", "while", "with", "yield",
			"async", "await", "from", "as", "null", "undefined", "true", "false",
		},
		Types: []string{
			"Array", "Object", "String", "Number", "Boolean", "Function", "Date", "RegExp",
			"Error", "Promise", "Map", "Set", "WeakMap", "WeakSet",
		},
		Functions: []string{
			"console", "parseInt", "parseFloat", "isNaN", "isFinite", "setTimeout",
			"setInterval", "clearTimeout", "clearInterval", "require",
		},
		Operators:        []string{"+", "-", "*", "/", "%", "==", "===", "!=", "!==", "<", "<=", ">", ">=", "&&", "||", "!", "&", "|", "^", "~", "<<", ">>", ">>>"},
		StringDelimiters: []string{`"`, `'`, "`"},
		CommentPatterns: []CommentPattern{
			{Type: CommentTypeLine, Start: "//", LineOnly: true},
			{Type: CommentTypeBlock, Start: "/*", End: "*/"},
		},
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?(e[+-]?\d+)?\b`),
		FunctionPattern: regexp.MustCompile(`\b([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\(`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_$][a-zA-Z0-9_$]*\b`),
		ImportPattern:   regexp.MustCompile(`['"][^'"]+['"]`),
		BracketPairs:    map[rune]rune{'(': ')', '[': ']', '{': '}'},
	}

	// Python rules
	sh.rules["python"] = &LanguageRules{
		Language: "python",
		Keywords: []string{
			"False", "None", "True", "and", "as", "assert", "break", "class", "continue",
			"def", "del", "elif", "else", "except", "finally", "for", "from", "global",
			"if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass",
			"raise", "return", "try", "while", "with", "yield", "async", "await",
		},
		Types: []string{
			"int", "float", "str", "bool", "list", "tuple", "dict", "set", "frozenset",
			"bytes", "bytearray", "memoryview", "range", "enumerate", "zip",
		},
		Functions: []string{
			"print", "len", "range", "enumerate", "zip", "map", "filter", "reduce",
			"sum", "min", "max", "abs", "round", "sorted", "reversed", "any", "all",
			"isinstance", "issubclass", "hasattr", "getattr", "setattr", "delattr",
		},
		Operators:        []string{"+", "-", "*", "/", "//", "%", "**", "==", "!=", "<", "<=", ">", ">=", "and", "or", "not", "in", "is", "&", "|", "^", "~", "<<", ">>"},
		StringDelimiters: []string{`"`, `'`, `"""`, `'''`},
		CommentPatterns: []CommentPattern{
			{Type: CommentTypeLine, Start: "#", LineOnly: true},
		},
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?(e[+-]?\d+)?[jJ]?\b`),
		FunctionPattern: regexp.MustCompile(`\bdef\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`),
		ImportPattern:   regexp.MustCompile(`['"][^'"]+['"]`),
		BracketPairs:    map[rune]rune{'(': ')', '[': ']', '{': '}'},
	}

	// JSON rules (simplified)
	sh.rules["json"] = &LanguageRules{
		Language:         "json",
		Keywords:         []string{"true", "false", "null"},
		StringDelimiters: []string{`"`},
		CommentPatterns:  []CommentPattern{}, // JSON doesn't officially support comments
		NumberPattern:    regexp.MustCompile(`\b-?\d+(\.\d+)?(e[+-]?\d+)?\b`),
		BracketPairs:     map[rune]rune{'(': ')', '[': ']', '{': '}'},
	}
}

// Highlight highlights a line of code and returns the colorized string
func (sh *SyntaxHighlighter) Highlight(line, language string) string {
	if line == "" {
		return line
	}

	// Get language rules and theme
	rules := sh.getLanguageRules(language)
	theme := sh.getTheme("dark") // Default to dark theme

	// Tokenize the line
	tokens := sh.tokenize(line, rules)

	// Apply colors to tokens
	var result strings.Builder
	for _, token := range tokens {
		color := sh.getTokenColor(token.Type, theme)
		if color != nil {
			result.WriteString(color.Sprint(token.Value))
		} else {
			result.WriteString(token.Value)
		}
	}

	return result.String()
}

// HighlightBlock highlights a multi-line code block
func (sh *SyntaxHighlighter) HighlightBlock(code, language string) string {
	if code == "" {
		return code
	}

	lines := strings.Split(code, "\n")
	highlightedLines := make([]string, len(lines))

	for i, line := range lines {
		highlightedLines[i] = sh.Highlight(line, language)
	}

	return strings.Join(highlightedLines, "\n")
}

// tokenize breaks a line into tokens for highlighting
func (sh *SyntaxHighlighter) tokenize(line string, rules *LanguageRules) []Token {
	if rules == nil {
		return []Token{{Type: TokenNormal, Value: line}}
	}

	var tokens []Token
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		start := i

		// Skip whitespace
		if isWhitespace(runes[i]) {
			for i < len(runes) && isWhitespace(runes[i]) {
				i++
			}
			tokens = append(tokens, Token{
				Type:  TokenNormal,
				Value: string(runes[start:i]),
				Start: start,
				End:   i,
			})
			continue
		}

		// Check for comments
		if commentToken, newPos := sh.parseComment(runes, i, rules); commentToken != nil {
			tokens = append(tokens, *commentToken)
			i = newPos
			continue
		}

		// Check for strings
		if stringToken, newPos := sh.parseString(runes, i, rules); stringToken != nil {
			tokens = append(tokens, *stringToken)
			i = newPos
			continue
		}

		// Check for numbers
		if numberToken, newPos := sh.parseNumber(runes, i, rules); numberToken != nil {
			tokens = append(tokens, *numberToken)
			i = newPos
			continue
		}

		// Check for operators
		if operatorToken, newPos := sh.parseOperator(runes, i, rules); operatorToken != nil {
			tokens = append(tokens, *operatorToken)
			i = newPos
			continue
		}

		// Check for brackets
		if isBracket(runes[i], rules) {
			tokens = append(tokens, Token{
				Type:  TokenBracket,
				Value: string(runes[i]),
				Start: i,
				End:   i + 1,
			})
			i++
			continue
		}

		// Parse identifier (keyword, type, function, or variable)
		if identifierToken, newPos := sh.parseIdentifier(runes, i, rules); identifierToken != nil {
			tokens = append(tokens, *identifierToken)
			i = newPos
			continue
		}

		// Single character (fallback)
		tokens = append(tokens, Token{
			Type:  TokenNormal,
			Value: string(runes[i]),
			Start: i,
			End:   i + 1,
		})
		i++
	}

	return tokens
}

// parseComment parses comment tokens
func (sh *SyntaxHighlighter) parseComment(runes []rune, start int, rules *LanguageRules) (*Token, int) {
	for _, pattern := range rules.CommentPatterns {
		startRunes := []rune(pattern.Start)
		if start+len(startRunes) > len(runes) {
			continue
		}

		// Check if comment starts here
		match := true
		for i, r := range startRunes {
			if runes[start+i] != r {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		if pattern.LineOnly {
			// Line comment - consume to end of line
			return &Token{
				Type:  TokenComment,
				Value: string(runes[start:]),
				Start: start,
				End:   len(runes),
			}, len(runes)
		} else {
			// Block comment - find end pattern
			endRunes := []rune(pattern.End)
			pos := start + len(startRunes)

			for pos+len(endRunes) <= len(runes) {
				match := true
				for i, r := range endRunes {
					if runes[pos+i] != r {
						match = false
						break
					}
				}
				if match {
					pos += len(endRunes)
					return &Token{
						Type:  TokenComment,
						Value: string(runes[start:pos]),
						Start: start,
						End:   pos,
					}, pos
				}
				pos++
			}

			// Unterminated block comment
			return &Token{
				Type:  TokenComment,
				Value: string(runes[start:]),
				Start: start,
				End:   len(runes),
			}, len(runes)
		}
	}

	return nil, start
}

// parseString parses string literals
func (sh *SyntaxHighlighter) parseString(runes []rune, start int, rules *LanguageRules) (*Token, int) {
	for _, delimiter := range rules.StringDelimiters {
		delimRunes := []rune(delimiter)
		if start+len(delimRunes) > len(runes) {
			continue
		}

		// Check if string starts here
		match := true
		for i, r := range delimRunes {
			if runes[start+i] != r {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		// Find closing delimiter
		pos := start + len(delimRunes)
		escaped := false

		for pos < len(runes) {
			if !escaped && pos+len(delimRunes) <= len(runes) {
				// Check for closing delimiter
				match := true
				for i, r := range delimRunes {
					if runes[pos+i] != r {
						match = false
						break
					}
				}
				if match {
					pos += len(delimRunes)
					return &Token{
						Type:  TokenString,
						Value: string(runes[start:pos]),
						Start: start,
						End:   pos,
					}, pos
				}
			}

			if runes[pos] == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
			pos++
		}

		// Unterminated string
		return &Token{
			Type:  TokenString,
			Value: string(runes[start:]),
			Start: start,
			End:   len(runes),
		}, len(runes)
	}

	return nil, start
}

// parseNumber parses numeric literals
func (sh *SyntaxHighlighter) parseNumber(runes []rune, start int, rules *LanguageRules) (*Token, int) {
	if rules.NumberPattern == nil {
		return nil, start
	}

	line := string(runes)
	matches := rules.NumberPattern.FindStringIndex(line[start:])
	if matches == nil || matches[0] != 0 {
		return nil, start
	}

	end := start + matches[1]
	return &Token{
		Type:  TokenNumber,
		Value: string(runes[start:end]),
		Start: start,
		End:   end,
	}, end
}

// parseOperator parses operators
func (sh *SyntaxHighlighter) parseOperator(runes []rune, start int, rules *LanguageRules) (*Token, int) {
	// Sort operators by length (longest first) to match correctly
	for _, op := range rules.Operators {
		opRunes := []rune(op)
		if start+len(opRunes) > len(runes) {
			continue
		}

		match := true
		for i, r := range opRunes {
			if runes[start+i] != r {
				match = false
				break
			}
		}

		if match {
			return &Token{
				Type:  TokenOperator,
				Value: op,
				Start: start,
				End:   start + len(opRunes),
			}, start + len(opRunes)
		}
	}

	return nil, start
}

// parseIdentifier parses identifiers (keywords, types, functions, variables)
func (sh *SyntaxHighlighter) parseIdentifier(runes []rune, start int, rules *LanguageRules) (*Token, int) {
	if start >= len(runes) || !isIdentifierStart(runes[start]) {
		return nil, start
	}

	end := start + 1
	for end < len(runes) && isIdentifierContinue(runes[end]) {
		end++
	}

	identifier := string(runes[start:end])

	// Determine token type
	tokenType := TokenVariable // Default

	// Check if it's a keyword
	for _, keyword := range rules.Keywords {
		if identifier == keyword {
			tokenType = TokenKeyword
			break
		}
	}

	// Check if it's a type
	if tokenType == TokenVariable {
		for _, typeName := range rules.Types {
			if identifier == typeName {
				tokenType = TokenTypeDecl
				break
			}
		}
	}

	// Check if it's a built-in function
	if tokenType == TokenVariable {
		for _, funcName := range rules.Functions {
			if identifier == funcName {
				tokenType = TokenFunction
				break
			}
		}
	}

	// Check if it looks like a function call
	if tokenType == TokenVariable && end < len(runes) {
		// Skip whitespace
		pos := end
		for pos < len(runes) && isWhitespace(runes[pos]) {
			pos++
		}
		if pos < len(runes) && runes[pos] == '(' {
			tokenType = TokenFunction
		}
	}

	return &Token{
		Type:  tokenType,
		Value: identifier,
		Start: start,
		End:   end,
	}, end
}

// Helper functions

func (sh *SyntaxHighlighter) getLanguageRules(language string) *LanguageRules {
	if rules, exists := sh.rules[strings.ToLower(language)]; exists {
		return rules
	}
	return nil // Return nil for unsupported languages
}

func (sh *SyntaxHighlighter) getTheme(themeName string) *HighlightTheme {
	if theme, exists := sh.themes[themeName]; exists {
		return theme
	}
	return sh.themes["dark"] // Default to dark theme
}

func (sh *SyntaxHighlighter) getTokenColor(tokenType TokenType, theme *HighlightTheme) *color.Color {
	switch tokenType {
	case TokenKeyword:
		return theme.Keyword
	case TokenString:
		return theme.String
	case TokenComment:
		return theme.Comment
	case TokenFunction:
		return theme.Function
	case TokenVariable:
		return theme.Variable
	case TokenNumber:
		return theme.Number
	case TokenTypeDecl:
		return theme.Type
	case TokenOperator:
		return theme.Operator
	case TokenBracket:
		return theme.Bracket
	case TokenImport:
		return theme.Import
	default:
		return nil // Use default color
	}
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '$'
}

func isIdentifierContinue(r rune) bool {
	return isIdentifierStart(r) || (r >= '0' && r <= '9')
}

func isBracket(r rune, rules *LanguageRules) bool {
	if rules.BracketPairs == nil {
		return false
	}

	for open, close := range rules.BracketPairs {
		if r == open || r == close {
			return true
		}
	}
	return false
}

// SetTheme changes the active theme
func (sh *SyntaxHighlighter) SetTheme(themeName string) {
	// Theme is selected in getTheme method
	// This could store the active theme if needed
}

// AddLanguage adds support for a new programming language
func (sh *SyntaxHighlighter) AddLanguage(rules *LanguageRules) {
	sh.rules[strings.ToLower(rules.Language)] = rules
}

// GetSupportedLanguages returns a list of supported languages
func (sh *SyntaxHighlighter) GetSupportedLanguages() []string {
	languages := make([]string, 0, len(sh.rules))
	for lang := range sh.rules {
		languages = append(languages, lang)
	}
	return languages
}

// HighlightDiff highlights code differences (for showing changes)
func (sh *SyntaxHighlighter) HighlightDiff(oldCode, newCode, language string) (string, string) {
	oldHighlighted := sh.HighlightBlock(oldCode, language)
	newHighlighted := sh.HighlightBlock(newCode, language)

	// Add diff colors
	_ = sh.getTheme("dark")

	// Wrap old code with deletion color
	oldHighlighted = color.New(color.BgRed, color.FgWhite).Sprint(oldHighlighted)

	// Wrap new code with addition color
	newHighlighted = color.New(color.BgGreen, color.FgWhite).Sprint(newHighlighted)

	return oldHighlighted, newHighlighted
}

// CreateCustomTheme creates a custom color theme
func (sh *SyntaxHighlighter) CreateCustomTheme(name string, colors map[string]*color.Color) {
	theme := &HighlightTheme{Name: name}

	if c, ok := colors["keyword"]; ok {
		theme.Keyword = c
	} else {
		theme.Keyword = color.New(color.FgMagenta, color.Bold)
	}

	if c, ok := colors["string"]; ok {
		theme.String = c
	} else {
		theme.String = color.New(color.FgGreen)
	}

	if c, ok := colors["comment"]; ok {
		theme.Comment = c
	} else {
		theme.Comment = color.New(color.FgBlue)
	}

	if c, ok := colors["function"]; ok {
		theme.Function = c
	} else {
		theme.Function = color.New(color.FgYellow, color.Bold)
	}

	if c, ok := colors["variable"]; ok {
		theme.Variable = c
	} else {
		theme.Variable = color.New(color.FgCyan)
	}

	if c, ok := colors["number"]; ok {
		theme.Number = c
	} else {
		theme.Number = color.New(color.FgRed)
	}

	if c, ok := colors["type"]; ok {
		theme.Type = c
	} else {
		theme.Type = color.New(color.FgGreen, color.Bold)
	}

	if c, ok := colors["operator"]; ok {
		theme.Operator = c
	} else {
		theme.Operator = color.New(color.FgWhite, color.Bold)
	}

	if c, ok := colors["bracket"]; ok {
		theme.Bracket = c
	} else {
		theme.Bracket = color.New(color.FgYellow)
	}

	if c, ok := colors["import"]; ok {
		theme.Import = c
	} else {
		theme.Import = color.New(color.FgMagenta)
	}

	sh.themes[name] = theme
}
