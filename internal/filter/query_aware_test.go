package filter

import (
	"strconv"
	"strings"
	"testing"
)

func TestQueryAwareFilter_Name(t *testing.T) {
	f := NewQueryAwareFilter()
	if f.Name() != "query_aware" {
		t.Errorf("expected name 'query_aware', got %q", f.Name())
	}
}

func TestQueryAwareFilter_NoQuery(t *testing.T) {
	f := NewQueryAwareFilter()

	input := "some output content"
	output, saved := f.Apply(input, ModeMinimal)

	// Without a query, should pass through unchanged
	if output != input {
		t.Error("should pass through unchanged without query")
	}
	if saved != 0 {
		t.Error("should not save tokens without query")
	}
}

func TestQueryAwareFilter_ClassifyQuery(t *testing.T) {
	f := NewQueryAwareFilter()

	tests := []struct {
		query    string
		expected QueryIntent
	}{
		{"debug the failing test", IntentDebug},
		{"find the error in the build", IntentDebug},
		{"fix the bug in auth", IntentDebug},
		{"review the pull request", IntentReview},
		{"check the diff", IntentReview},
		{"analyze the commit", IntentReview},
		{"deploy to production", IntentDeploy},
		{"release version 2.0", IntentDeploy},
		{"find the function definition", IntentSearch},
		{"search for the class", IntentSearch},
		{"run the test suite", IntentTest},
		{"check test coverage", IntentTest},
		{"build the project", IntentBuild},
		{"compile the code", IntentBuild},
		{"unknown query type", IntentUnknown},
	}

	for _, tt := range tests {
		intent := f.classifyQuery(tt.query)
		if intent != tt.expected {
			t.Errorf("classifyQuery(%q) = %v, expected %v", tt.query, intent, tt.expected)
		}
	}
}

func TestQueryAwareFilter_SetQuery(t *testing.T) {
	f := NewQueryAwareFilter()

	f.SetQuery("debug the error")

	if f.query != "debug the error" {
		t.Errorf("query not set correctly: got %q", f.query)
	}
	if f.intent != IntentDebug {
		t.Errorf("intent not classified: got %v", f.intent)
	}
}

func TestQueryAwareFilter_DebugIntent(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("debug the failing test")

	input := `Running tests...
test_auth.py:42: ERROR: Authentication failed
  File "test_auth.py", line 42
AssertionError: Expected 200, got 401

test_api.py:15: passed
test_user.py:20: passed
test_cache.py:30: passed
test_db.py:40: passed

Success: 4/5 tests passed
Build completed successfully`

	output, saved := f.Apply(input, ModeMinimal)

	// Should keep error content
	if !strings.Contains(output, "ERROR") {
		t.Error("should keep ERROR for debug intent")
	}
	if !strings.Contains(output, "test_auth.py:42") {
		t.Error("should keep file reference")
	}

	// Debug intent should filter out success messages
	if saved == 0 {
		t.Error("should save some tokens by filtering success messages")
	}
}

func TestQueryAwareFilter_ReviewIntent(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("review the changes in the PR")

	input := `diff --git a/auth.go b/auth.go
index 1234567..abcdefg 100644
--- a/auth.go
+++ b/auth.go
@@ -42,5 +42,6 @@ func authenticate() {
     token := getToken()
+    validateToken(token)
     return token
 }

Modified: auth.go (5 lines added, 2 removed)
Modified: user.go (3 lines added, 1 removed)

Build status: success
All tests passed`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep diff content
	if !strings.Contains(output, "diff --git") {
		t.Error("should keep diff for review intent")
	}
	if !strings.Contains(output, "@@") {
		t.Error("should keep diff hunks")
	}
	if !strings.Contains(output, "Modified:") {
		t.Error("should keep modification summary")
	}
}

func TestQueryAwareFilter_SearchIntent(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("find the function definition")

	input := `src/auth/login.go:42: func authenticate(user User) error {
src/auth/login.go:100: func validateToken(token string) bool {
src/auth/user.go:15: type User struct {
src/auth/user.go:30: func (u *User) Save() error {

Running tests...
test_001: passed
test_002: passed
...

Build: success`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep file references
	if !strings.Contains(output, "src/auth/login.go:42") {
		t.Error("should keep file:line references for search")
	}
	if !strings.Contains(output, "func authenticate") {
		t.Error("should keep function definitions")
	}
}

