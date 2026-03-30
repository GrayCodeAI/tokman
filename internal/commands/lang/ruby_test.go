package lang

import (
	"testing"
)

// =============================================================================
// RSpec Output Tests
// =============================================================================

func TestFilterRspecOutputJSON(t *testing.T) {
	jsonOutput := `{
  "version": "3.12.0",
  "examples": [
    {
      "id": "./spec/example_spec.rb[1:1]",
      "description": "does something",
      "full_description": "Example does something",
      "status": "passed",
      "file_path": "./spec/example_spec.rb",
      "line_number": 4
    },
    {
      "id": "./spec/example_spec.rb[1:2]",
      "description": "fails",
      "full_description": "Example fails",
      "status": "failed",
      "file_path": "./spec/example_spec.rb",
      "line_number": 8,
      "exception": {
        "class": "RuntimeError",
        "message": "Expected true, got false"
      }
    }
  ],
  "summary": {
    "duration": 0.5,
    "example_count": 2,
    "failure_count": 1,
    "pending_count": 0
  }
}`

	result := filterRspecOutput(jsonOutput)

	if result == "" {
		t.Error("Expected non-empty output")
	}

	// Check for expected elements
	if !containsStr(result, "RSpec Results") {
		t.Error("Expected 'RSpec Results' in output")
	}
	if !containsStr(result, "1 passed") {
		t.Error("Expected '1 passed' in output")
	}
	if !containsStr(result, "1 failed") {
		t.Error("Expected '1 failed' in output")
	}
}

func TestFilterRspecOutputText(t *testing.T) {
	textOutput := `..F..

Failures:

  1) Example fails
     Failure/Error: expect(true).to be_falsey

Finished in 0.5 seconds (files took 0.1 seconds to load)
5 examples, 1 failure
`

	result := filterRspecTextOutput(textOutput)

	if result == "" {
		t.Error("Expected non-empty output")
	}

	if !containsStr(result, "RSpec") {
		t.Error("Expected 'RSpec' in output")
	}
}

func TestFilterRspecOutputUltraCompact(t *testing.T) {
	rspec := RSpecJSON{
		Summary: RSpecSummary{
			ExampleCount: 10,
			FailureCount: 2,
			PendingCount: 1,
		},
		Examples: []RSpecExample{
			{
				Status:     "failed",
				FilePath:   "./spec/example_spec.rb",
				LineNumber: 10,
			},
			{
				Status:     "failed",
				FilePath:   "./spec/other_spec.rb",
				LineNumber: 20,
			},
		},
	}

	result := filterRspecOutputUltraCompact(rspec)

	if !containsStr(result, "P:7") {
		t.Errorf("Expected 'P:7' in output, got: %s", result)
	}
	if !containsStr(result, "F:2") {
		t.Errorf("Expected 'F:2' in output, got: %s", result)
	}
	if !containsStr(result, "S:1") {
		t.Errorf("Expected 'S:1' in output, got: %s", result)
	}
}

// =============================================================================
// RuboCop Output Tests
// =============================================================================

func TestFilterRubocopOutputJSON(t *testing.T) {
	jsonOutput := `{
  "metadata": {
    "rubocop_version": "1.50.0"
  },
  "files": [
    {
      "path": "lib/example.rb",
      "offenses": [
        {
          "severity": "convention",
          "message": "Line is too long",
          "cop_name": "Layout/LineLength",
          "location": {"start_line": 10}
        }
      ]
    },
    {
      "path": "lib/other.rb",
      "offenses": []
    }
  ],
  "summary": {
    "offense_count": 1,
    "target_file_count": 2,
    "inspected_file_count": 2
  }
}`

	result := filterRubocopOutput(jsonOutput)

	if result == "" {
		t.Error("Expected non-empty output")
	}

	if !containsStr(result, "RuboCop Results") {
		t.Error("Expected 'RuboCop Results' in output")
	}
	if !containsStr(result, "1 offenses") {
		t.Error("Expected '1 offenses' in output")
	}
}

func TestFilterRubocopOutputClean(t *testing.T) {
	jsonOutput := `{
  "metadata": {"rubocop_version": "1.50.0"},
  "files": [],
  "summary": {
    "offense_count": 0,
    "target_file_count": 1,
    "inspected_file_count": 1
  }
}`

	result := filterRubocopOutput(jsonOutput)

	if !containsStr(result, "No offenses detected") {
		t.Errorf("Expected 'No offenses detected' in output, got: %s", result)
	}
}

