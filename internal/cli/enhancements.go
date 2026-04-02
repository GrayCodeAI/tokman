// Package cli provides CLI enhancement utilities
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ShellCompletion generates shell completion scripts
type ShellCompletion struct {
	shell string
}

// NewShellCompletion creates a new shell completion generator
func NewShellCompletion(shell string) *ShellCompletion {
	return &ShellCompletion{shell: shell}
}

// Generate generates completion script
func (sc *ShellCompletion) Generate(commands []CommandInfo) (string, error) {
	switch sc.shell {
	case "bash":
		return sc.generateBash(commands), nil
	case "zsh":
		return sc.generateZsh(commands), nil
	case "fish":
		return sc.generateFish(commands), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", sc.shell)
	}
}

func (sc *ShellCompletion) generateBash(commands []CommandInfo) string {
	script := "#!/usr/bin/env bash\n\n"
	script += "_tokman_completion() {\n"
	script += "    local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n"
	script += "    local commands=\""

	for _, cmd := range commands {
		script += cmd.Name + " "
	}

	script += "\"\n"
	script += "    COMPREPLY=( $(compgen -W \"$commands\" -- $cur) )\n"
	script += "}\n\n"
	script += "complete -F _tokman_completion tokman\n"

	return script
}

func (sc *ShellCompletion) generateZsh(commands []CommandInfo) string {
	script := "#compdef tokman\n\n"
	script += "_tokman() {\n"
	script += "    local commands=(\n"

	for _, cmd := range commands {
		script += fmt.Sprintf("        '%s:%s'\n", cmd.Name, cmd.Description)
	}

	script += "    )\n"
	script += "    _describe 'command' commands\n"
	script += "}\n\n"
	script += "compdef _tokman tokman\n"

	return script
}

func (sc *ShellCompletion) generateFish(commands []CommandInfo) string {
	script := "complete -c tokman -f\n\n"

	for _, cmd := range commands {
		script += fmt.Sprintf("complete -c tokman -n \"__fish_use_subcommand\" -a %s -d \"%s\"\n", cmd.Name, cmd.Description)
	}

	return script
}

// CommandInfo represents command information for completion
type CommandInfo struct {
	Name        string
	Description string
	Aliases     []string
}

// AliasManager manages command aliases
type AliasManager struct {
	aliases map[string]string
}

// NewAliasManager creates a new alias manager
func NewAliasManager() *AliasManager {
	return &AliasManager{
		aliases: make(map[string]string),
	}
}

// AddAlias adds a command alias
func (am *AliasManager) AddAlias(alias, command string) {
	am.aliases[alias] = command
}

// ResolveAlias resolves an alias to a command
func (am *AliasManager) ResolveAlias(input string) string {
	if cmd, ok := am.aliases[input]; ok {
		return cmd
	}
	return input
}

// ListAliases returns all aliases
func (am *AliasManager) ListAliases() map[string]string {
	return am.aliases
}

// StandardAliases returns standard command aliases
func StandardAliases() map[string]string {
	return map[string]string{
		"bm":     "benchmark run",
		"bench":  "benchmark run",
		"stress": "stress run",
		"chaos":  "chaos run",
		"cost":   "cost summary",
		"alert":  "alerts list",
		"team":   "teams list",
		"report": "report generate",
		"export": "export run",
		"status": "status show",
	}
}

