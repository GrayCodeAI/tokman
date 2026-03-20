#!/bin/bash
# Test suite for tokman-rewrite.sh
# Feeds mock JSON through the hook and verifies the rewritten commands.
#
# Usage: bash hooks/test-tokman-rewrite.sh

HOOK="${HOOK:-$HOME/.claude/hooks/tokman-rewrite.sh}"
PASS=0
FAIL=0
TOTAL=0

# Colors
GREEN='\033[32m'
RED='\033[31m'
DIM='\033[2m'
RESET='\033[0m'

# Skip if tokman binary not available
if ! command -v tokman &>/dev/null; then
  echo "tokman binary not found in PATH — skipping hook tests"
  exit 0
fi

# Enable tokman for tests
mkdir -p "$HOME/.local/share/tokman"
touch "$HOME/.local/share/tokman/.enabled"

test_rewrite() {
  local description="$1"
  local input_cmd="$2"
  local expected_cmd="$3"  # empty string = expect no rewrite
  TOTAL=$((TOTAL + 1))

  local input_json
  input_json=$(jq -n --arg cmd "$input_cmd" '{"tool_name":"Bash","tool_input":{"command":$cmd}}')
  local output
  output=$(echo "$input_json" | bash "$HOOK" 2>/dev/null) || true

  if [ -z "$expected_cmd" ]; then
    if [ -z "$output" ]; then
      printf "  ${GREEN}PASS${RESET} %s ${DIM}→ (no rewrite)${RESET}\n" "$description"
      PASS=$((PASS + 1))
    else
      local actual
      actual=$(echo "$output" | jq -r '.hookSpecificOutput.updatedInput.command // empty')
      printf "  ${RED}FAIL${RESET} %s\n" "$description"
      printf "       expected: (no rewrite)\n"
      printf "       actual:   %s\n" "$actual"
      FAIL=$((FAIL + 1))
    fi
  else
    local actual
    actual=$(echo "$output" | jq -r '.hookSpecificOutput.updatedInput.command // empty' 2>/dev/null)
    if [ "$actual" = "$expected_cmd" ]; then
      printf "  ${GREEN}PASS${RESET} %s ${DIM}→ %s${RESET}\n" "$description" "$actual"
      PASS=$((PASS + 1))
    else
      printf "  ${RED}FAIL${RESET} %s\n" "$description"
      printf "       expected: %s\n" "$expected_cmd"
      printf "       actual:   %s\n" "$actual"
      FAIL=$((FAIL + 1))
    fi
  fi
}

echo "============================================"
echo "  TokMan Rewrite Hook Test Suite"
echo "============================================"
echo ""

# ---- SECTION 1: Existing patterns (regression tests) ----
echo "--- Existing patterns (regression) ---"
test_rewrite "git status" \
  "git status" \
  "tokman git status"

test_rewrite "git log --oneline -10" \
  "git log --oneline -10" \
  "tokman git log --oneline -10"

test_rewrite "git diff HEAD" \
  "git diff HEAD" \
  "tokman git diff HEAD"

test_rewrite "git show abc123" \
  "git show abc123" \
  "tokman git show abc123"

test_rewrite "git add ." \
  "git add ." \
  "tokman git add ."

test_rewrite "gh pr list" \
  "gh pr list" \
  "tokman gh pr list"

test_rewrite "npx playwright test" \
  "npx playwright test" \
  "tokman playwright test"

test_rewrite "ls -la" \
  "ls -la" \
  "tokman ls -la"

test_rewrite "curl -s https://example.com" \
  "curl -s https://example.com" \
  "tokman curl -s https://example.com"

test_rewrite "cat package.json" \
  "cat package.json" \
  "tokman read package.json"

test_rewrite "grep -rn pattern src/" \
  "grep -rn pattern src/" \
  "tokman grep -rn pattern src/"

test_rewrite "rg pattern src/" \
  "rg pattern src/" \
  "tokman grep pattern src/"

test_rewrite "cargo test" \
  "cargo test" \
  "tokman cargo test"

test_rewrite "npx prisma migrate" \
  "npx prisma migrate" \
  "tokman prisma migrate"

echo ""

# ---- SECTION 2: Env var prefix handling ----
echo "--- Env var prefix handling ---"
test_rewrite "env + playwright" \
  "TEST_SESSION_ID=2 npx playwright test --config=foo" \
  "TEST_SESSION_ID=2 tokman playwright test --config=foo"

test_rewrite "env + git status" \
  "GIT_PAGER=cat git status" \
  "GIT_PAGER=cat tokman git status"

test_rewrite "env + git log" \
  "GIT_PAGER=cat git log --oneline -10" \
  "GIT_PAGER=cat tokman git log --oneline -10"

test_rewrite "multi env + vitest" \
  "NODE_ENV=test CI=1 npx vitest run" \
  "NODE_ENV=test CI=1 tokman vitest run"

test_rewrite "env + ls" \
  "LANG=C ls -la" \
  "LANG=C tokman ls -la"

test_rewrite "env + npm run" \
  "NODE_ENV=test npm run test:e2e" \
  "NODE_ENV=test tokman npm test:e2e"

test_rewrite "env + docker compose (unsupported subcommand, NOT rewritten)" \
  "COMPOSE_PROJECT_NAME=test docker compose up -d" \
  ""

test_rewrite "env + docker compose logs (supported, rewritten)" \
  "COMPOSE_PROJECT_NAME=test docker compose logs web" \
  "COMPOSE_PROJECT_NAME=test tokman docker compose logs web"