func TestQueryAwareFilter_TestIntent(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("run tests and check results")

	input := `Building project...
Compiling module1...
Compiling module2...

running 100 tests
test_001 ... ok
test_002 ... ok
test_003 ... FAILED
test_004 ... ok

test result: FAILED. 98 passed; 2 failed

Stack trace:
  at test_003 in test_auth.go:42
  at main in main.go:100

Finished in 0.45s`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep test results
	if !strings.Contains(output, "test result:") {
		t.Error("should keep test result summary")
	}
	if !strings.Contains(output, "FAILED") {
		t.Error("should keep failed status")
	}
	if !strings.Contains(output, "Stack trace") {
		t.Error("should keep stack trace for failed tests")
	}
}

func TestQueryAwareFilter_DeployIntent(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("deploy to production")

	input := `Building Docker image...
Step 1/10: FROM node:20
Step 2/10: COPY package.json
Step 3/10: RUN npm install
...
Step 10/10: CMD ["npm", "start"]

Successfully built image: myapp:v2.0.1
Tag: latest, v2.0.1
Deployed to: production
Version: 2.0.1
Time: 2024-03-18 10:30:00 UTC

Container logs:
Server started on port 3000
Database connected
Cache initialized`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep deployment status
	if !strings.Contains(output, "Successfully built") {
		t.Error("should keep success status for deploy intent")
	}
	if !strings.Contains(output, "Deployed to:") {
		t.Error("should keep deployment info")
	}
	if !strings.Contains(output, "Version:") {
		t.Error("should keep version info")
	}
}

func TestQueryAwareFilter_RelevanceThresholds(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("debug error")

	// High relevance segment
	highRel := f.calculateRelevance("ERROR: failed to connect", IntentDebug)

	// Low relevance segment
	lowRel := f.calculateRelevance("Success: operation completed", IntentDebug)

	if highRel <= lowRel {
		t.Errorf("error content should have higher relevance for debug intent: high=%v, low=%v", highRel, lowRel)
	}
}

func TestQueryAwareFilter_ShortInput(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("debug")

	input := "short"
	output, saved := f.Apply(input, ModeMinimal)

	// Short input should pass through
	if output != input {
		t.Error("short input should not be filtered")
	}
	if saved != 0 {
		t.Error("short input should not save tokens")
	}
}

func TestQueryAwareFilter_HasFileReference(t *testing.T) {
	f := NewQueryAwareFilter()

	tests := []struct {
		segment  string
		expected bool
	}{
		{"Error at main.go:42", true},
		{"src/lib.rs:100", true},
		{"file.py:50", true},
		{"no reference here", false},
	}

	for _, tt := range tests {
		result := f.hasFileReference(tt.segment)
		if result != tt.expected {
			t.Errorf("hasFileReference(%q) = %v, expected %v", tt.segment, result, tt.expected)
		}
	}
}

func TestQueryAwareFilter_RealWorld(t *testing.T) {
	f := NewQueryAwareFilter()
	f.SetQuery("find why the build is failing")

	// Realistic build output
	input := `Running build for myproject...

[1/50] Compiling dependency A
[2/50] Compiling dependency B
[3/50] Compiling dependency C
...
[25/50] Compiling mymodule
error[E0425]: cannot find value 'user' in this scope
  --> src/auth/login.rs:42:5
   |
42 |     let u = user.id;
   |             ^^^^ not found in this scope
   |
help: consider using 'self.user' instead

error[E0425]: cannot find value 'token' in this scope
  --> src/auth/login.rs:50:5
   |
50 |     return token;
   |            ^^^^^ help: a local variable with a similar name exists: 'auth_token'

[26/50] Compiling another module
...
[50/50] Linking

Build FAILED: 2 errors, 15 warnings

Warning: unused variable in src/cache.rs
Warning: deprecated API in src/api.rs
... and 13 more warnings`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep errors
	if !strings.Contains(output, "error[E0425]") {
		t.Error("should keep error messages")
	}
	if !strings.Contains(output, "src/auth/login.rs:42") {
		t.Error("should keep file:line references")
	}
	if !strings.Contains(output, "Build FAILED") {
		t.Error("should keep build status")
	}
}

func BenchmarkQueryAwareFilter_Apply(b *testing.B) {
	f := NewQueryAwareFilter()
	f.SetQuery("debug the error")

	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "Compiling module"+strconv.Itoa(i)+"...")
	}
	lines = append(lines, "ERROR: Failed at module25")
	lines = append(lines, "  --> src/main.rs:100:5")
	for i := 0; i < 50; i++ {
		lines = append(lines, "Processing step"+strconv.Itoa(i))
	}

	input := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeMinimal)
	}
}
