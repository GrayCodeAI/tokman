package discover

import (
	"fmt"
	"regexp"
	"strings"
)

// TokmanStatus represents the status of a command rewrite
type TokmanStatus int

const (
	StatusExisting TokmanStatus = iota
	StatusPassthrough
	StatusNew
)

// TokmanRule defines how a command pattern should be rewritten
type TokmanRule struct {
	TokManCmd       string
	RewritePrefixes []string
	Category        string
	SavingsPct      float64
	SubcmdSavings   map[string]float64
	SubcmdStatus    map[string]TokmanStatus
}

// Classification represents the result of classifying a command
type Classification struct {
	Supported   bool
	TokManCmd   string
	Category    string
	SavingsPct  float64
	Status      TokmanStatus
	BaseCommand string // For unsupported commands
}

var (
	// Compiled regex patterns for command matching
	patterns = []*regexp.Regexp{
		regexp.MustCompile(`^git\s+(status|log|diff|show|add|commit|push|pull|branch|fetch|stash|worktree)`),
		regexp.MustCompile(`^gh\s+(pr|issue|run|repo|api|release)`),
		regexp.MustCompile(`^cargo\s+(build|test|clippy|check|fmt|install)`),
		regexp.MustCompile(`^pnpm\s+(list|ls|outdated|install)`),
		regexp.MustCompile(`^npm\s+(run|exec)`),
		regexp.MustCompile(`^npx\s+`),
		regexp.MustCompile(`^(cat|head|tail)\s+`),
		regexp.MustCompile(`^(rg|grep)\s+`),
		regexp.MustCompile(`^ls(\s|$)`),
		regexp.MustCompile(`^find\s+`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?tsc(\s|$)`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?(eslint|biome|lint)(\s|$)`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?prettier`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?next\s+build`),
		regexp.MustCompile(`^(pnpm\s+|npx\s+)?(vitest|jest|test)(\s|$)`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?playwright`),
		regexp.MustCompile(`^(npx\s+|pnpm\s+)?prisma`),
		regexp.MustCompile(`^docker\s+(ps|images|logs|run|exec|build|compose\s+(ps|logs|build))`),
		regexp.MustCompile(`^kubectl\s+(get|logs|describe|apply)`),
		regexp.MustCompile(`^tree(\s|$)`),
		regexp.MustCompile(`^diff\s+`),
		regexp.MustCompile(`^curl\s+`),
		regexp.MustCompile(`^wget\s+`),
		regexp.MustCompile(`^(python3?\s+-m\s+)?mypy(\s|$)`),
		regexp.MustCompile(`^ruff\s+(check|format)`),
		regexp.MustCompile(`^(python\s+-m\s+)?pytest(\s|$)`),
		regexp.MustCompile(`^(pip3?|uv\s+pip)\s+(list|outdated|install)`),
		regexp.MustCompile(`^go\s+(test|build|vet)`),
		regexp.MustCompile(`^golangci-lint(\s|$)`),
		regexp.MustCompile(`^aws\s+`),
		regexp.MustCompile(`^psql(\s|$)`),
	}

	// Rules corresponding to patterns (index-aligned)
	rules = []TokmanRule{
		{TokManCmd: "tokman git", RewritePrefixes: []string{"git"}, Category: "Git", SavingsPct: 70.0, SubcmdSavings: map[string]float64{"diff": 80.0, "show": 80.0, "add": 59.0, "commit": 59.0}},
		{TokManCmd: "tokman gh", RewritePrefixes: []string{"gh"}, Category: "GitHub", SavingsPct: 82.0, SubcmdSavings: map[string]float64{"pr": 87.0, "run": 82.0, "issue": 80.0}},
		{TokManCmd: "tokman cargo", RewritePrefixes: []string{"cargo"}, Category: "Cargo", SavingsPct: 80.0, SubcmdSavings: map[string]float64{"test": 90.0, "check": 80.0}, SubcmdStatus: map[string]TokmanStatus{"fmt": StatusPassthrough}},
		{TokManCmd: "tokman pnpm", RewritePrefixes: []string{"pnpm"}, Category: "PackageManager", SavingsPct: 80.0},
		{TokManCmd: "tokman npm", RewritePrefixes: []string{"npm"}, Category: "PackageManager", SavingsPct: 70.0},
		{TokManCmd: "tokman npx", RewritePrefixes: []string{"npx"}, Category: "PackageManager", SavingsPct: 70.0},
		{TokManCmd: "tokman read", RewritePrefixes: []string{"cat", "head", "tail"}, Category: "Files", SavingsPct: 60.0},
		{TokManCmd: "tokman grep", RewritePrefixes: []string{"rg", "grep"}, Category: "Files", SavingsPct: 75.0},
		{TokManCmd: "tokman ls", RewritePrefixes: []string{"ls"}, Category: "Files", SavingsPct: 65.0},
		{TokManCmd: "tokman find", RewritePrefixes: []string{"find"}, Category: "Files", SavingsPct: 70.0},
		{TokManCmd: "tokman tsc", RewritePrefixes: []string{"pnpm tsc", "npx tsc", "tsc"}, Category: "Build", SavingsPct: 83.0},
		{TokManCmd: "tokman lint", RewritePrefixes: []string{"npx eslint", "pnpm lint", "npx biome", "eslint", "biome", "lint"}, Category: "Build", SavingsPct: 84.0},
		{TokManCmd: "tokman prettier", RewritePrefixes: []string{"npx prettier", "pnpm prettier", "prettier"}, Category: "Build", SavingsPct: 70.0},
		{TokManCmd: "tokman next", RewritePrefixes: []string{"npx next build", "pnpm next build", "next build"}, Category: "Build", SavingsPct: 87.0},
		{TokManCmd: "tokman vitest", RewritePrefixes: []string{"pnpm vitest", "npx vitest", "vitest", "jest"}, Category: "Tests", SavingsPct: 99.0},
		{TokManCmd: "tokman playwright", RewritePrefixes: []string{"npx playwright", "pnpm playwright", "playwright"}, Category: "Tests", SavingsPct: 94.0},
		{TokManCmd: "tokman prisma", RewritePrefixes: []string{"npx prisma", "pnpm prisma", "prisma"}, Category: "Build", SavingsPct: 88.0},
		{TokManCmd: "tokman docker", RewritePrefixes: []string{"docker"}, Category: "Infra", SavingsPct: 85.0},
		{TokManCmd: "tokman kubectl", RewritePrefixes: []string{"kubectl"}, Category: "Infra", SavingsPct: 85.0},
		{TokManCmd: "tokman tree", RewritePrefixes: []string{"tree"}, Category: "Files", SavingsPct: 70.0},
		{TokManCmd: "tokman diff", RewritePrefixes: []string{"diff"}, Category: "Files", SavingsPct: 60.0},
		{TokManCmd: "tokman curl", RewritePrefixes: []string{"curl"}, Category: "Network", SavingsPct: 70.0},
		{TokManCmd: "tokman wget", RewritePrefixes: []string{"wget"}, Category: "Network", SavingsPct: 65.0},
		{TokManCmd: "tokman mypy", RewritePrefixes: []string{"python3 -m mypy", "python -m mypy", "mypy"}, Category: "Build", SavingsPct: 80.0},
		{TokManCmd: "tokman ruff", RewritePrefixes: []string{"ruff"}, Category: "Python", SavingsPct: 80.0, SubcmdSavings: map[string]float64{"check": 80.0, "format": 75.0}},
		{TokManCmd: "tokman pytest", RewritePrefixes: []string{"python -m pytest", "pytest"}, Category: "Python", SavingsPct: 90.0},
		{TokManCmd: "tokman pip", RewritePrefixes: []string{"pip3", "pip", "uv pip"}, Category: "Python", SavingsPct: 75.0, SubcmdSavings: map[string]float64{"list": 75.0, "outdated": 80.0}},
		{TokManCmd: "tokman go", RewritePrefixes: []string{"go"}, Category: "Go", SavingsPct: 85.0, SubcmdSavings: map[string]float64{"test": 90.0, "build": 80.0, "vet": 75.0}},
		{TokManCmd: "tokman golangci-lint", RewritePrefixes: []string{"golangci-lint", "golangci"}, Category: "Go", SavingsPct: 85.0},
		{TokManCmd: "tokman aws", RewritePrefixes: []string{"aws"}, Category: "Infra", SavingsPct: 80.0},
		{TokManCmd: "tokman psql", RewritePrefixes: []string{"psql"}, Category: "Infra", SavingsPct: 75.0},
	}

	// Ignored prefixes (shell builtins, trivial commands)
	ignoredPrefixes = []string{
		"cd ", "cd\t", "echo ", "printf ", "export ", "source ", "mkdir ", "rm ", "mv ", "cp ",
		"chmod ", "chown ", "touch ", "which ", "type ", "command ", "test ", "true", "false",
		"sleep ", "wait", "kill ", "set ", "unset ", "wc ", "sort ", "uniq ", "tr ", "cut ",
		"awk ", "sed ", "python3 -c", "python -c", "node -e", "ruby -e", "tokman ", "pwd",
		"bash ", "sh ", "then\n", "then ", "else\n", "else ", "do\n", "do ", "for ", "while ", "if ", "case ",
	}

	// Ignored exact matches
	ignoredExact = map[string]bool{
		"cd": true, "echo": true, "true": true, "false": true, "wait": true,
		"pwd": true, "bash": true, "sh": true, "fi": true, "done": true,
	}

	// Env prefix regex (sudo, env VAR=val, VAR=val)
	envPrefixRegex = regexp.MustCompile(`^(?:sudo\s+|env\s+|[A-Z_][A-Z0-9_]*=[^\s]*\s+)+`)
)

// ClassifyCommand classifies a single command
func ClassifyCommand(cmd string) Classification {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return Classification{}
	}

	// Check ignored exact
	if ignoredExact[trimmed] {
		return Classification{}
	}

	// Check ignored prefixes
	for _, prefix := range ignoredPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return Classification{}
		}
	}

	// Strip env prefixes
	stripped := envPrefixRegex.ReplaceAllString(trimmed, "")
	cmdClean := strings.TrimSpace(stripped)
	if cmdClean == "" {
		return Classification{}
	}

	// Check if already tokman
	if strings.HasPrefix(cmdClean, "tokman ") || cmdClean == "tokman" {
		return Classification{}
	}

	// Try pattern matching (take last/most specific match)
	lastMatch := -1
	for i, pattern := range patterns {
		if pattern.MatchString(cmdClean) {
			lastMatch = i
		}
	}

	if lastMatch >= 0 {
		rule := rules[lastMatch]
		savings := rule.SavingsPct
		status := StatusExisting

		// Extract subcommand for savings/status override
		matches := patterns[lastMatch].FindStringSubmatch(cmdClean)
		if len(matches) > 1 {
			subcmd := matches[1]
			if s, ok := rule.SubcmdSavings[subcmd]; ok {
				savings = s
			}
			if st, ok := rule.SubcmdStatus[subcmd]; ok {
				status = st
			}
		}

		return Classification{
			Supported:  true,
			TokManCmd:  rule.TokManCmd,
			Category:   rule.Category,
			SavingsPct: savings,
			Status:     status,
		}
	}

	// Unsupported - extract base command
	base := extractBaseCommand(cmdClean)
	return Classification{
		Supported:   false,
		BaseCommand: base,
	}
}

// extractBaseCommand extracts the base command (first word or first two if subcommand)
func extractBaseCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// If second token looks like a subcommand (no leading -)
	if !strings.HasPrefix(parts[1], "-") && !strings.Contains(parts[1], "/") && !strings.Contains(parts[1], ".") {
		return parts[0] + " " + parts[1]
	}
	return parts[0]
}

// RewriteCommand rewrites a command to its TokMan equivalent.
// Returns the rewritten command and true if rewritten, or original and false if not.
// Handles compound commands (&&, ||, ;, |) by rewriting each segment.
func RewriteCommand(cmd string, excluded []string) (string, bool) {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return "", false
	}

	// Heredoc or arithmetic expansion — unsafe to split/rewrite
	if strings.Contains(trimmed, "<<") || strings.Contains(trimmed, "$((") {
		return "", false
	}

	// Check for compound operators
	hasCompound := strings.Contains(trimmed, "&&") ||
		strings.Contains(trimmed, "||") ||
		strings.Contains(trimmed, ";") ||
		strings.Contains(trimmed, "|") ||
		strings.Contains(trimmed, " & ")

	// Simple (non-compound) already-tokman command — return as-is
	if !hasCompound && (strings.HasPrefix(trimmed, "tokman ") || trimmed == "tokman") {
		return trimmed, false // Already tokman, no change needed
	}

	return rewriteCompound(trimmed, excluded)
}

// rewriteCompound handles compound commands with &&, ||, ;, |
func rewriteCompound(cmd string, excluded []string) (string, bool) {
	result := ""
	anyChanged := false
	segStart := 0
	inSingle := false
	inDouble := false
	bytes := []byte(cmd)

	for i := 0; i < len(bytes); i++ {
		b := bytes[i]
		switch {
		case b == '\'' && !inDouble:
			inSingle = !inSingle
		case b == '"' && !inSingle:
			inDouble = !inDouble
		case b == '|' && !inSingle && !inDouble:
			if i+1 < len(bytes) && bytes[i+1] == '|' {
				// || operator
				seg := strings.TrimSpace(cmd[segStart:i])
				rewritten, changed := rewriteSegment(seg, excluded)
				if changed {
					anyChanged = true
				}
				result += rewritten + " || "
				i += 2
				for i < len(bytes) && bytes[i] == ' ' {
					i++
				}
				segStart = i
			} else {
				// | pipe — rewrite first segment only, pass through rest
				// Skip rewriting if the command uses piped output format (find, fd, etc.)
				seg := strings.TrimSpace(cmd[segStart:i])
				if isInPipe(cmd, segStart, i) {
					result += seg + " " + strings.TrimSpace(cmd[i:])
					return result, anyChanged
				}
				rewritten, changed := rewriteSegment(seg, excluded)
				if changed {
					anyChanged = true
				}
				result += rewritten + " " + strings.TrimSpace(cmd[i:])
				return result, anyChanged
			}
		case b == '&' && !inSingle && !inDouble && i+1 < len(bytes) && bytes[i+1] == '&':
			// && operator
			seg := strings.TrimSpace(cmd[segStart:i])
			rewritten, changed := rewriteSegment(seg, excluded)
			if changed {
				anyChanged = true
			}
			result += rewritten + " && "
			i += 2
			for i < len(bytes) && bytes[i] == ' ' {
				i++
			}
			segStart = i
		case b == '&' && !inSingle && !inDouble:
			// Check for redirect (2>&1 or &>)
			isRedirect := (i > 0 && bytes[i-1] == '>') || (i+1 < len(bytes) && bytes[i+1] == '>')
			if !isRedirect {
				// Background execution
				seg := strings.TrimSpace(cmd[segStart:i])
				rewritten, changed := rewriteSegment(seg, excluded)
				if changed {
					anyChanged = true
				}
				result += rewritten + " & "
				i++
				for i < len(bytes) && bytes[i] == ' ' {
					i++
				}
				segStart = i
			}
		case b == ';' && !inSingle && !inDouble:
			seg := strings.TrimSpace(cmd[segStart:i])
			rewritten, changed := rewriteSegment(seg, excluded)
			if changed {
				anyChanged = true
			}
			result += rewritten + ";"
			i++
			if i < len(bytes) && bytes[i] == ' ' {
				result += " "
			}
			segStart = i
		}
	}

	// Last segment
	seg := strings.TrimSpace(cmd[segStart:])
	rewritten, changed := rewriteSegment(seg, excluded)
	if changed {
		anyChanged = true
	}
	result += rewritten

	return result, anyChanged
}

// rewriteHeadNumeric handles head -N file → tokman read file --max-lines N
// Returns (rewritten, true) if matched, or ("", false) to fall through to generic logic
func rewriteHeadNumeric(envPrefix, cmdClean string) (string, bool) {
	// Match: head -<digits> <file>
	headNumeric := regexp.MustCompile(`^head\s+-(\d+)\s+(.+)$`)
	// Match: head --lines=<digits> <file>
	headLines := regexp.MustCompile(`^head\s+--lines=(\d+)\s+(.+)$`)

	if matches := headNumeric.FindStringSubmatch(cmdClean); len(matches) == 3 {
		n := matches[1]
		file := matches[2]
		return fmt.Sprintf("%stokman read %s --max-lines %s", envPrefix, file, n), true
	}
	if matches := headLines.FindStringSubmatch(cmdClean); len(matches) == 3 {
		n := matches[1]
		file := matches[2]
		return fmt.Sprintf("%stokman read %s --max-lines %s", envPrefix, file, n), true
	}
	// head with any other flag (e.g. -c, -q): skip rewriting to avoid clap errors
	if strings.HasPrefix(cmdClean, "head -") {
		return "", false
	}
	return "", false
}

// rewriteTailNumeric handles tail -N file → tokman read file --tail-lines N
// Returns (rewritten, true) if matched, or ("", false) to fall through to generic logic
func rewriteTailNumeric(envPrefix, cmdClean string) (string, bool) {
	// Match: tail -<digits> <file>
	tailNumeric := regexp.MustCompile(`^tail\s+-(\d+)\s+(.+)$`)
	// Match: tail -n <digits> <file>
	tailN := regexp.MustCompile(`^tail\s+-n\s+(\d+)\s+(.+)$`)
	// Match: tail --lines=<digits> <file>
	tailLines := regexp.MustCompile(`^tail\s+--lines=(\d+)\s+(.+)$`)

	if matches := tailNumeric.FindStringSubmatch(cmdClean); len(matches) == 3 {
		n := matches[1]
		file := matches[2]
		return fmt.Sprintf("%stokman read %s --tail-lines %s", envPrefix, file, n), true
	}
	if matches := tailN.FindStringSubmatch(cmdClean); len(matches) == 3 {
		n := matches[1]
		file := matches[2]
		return fmt.Sprintf("%stokman read %s --tail-lines %s", envPrefix, file, n), true
	}
	if matches := tailLines.FindStringSubmatch(cmdClean); len(matches) == 3 {
		n := matches[1]
		file := matches[2]
		return fmt.Sprintf("%stokman read %s --tail-lines %s", envPrefix, file, n), true
	}
	// tail with any other flag (e.g. -f, -c, -q): skip rewriting
	if strings.HasPrefix(cmdClean, "tail -") {
		return "", false
	}
	return "", false
}

// gitGlobalOpts are git options that apply globally (before the subcommand)
var gitGlobalOpts = []string{"-C", "-c", "--git-dir", "--work-tree", "--no-pager", "--no-optional-locks", "--bare", "--literal-pathspecs"}

// stripGitGlobalOpts removes git global options from the command.
// e.g. "git -C /tmp status" → "git status"
func stripGitGlobalOpts(cmdClean string) string {
	if !strings.HasPrefix(cmdClean, "git ") {
		return cmdClean
	}
	parts := strings.Fields(cmdClean)
	if len(parts) <= 2 {
		return cmdClean
	}
	filtered := []string{parts[0]} // keep "git"
	i := 1
	for i < len(parts) {
		part := parts[i]
		isGlobalOpt := false
		for _, opt := range gitGlobalOpts {
			if part == opt {
				isGlobalOpt = true
				// -C and -c take a value argument — skip next token too
				if (opt == "-C" || opt == "-c") && i+1 < len(parts) {
					i += 2
				} else {
					i++
				}
				break
			}
		}
		if !isGlobalOpt {
			filtered = append(filtered, parts[i])
			i++
		}
	}
	return strings.Join(filtered, " ")
}

// absolutePathRegex matches absolute paths like /usr/bin/grep, /usr/local/bin/rg
var absolutePathRegex = regexp.MustCompile(`^(/[a-zA-Z0-9._-]+)+/([a-zA-Z0-9._-]+)(\s|$)`)

// stripAbsolutePath normalizes absolute binary paths to just the command name.
// e.g. "/usr/bin/grep foo" → "grep foo"
func stripAbsolutePath(cmdClean string) string {
	if matches := absolutePathRegex.FindStringSubmatch(cmdClean); len(matches) >= 3 {
		binary := matches[2]
		// The absolute path was the first token, replace it with just the binary name
		firstSpace := strings.IndexByte(cmdClean, ' ')
		if firstSpace > 0 {
			return binary + cmdClean[firstSpace:]
		}
		return binary
	}
	return cmdClean
}

// pipedCommands are commands whose output format is incompatible with piping (e.g. to xargs, grep).
// When these appear before a pipe, we skip rewriting to avoid breaking the pipeline.
var pipedCommands = map[string]bool{
	"find": true, "fd": true, "locate": true,
}

// isInPipe checks if the command segment is the first part of a pipe and uses a piped command.
func isInPipe(fullCmd string, segStart int, pipePos int) bool {
	seg := strings.TrimSpace(fullCmd[segStart:pipePos])
	parts := strings.Fields(seg)
	if len(parts) == 0 {
		return false
	}
	return pipedCommands[parts[0]]
}

// rewriteSegment rewrites a single command segment
func rewriteSegment(seg string, excluded []string) (string, bool) {
	trimmed := strings.TrimSpace(seg)
	if trimmed == "" {
		return seg, false
	}

	// Already tokman — pass through
	if strings.HasPrefix(trimmed, "tokman ") || trimmed == "tokman" {
		return trimmed, false
	}

	// Extract env prefix
	envPrefix := envPrefixRegex.FindString(trimmed)
	cmdClean := strings.TrimPrefix(trimmed, envPrefix)
	cmdClean = strings.TrimSpace(cmdClean)

	// Check TOKMAN_DISABLED
	if strings.Contains(envPrefix, "TOKMAN_DISABLED=") {
		return seg, false
	}

	// Strip git global options (e.g. "git -C /tmp status" → "git status")
	cmdClean = stripGitGlobalOpts(cmdClean)

	// Strip absolute binary paths (e.g. "/usr/bin/grep foo" → "grep foo")
	cmdClean = stripAbsolutePath(cmdClean)

	// Special case: head -N file → tokman read file --max-lines N
	// Must intercept before generic prefix replacement (which would produce tokman read -N file)
	if rewritten, ok := rewriteHeadNumeric(envPrefix, cmdClean); ok {
		return rewritten, true
	}

	// Special case: tail -N file → tokman read file --tail-lines N
	if rewritten, ok := rewriteTailNumeric(envPrefix, cmdClean); ok {
		return rewritten, true
	}

	// Classify command
	class := ClassifyCommand(trimmed)
	if !class.Supported {
		return seg, false
	}

	// Check if excluded
	base := strings.Fields(cmdClean)
	if len(base) > 0 {
		for _, e := range excluded {
			if e == base[0] {
				return seg, false
			}
		}
	}

	// Find the matching rule
	var rule *TokmanRule
	for i := range rules {
		if rules[i].TokManCmd == class.TokManCmd {
			rule = &rules[i]
			break
		}
	}
	if rule == nil {
		return seg, false
	}

	// #196: gh with --json/--jq/--template produces structured output
	if rule.TokManCmd == "tokman gh" {
		argsLower := strings.ToLower(cmdClean)
		if strings.Contains(argsLower, "--json") || strings.Contains(argsLower, "--jq") || strings.Contains(argsLower, "--template") {
			return seg, false
		}
	}

	// Try rewrite prefixes (longest first)
	for _, prefix := range rule.RewritePrefixes {
		if rest := stripWordPrefix(cmdClean, prefix); rest != nil {
			rewritten := envPrefix + rule.TokManCmd
			if *rest != "" {
				rewritten += " " + *rest
			}
			return rewritten, true
		}
	}

	return seg, false
}

// stripWordPrefix strips a command prefix with word-boundary check
func stripWordPrefix(cmd, prefix string) *string {
	if cmd == prefix {
		empty := ""
		return &empty
	}
	if len(cmd) > len(prefix) && strings.HasPrefix(cmd, prefix) && cmd[len(prefix)] == ' ' {
		rest := strings.TrimSpace(cmd[len(prefix)+1:])
		return &rest
	}
	return nil
}

// Rewrite is the legacy API for backward compatibility
func Rewrite(command string) string {
	rewritten, _ := RewriteCommand(command, nil)
	if rewritten == "" {
		return command
	}
	return rewritten
}

// ShouldRewrite returns true if a command should be rewritten
func ShouldRewrite(command string) bool {
	_, changed := RewriteCommand(command, nil)
	return changed
}

// GetMapping returns the mapping for a command if one exists (legacy API)
func GetMapping(command string) (CommandMapping, bool) {
	class := ClassifyCommand(command)
	if class.Supported {
		return CommandMapping{
			Original:  command,
			TokManCmd: class.TokManCmd,
			Enabled:   true,
			PassArgs:  true,
		}, true
	}
	return CommandMapping{}, false
}

// ListRewrites returns all enabled rewrites
func ListRewrites() []CommandMapping {
	var rewrites []CommandMapping
	for _, rule := range rules {
		rewrites = append(rewrites, CommandMapping{
			Original:  rule.RewritePrefixes[0],
			TokManCmd: rule.TokManCmd,
			Enabled:   true,
			PassArgs:  true,
		})
	}
	return rewrites
}

// CommandMapping defines how a command should be rewritten (legacy API)
type CommandMapping struct {
	Original  string
	TokManCmd string
	Enabled   bool
	PassArgs  bool
}

// Registry is kept for backward compatibility (legacy API)
var Registry = map[string]CommandMapping{}