echo ""

# ---- SECTION 3: New patterns ----
echo "--- New patterns ---"
test_rewrite "npm run test:e2e" \
  "npm run test:e2e" \
  "tokman npm test:e2e"

test_rewrite "npm run build" \
  "npm run build" \
  "tokman npm build"

test_rewrite "npm test" \
  "npm test" \
  "tokman npm test"

test_rewrite "docker compose logs postgrest" \
  "docker compose logs postgrest" \
  "tokman docker compose logs postgrest"

test_rewrite "docker compose ps" \
  "docker compose ps" \
  "tokman docker compose ps"

test_rewrite "docker compose build" \
  "docker compose build" \
  "tokman docker compose build"

test_rewrite "docker run --rm postgres" \
  "docker run --rm postgres" \
  "tokman docker run --rm postgres"

test_rewrite "docker exec -it db psql" \
  "docker exec -it db psql" \
  "tokman docker exec -it db psql"

test_rewrite "gh api repos/owner/repo" \
  "gh api repos/owner/repo" \
  "tokman gh api repos/owner/repo"

test_rewrite "gh release list" \
  "gh release list" \
  "tokman gh release list"

test_rewrite "kubectl describe pod foo" \
  "kubectl describe pod foo" \
  "tokman kubectl describe pod foo"

test_rewrite "kubectl apply -f deploy.yaml" \
  "kubectl apply -f deploy.yaml" \
  "tokman kubectl apply -f deploy.yaml"

echo ""

# ---- SECTION 3b: TOKMAN_DISABLED ----
echo "--- TOKMAN_DISABLED ---"
test_rewrite "TOKMAN_DISABLED=1 git status (no rewrite)" \
  "TOKMAN_DISABLED=1 git status" \
  ""

test_rewrite "TOKMAN_DISABLED=1 cargo test (no rewrite)" \
  "TOKMAN_DISABLED=1 cargo test" \
  ""

test_rewrite "FOO=1 TOKMAN_DISABLED=1 git status (no rewrite)" \
  "FOO=1 TOKMAN_DISABLED=1 git status" \
  ""

echo ""
echo "--- Redirect operators ---"
test_rewrite "cargo test 2>&1 | head" \
  "cargo test 2>&1 | head" \
  "tokman cargo test 2>&1 | head"

test_rewrite "cargo test 2>&1" \
  "cargo test 2>&1" \
  "tokman cargo test 2>&1"

test_rewrite "cargo test &>/dev/null" \
  "cargo test &>/dev/null" \
  "tokman cargo test &>/dev/null"

test_rewrite "cargo test & git status (hook rewrites first segment only)" \
  "cargo test & git status" \
  "tokman cargo test & git status"

echo ""

# ---- SECTION 4: Tail rewriting ----
echo "--- Tail rewriting ---"
test_rewrite "tail -10 file.txt" \
  "tail -10 file.txt" \
  "tokman read file.txt --tail-lines 10"

test_rewrite "tail -n 5 log.txt" \
  "tail -n 5 log.txt" \
  "tokman read log.txt --tail-lines 5"

test_rewrite "tail --lines=20 output.log" \
  "tail --lines=20 output.log" \
  "tokman read output.log --tail-lines 20"

echo ""

# ---- SECTION 5: Head rewriting ----
echo "--- Head rewriting ---"
test_rewrite "head -10 file.txt" \
  "head -10 file.txt" \
  "tokman read file.txt --max-lines 10"

test_rewrite "head --lines=5 log.txt" \
  "head --lines=5 log.txt" \
  "tokman read log.txt --max-lines 5"

echo ""

# ---- SECTION 6: Should NOT rewrite ----
echo "--- Should NOT rewrite ---"
test_rewrite "already tokman" \
  "tokman git status" \
  ""

test_rewrite "heredoc" \
  "cat <<'EOF'
hello
EOF" \
  ""

test_rewrite "echo (no pattern)" \
  "echo hello world" \
  ""

test_rewrite "cd (no pattern)" \
  "cd /tmp" \
  ""

test_rewrite "mkdir (no pattern)" \
  "mkdir -p foo/bar" \
  ""

test_rewrite "python3 (no pattern)" \
  "python3 script.py" \
  ""

test_rewrite "node (no pattern)" \
  "node -e 'console.log(1)'" \
  ""

echo ""

# ---- SECTION 7: Find pipe skip ----
echo "--- Find pipe skip ---"
test_rewrite "find piped to xargs (NOT rewritten)" \
  "find . -name '*.go' | xargs grep TODO" \
  ""

test_rewrite "find piped to head (NOT rewritten)" \
  "find . -type f | head -10" \
  ""

echo ""

# ---- SECTION 8: Vitest edge case ----
echo "--- Vitest run dedup ---"
test_rewrite "vitest (no args)" \
  "vitest" \
  "tokman vitest run"

test_rewrite "vitest run (no double run)" \
  "vitest run" \
  "tokman vitest run"

test_rewrite "npx vitest run" \
  "npx vitest run" \
  "tokman vitest run"

test_rewrite "pnpm vitest run --coverage" \
  "pnpm vitest run --coverage" \
  "tokman vitest run --coverage"

echo ""

# ---- SUMMARY ----
echo "============================================"
if [ $FAIL -eq 0 ]; then
  printf "  ${GREEN}ALL $TOTAL TESTS PASSED${RESET}\n"
else
  printf "  ${RED}$FAIL FAILED${RESET} / $TOTAL total ($PASS passed)\n"
fi
echo "============================================"

exit $FAIL
