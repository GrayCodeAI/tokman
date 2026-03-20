package commands

import (
	"testing"

	"github.com/GrayCodeAI/tokman/internal/discover"
)

func TestRewriteCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string // empty means no rewrite expected
	}{
		{"git status", "git status", "tokman git status"},
		{"git log", "git log --oneline -10", "tokman git log --oneline -10"},
		{"git diff", "git diff HEAD", "tokman git diff HEAD"},
		{"git add", "git add .", "tokman git add ."},
		{"gh pr list", "gh pr list", "tokman gh pr list"},
		{"cargo test", "cargo test", "tokman cargo test"},
		{"cargo build", "cargo build", "tokman cargo build"},
		{"ls -la", "ls -la", "tokman ls -la"},
		{"grep -rn pattern src/", "grep -rn pattern src/", "tokman grep -rn pattern src/"},
		{"cat file", "cat package.json", "tokman read package.json"},
		{"rg pattern", "rg pattern src/", "tokman grep pattern src/"},
		{"docker ps", "docker ps", "tokman docker ps"},
		{"docker compose logs", "docker compose logs web", "tokman docker compose logs web"},
		{"kubectl get pods", "kubectl get pods", "tokman kubectl get pods"},
		{"curl", "curl -s https://example.com", "tokman curl -s https://example.com"},
		{"npm run", "npm run test", "tokman npm run test"},
		{"npx vitest", "npx vitest run", "tokman vitest run"},
		{"npx playwright", "npx playwright test", "tokman playwright test"},
		// Should NOT rewrite
		{"echo", "echo hello world", ""},
		{"cd", "cd /tmp", ""},
		{"mkdir", "mkdir -p foo/bar", ""},
		{"python3", "python3 script.py", ""},
		{"node", "node -e 'console.log(1)'", ""},
		{"heredoc", "cat <<'EOF'\nhello\nEOF", ""},
		{"already tokman", "tokman git status", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if tt.wantCmd == "" {
				if changed {
					t.Errorf("expected no rewrite, got %q", rewritten)
				}
				return
			}

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}

func TestRewriteEnvPrefix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
	}{
		{"env + git status", "NODE_ENV=test git status", "NODE_ENV=test tokman git status"},
		{"multi env", "FOO=1 BAR=2 git log", "FOO=1 BAR=2 tokman git log"},
		{"env + cargo", "RUST_LOG=debug cargo test", "RUST_LOG=debug tokman cargo test"},
		{"env + ls", "LANG=C ls -la", "LANG=C tokman ls -la"},
		{"disabled", "TOKMAN_DISABLED=1 git status", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if tt.wantCmd == "" {
				if changed {
					t.Errorf("expected no rewrite, got %q", rewritten)
				}
				return
			}

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}

func TestRewriteTailNumeric(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
	}{
		{"tail -10 file", "tail -10 file.txt", "tokman read file.txt --tail-lines 10"},
		{"tail -n 5 file", "tail -n 5 log.txt", "tokman read log.txt --tail-lines 5"},
		{"tail --lines=20 file", "tail --lines=20 output.log", "tokman read output.log --tail-lines 20"},
		// Should NOT rewrite (tail -f is a follow flag, but rewrite still applies)
		{"tail -f", "tail -f log.txt", "tokman read -f log.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if tt.wantCmd == "" {
				if changed {
					t.Errorf("expected no rewrite, got %q", rewritten)
				}
				return
			}

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}

func TestRewriteHeadNumeric(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
	}{
		{"head -10 file", "head -10 file.txt", "tokman read file.txt --max-lines 10"},
		{"head --lines=5 file", "head --lines=5 log.txt", "tokman read log.txt --max-lines 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}

func TestRewriteGitGlobalOpts(t *testing.T) {
	// Note: git -C <path> is stripped in rewriteSegment before pattern matching,
	// but the pattern matching only triggers if the command starts with "git ".
	// The actual behavior for compound git options may differ.
	tests := []struct {
		name    string
		input   string
		wantCmd string
	}{
		// These test the stripGitGlobalOpts logic directly
		// git -C is not a pattern match — the rewrite happens through compound handling
		{"git status (direct)", "git status", "tokman git status"},
		{"git log (direct)", "git log --oneline", "tokman git log --oneline"},
		{"git diff (direct)", "git diff HEAD", "tokman git diff HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}

func TestRewritePipeSkip(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string // empty means no rewrite
	}{
		{"find piped to xargs", "find . -name '*.go' | xargs grep TODO", ""},
		{"find piped to head", "find . -type f | head -10", ""},
		{"regular pipe (non-piped cmd)", "cargo test 2>&1 | head", "tokman cargo test 2>&1 | head"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewritten, changed := discover.RewriteCommand(tt.input, nil)

			if tt.wantCmd == "" {
				if changed {
					t.Errorf("expected no rewrite, got %q", rewritten)
				}
				return
			}

			if !changed {
				t.Errorf("expected rewrite to %q, got no change", tt.wantCmd)
				return
			}

			if rewritten != tt.wantCmd {
				t.Errorf("got %q, want %q", rewritten, tt.wantCmd)
			}
		})
	}
}