func TestFilterRubocopOutputUltraCompact(t *testing.T) {
	rubocop := RuboCopJSON{
		Summary: RuboCopSummary{
			OffenseCount:       3,
			InspectedFileCount: 5,
		},
		Files: []RuboCopFile{
			{
				Path:     "lib/example.rb",
				Offenses: []RuboCopOffense{{}, {}, {}},
			},
		},
	}

	result := filterRubocopOutputUltraCompact(rubocop)

	if !containsStr(result, "O:3") {
		t.Errorf("Expected 'O:3' in output, got: %s", result)
	}
}

func TestFilterRubocopOutputUltraCompactClean(t *testing.T) {
	rubocop := RuboCopJSON{
		Summary: RuboCopSummary{
			OffenseCount:       0,
			InspectedFileCount: 5,
		},
	}

	result := filterRubocopOutputUltraCompact(rubocop)

	if !containsStr(result, "Clean") {
		t.Errorf("Expected 'Clean' in output, got: %s", result)
	}
}

// =============================================================================
// Rake Output Tests
// =============================================================================

func TestFilterRakeOutput(t *testing.T) {
	output := `Running rake test...
.
Finished in 0.5 seconds
1 tests, 1 assertions, 0 failures`

	result := filterRakeOutput(output, []string{})

	if result == "" {
		t.Error("Expected non-empty output")
	}
}

func TestFilterRakeTestOutput(t *testing.T) {
	output := `..
Finished in 0.5 seconds
2 tests, 2 assertions, 0 failures`

	result := filterRakeTestOutput(output)

	if !containsStr(result, "tests") {
		t.Errorf("Expected 'tests' in output, got: %s", result)
	}
}

func TestFilterRakeOutputEmpty(t *testing.T) {
	result := filterRakeOutput("", []string{})

	if !containsStr(result, "Rake completed") {
		t.Errorf("Expected 'Rake completed' in output, got: %s", result)
	}
}

// =============================================================================
// Bundle Output Tests
// =============================================================================

func TestFilterBundleInstallOutput(t *testing.T) {
	output := `Fetching gem metadata...
Installing rails 7.0.0
Installing rspec 3.12.0
Using bundler 2.4.0
Bundle complete! 2 Gemfile dependencies, 2 gems now installed.`

	result := filterBundleInstallOutput(output)

	if !containsStr(result, "Bundle complete") {
		t.Errorf("Expected 'Bundle complete' in output, got: %s", result)
	}
}

func TestFilterBundleInstallOutputAlreadyUpToDate(t *testing.T) {
	output := `Using bundler 2.4.0
Bundle complete! 1 Gemfile dependency, 1 gem now installed.`

	result := filterBundleInstallOutput(output)

	if !containsStr(result, "unchanged") && !containsStr(result, "Bundle complete") {
		t.Errorf("Expected 'unchanged' or 'Bundle complete' in output, got: %s", result)
	}
}

func TestFilterBundleOutdatedOutput(t *testing.T) {
	// When up to date
	result := filterBundleOutdatedOutput("Bundle up to date!")
	if !containsStr(result, "up to date") {
		t.Errorf("Expected 'up to date' in output, got: %s", result)
	}
}

func TestFilterBundleUpdateOutput(t *testing.T) {
	output := `Fetching gem metadata...
Installing rails 7.0.0 (was 6.1.0)
Bundle updated!`

	result := filterBundleUpdateOutput(output)

	if !containsStr(result, "Updated") && !containsStr(result, "Bundle updated") {
		t.Errorf("Expected 'Updated' or 'Bundle updated' in output, got: %s", result)
	}
}

// =============================================================================
// Rails Output Tests
// =============================================================================

func TestFilterRailsTestOutput(t *testing.T) {
	output := `..
2 runs, 2 assertions, 0 failures, 0 errors`

	result := filterRailsTestOutput(output)

	if !containsStr(result, "runs") {
		t.Errorf("Expected 'runs' in output, got: %s", result)
	}
}

func TestFilterRailsTestOutputFailures(t *testing.T) {
	output := `.F
FAIL["test_example", 10]
2 runs, 1 assertions, 1 failures, 0 errors`

	result := filterRailsTestOutput(output)

	if !containsStr(result, "FAIL") && !containsStr(result, "failures") {
		t.Errorf("Expected failure info in output, got: %s", result)
	}
}

func TestFilterRailsDbMigrateOutput(t *testing.T) {
	output := `== 20240101000001 CreateUsers: migrating =====================================
-- create_table(:users)
   -> 0.0020s
== 20240101000001 CreateUsers: migrated 0.0023s ================================`

	result := filterRailsDbMigrateOutput(output)

	if !containsStr(result, "migrated") {
		t.Errorf("Expected 'migrated' in output, got: %s", result)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
