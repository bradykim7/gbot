package commands

import (
	"strings"

	"github.com/bradykim7/gbot/pkg/logger"
	"github.com/bwmarrin/discordgo"
)

// Command represents a bot command
type Command interface {
	Execute(s *discordgo.Session, m *discordgo.MessageCreate, args []string)
	Help() string
}

// Registry manages all bot commands
type Registry struct {
	prefix   string
	commands map[string]Command
	log      *logger.Logger
}

// NewRegistry creates a new command registry
func NewRegistry(prefix string) *Registry {
	return &Registry{
		prefix:   prefix,
		commands: make(map[string]Command),
		log:      logger.New("commands"),
	}
}

// Register registers a command with the registry
func (r *Registry) Register(name string, cmd Command) {
	r.commands[name] = cmd
	r.log.Infof("Registered command: %s", name)
}

// Handle processes a message and executes the appropriate command
func (r *Registry) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if the message starts with the command prefix
	if !strings.HasPrefix(m.Content, r.prefix) {
		return
	}
	
	// Split the message into command and arguments
	content := strings.TrimPrefix(m.Content, r.prefix)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}
	
	// Extract command name and arguments
	cmdName := parts[0]
	args := parts[1:]
	
	// Find the command
	cmd, ok := r.commands[cmdName]
	if !ok {
		return
	}
	
	// Execute the command
	r.log.Infof("Executing command: %s", cmdName)
	cmd.Execute(s, m, args)
}

// GetCommands returns all registered commands
func (r *Registry) GetCommands() map[string]Command {
	return r.commands
}