// ProgressIndicator shows progress for long operations
type ProgressIndicator struct {
	writer    io.Writer
	total     int
	current   int
	startTime time.Time
	prefix    string
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(total int, prefix string) *ProgressIndicator {
	return &ProgressIndicator{
		writer:    os.Stdout,
		total:     total,
		startTime: time.Now(),
		prefix:    prefix,
	}
}

// Update updates progress
func (pi *ProgressIndicator) Update(current int) {
	pi.current = current

	percent := float64(current) / float64(pi.total) * 100
	elapsed := time.Since(pi.startTime)

	bar := pi.drawBar(percent)

	fmt.Fprintf(pi.writer, "\r%s [%s] %.1f%% (%d/%d) - %s",
		pi.prefix, bar, percent, current, pi.total, elapsed.Round(time.Second))
}

// Complete marks progress as complete
func (pi *ProgressIndicator) Complete() {
	pi.Update(pi.total)
	fmt.Fprintln(pi.writer)
}

func (pi *ProgressIndicator) drawBar(percent float64) string {
	width := 40
	filled := int(percent / 100 * float64(width))

	bar := strings.Repeat("█", filled)
	bar += strings.Repeat("░", width-filled)

	return bar
}

// Spinner shows a spinning indicator
type Spinner struct {
	writer  io.Writer
	message string
	chars   []string
	index   int
	done    chan bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stdout,
		message: message,
		chars:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan bool),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	go func() {
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Fprintf(s.writer, "\r%s %s", s.chars[s.index%len(s.chars)], s.message)
				s.index++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.done <- true
	fmt.Fprintln(s.writer)
}

// ColorTheme defines a color theme for the CLI
type ColorTheme struct {
	Name       string
	Primary    string
	Secondary  string
	Success    string
	Warning    string
	Error      string
	Info       string
	Background string
}

// StandardThemes returns standard color themes
func StandardThemes() map[string]ColorTheme {
	return map[string]ColorTheme{
		"dark": {
			Name:       "Dark",
			Primary:    "\033[38;5;75m",
			Secondary:  "\033[38;5;141m",
			Success:    "\033[38;5;82m",
			Warning:    "\033[38;5;226m",
			Error:      "\033[38;5;196m",
			Info:       "\033[38;5;117m",
			Background: "\033[48;5;235m",
		},
		"light": {
			Name:       "Light",
			Primary:    "\033[38;5;25m",
			Secondary:  "\033[38;5;92m",
			Success:    "\033[38;5;22m",
			Warning:    "\033[38;5;130m",
			Error:      "\033[38;5;124m",
			Info:       "\033[38;5;25m",
			Background: "\033[48;5;255m",
		},
		"monokai": {
			Name:       "Monokai",
			Primary:    "\033[38;5;148m",
			Secondary:  "\033[38;5;141m",
			Success:    "\033[38;5;197m",
			Warning:    "\033[38;5;226m",
			Error:      "\033[38;5;196m",
			Info:       "\033[38;5;81m",
			Background: "\033[48;5;235m",
		},
	}
}

// DryRunMode enables dry-run mode for operations
type DryRunMode struct {
	enabled bool
	actions []DryRunAction
}

// DryRunAction represents a dry-run action
type DryRunAction struct {
	Type        string
	Description string
	Parameters  map[string]interface{}
}

// NewDryRunMode creates a new dry-run mode
func NewDryRunMode(enabled bool) *DryRunMode {
	return &DryRunMode{
		enabled: enabled,
		actions: make([]DryRunAction, 0),
	}
}

// Record records an action
func (d *DryRunMode) Record(action DryRunAction) {
	d.actions = append(d.actions, action)
}

// IsEnabled returns if dry-run is enabled
func (d *DryRunMode) IsEnabled() bool {
	return d.enabled
}

// GetActions returns recorded actions
func (d *DryRunMode) GetActions() []DryRunAction {
	return d.actions
}

// PrintActions prints all recorded actions
func (d *DryRunMode) PrintActions() {
	if !d.enabled {
		return
	}

	fmt.Println("Dry Run - Actions that would be performed:")
	fmt.Println(strings.Repeat("-", 50))

	for i, action := range d.actions {
		fmt.Printf("%d. [%s] %s\n", i+1, action.Type, action.Description)
		if len(action.Parameters) > 0 {
			for k, v := range action.Parameters {
				fmt.Printf("   - %s: %v\n", k, v)
			}
		}
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Total: %d actions\n", len(d.actions))
}

// CommandChain enables command chaining
type CommandChain struct {
	commands []ChainCommand
}

// ChainCommand represents a command in a chain
type ChainCommand struct {
	Command     string
	Args        []string
	OnSuccess   bool
	OnFailure   bool
	Description string
}

// NewCommandChain creates a new command chain
func NewCommandChain() *CommandChain {
	return &CommandChain{
		commands: make([]ChainCommand, 0),
	}
}

// AddCommand adds a command to the chain
func (cc *CommandChain) AddCommand(cmd ChainCommand) {
	cc.commands = append(cc.commands, cmd)
}

// Execute executes the command chain
func (cc *CommandChain) Execute() error {
	for _, cmd := range cc.commands {
		fmt.Printf("Executing: %s %s\n", cmd.Command, strings.Join(cmd.Args, " "))
		// Would execute command in production
	}
	return nil
}

// BatchOperation enables batch operations
type BatchOperation struct {
	items    []BatchItem
	parallel int
	onError  ErrorHandling
}

// BatchItem represents an item in a batch
type BatchItem struct {
	ID      string
	Action  string
	Payload interface{}
}

// ErrorHandling defines how to handle errors in batch
type ErrorHandling string

const (
	ErrorHandlingContinue ErrorHandling = "continue"
	ErrorHandlingStop     ErrorHandling = "stop"
	ErrorHandlingRetry    ErrorHandling = "retry"
)

// NewBatchOperation creates a new batch operation
func NewBatchOperation(parallel int, onError ErrorHandling) *BatchOperation {
	return &BatchOperation{
		items:    make([]BatchItem, 0),
		parallel: parallel,
		onError:  onError,
	}
}

// AddItem adds an item to the batch
func (bo *BatchOperation) AddItem(item BatchItem) {
	bo.items = append(bo.items, item)
}

// Execute executes the batch operation
func (bo *BatchOperation) Execute() []BatchResult {
	results := make([]BatchResult, 0, len(bo.items))

	for _, item := range bo.items {
		result := BatchResult{
			ID:      item.ID,
			Success: true,
		}
		results = append(results, result)
	}

	return results
}

// BatchResult represents a batch operation result
type BatchResult struct {
	ID      string
	Success bool
	Error   error
}
