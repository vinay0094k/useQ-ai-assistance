package mcp

import (
	"sync"
)

// CommandRegistry manages available commands
type CommandRegistry struct {
	commands map[string]*CommandDefinition
	mu       sync.RWMutex
}

// SafetyValidator validates command safety
type SafetyValidator struct {
	allowedCommands map[string]bool
	blockedCommands map[string]bool
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*CommandDefinition),
	}
}

// Register registers a new command
func (cr *CommandRegistry) Register(cmd *CommandDefinition) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.commands[cmd.Name] = cmd
}

// Get retrieves a command by name
func (cr *CommandRegistry) Get(name string) *CommandDefinition {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.commands[name]
}

// List returns all registered commands
func (cr *CommandRegistry) List() []*CommandDefinition {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	commands := make([]*CommandDefinition, 0, len(cr.commands))
	for _, cmd := range cr.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// FindByTrigger finds commands that match trigger words
func (cr *CommandRegistry) FindByTrigger(trigger string) []*CommandDefinition {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	var matches []*CommandDefinition
	for _, cmd := range cr.commands {
		for _, cmdTrigger := range cmd.Triggers {
			if cmdTrigger == trigger {
				matches = append(matches, cmd)
				break
			}
		}
	}
	return matches
}

// NewSafetyValidator creates a new safety validator
func NewSafetyValidator() *SafetyValidator {
	return &SafetyValidator{
		allowedCommands: map[string]bool{
			"find":   true,
			"ls":     true,
			"cat":    true,
			"grep":   true,
			"wc":     true,
			"du":     true,
			"ps":     true,
			"git":    true,
			"tree":   true,
			"head":   true,
			"tail":   true,
			"pwd":    true,
			"whoami": true,
		},
		blockedCommands: map[string]bool{
			"rm":     true,
			"rmdir":  true,
			"mv":     true,
			"cp":     true,
			"chmod":  true,
			"chown":  true,
			"sudo":   true,
			"su":     true,
			"kill":   true,
			"killall": true,
		},
	}
}

// ValidateCommand validates if a command is safe to execute
func (sv *SafetyValidator) ValidateCommand(cmd *CommandDefinition) error {
	if sv.blockedCommands[cmd.Command] {
		return fmt.Errorf("command %s is blocked for safety", cmd.Command)
	}
	
	if !sv.allowedCommands[cmd.Command] {
		return fmt.Errorf("command %s is not in allowed list", cmd.Command)
	}
	
	return nil
}