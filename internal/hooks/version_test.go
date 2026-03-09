package hooks

import (
	"testing"
)

func TestParseHookVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected uint8
	}{
		{
			name:     "version_present",
			content:  "#!/usr/bin/env bash\n# tokman-hook-version: 2\n# some comment\n",
			expected: 2,
		},
		{
			name:     "version_missing",
			content:  "#!/usr/bin/env bash\n# old hook without version\n",
			expected: 0,
		},
		{
			name:     "version_future",
			content:  "#!/usr/bin/env bash\n# tokman-hook-version: 5\n",
			expected: 5,
		},
		{
			name:     "version_on_line_3",
			content:  "#!/usr/bin/env bash\n# comment\n# tokman-hook-version: 3\n",
			expected: 3,
		},
		{
			name:     "version_on_line_6_ignored",
			content:  "#!/usr/bin/env bash\n# line2\n# line3\n# line4\n# line5\n# tokman-hook-version: 99\n",
			expected: 0, // Only checks first 5 lines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHookVersion(tt.content)
			if got != tt.expected {
				t.Errorf("ParseHookVersion() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCurrentHookVersion(t *testing.T) {
	if CurrentHookVersion < 1 {
		t.Error("CurrentHookVersion should be at least 1")
	}
}

func TestWarnInterval(t *testing.T) {
	if WarnIntervalSecs < 3600 {
		t.Error("WarnIntervalSecs should be at least 1 hour")
	}
}
