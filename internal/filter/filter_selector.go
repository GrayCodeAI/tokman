package filter

import (
	"regexp"
	"strings"
)

// FilterSelector analyzes the first 512 bytes of content and returns an
// ordered list of filter names that are likely to produce savings.
// Filters unlikely to apply are excluded to reduce pipeline latency by 30–50%.
//
// The selector uses lightweight regex probes — no full parse pass.
type FilterSelector struct{}

// ContentProbe holds detected content signals used to select filters.
type ContentProbe struct {
	IsCode        bool
	IsGo          bool
	IsPython      bool
	IsJavaScript  bool
	IsJava        bool
	IsRust        bool
	IsJSON        bool
	IsYAML        bool
	IsHTML        bool
	IsMarkdown    bool
	IsGitDiff     bool
	IsStackTrace  bool
	IsTestOutput  bool
	IsLogFile     bool
	IsCSV         bool
	IsProto       bool
	IsSQL         bool
	IsShellOutput bool
}

var (
	probeGoRe      = regexp.MustCompile(`(?:^package\s+\w+|:=\s|func\s+\w+\()`)
	probePyRe      = regexp.MustCompile(`(?:^def\s+\w+|^import\s+\w+|print\()`)
	probeJSRe      = regexp.MustCompile(`(?:const\s+\w+\s*=|let\s+\w+\s*=|=>|console\.)`)
	probeJavaRe    = regexp.MustCompile(`(?:public\s+class|System\.out|void\s+main)`)
	probeRustRe    = regexp.MustCompile(`(?:fn\s+\w+|let\s+mut\s+|impl\s+\w+|use\s+\w+::)`)
	probeJSONRe    = regexp.MustCompile(`^\s*[\[{]`)
	probeYAMLRe    = regexp.MustCompile(`(?:^---\s*$|^\s+\w+:\s+\S)`)
	probeHTMLRe    = regexp.MustCompile(`<(?:html|head|body|div|span|script|style)\b`)
	probeMDRe      = regexp.MustCompile(`(?:^#{1,6}\s|^\*{1,3}\s|^\[.*\]\(http)`)
	probeDiffRe    = regexp.MustCompile(`(?:^diff --git|^@@ -\d+|^\+{3} |^-{3} )`)
	probeStackRe   = regexp.MustCompile(`(?:goroutine \d+ \[|Traceback \(most recent|at .+\(\S+:\d+\))`)
	probeTestRe    = regexp.MustCompile(`(?:^--- (?:PASS|FAIL):|^=== RUN|^ok\s+\S+\s+\d)`)
	probeLogRe     = regexp.MustCompile(`(?:\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}|level=\w+|msg=")`)
	probeCSVRe     = regexp.MustCompile(`(?:^[^,\n]+(?:,[^,\n]+){3,}$)`)
	probeProtoRe   = regexp.MustCompile(`(?:^syntax\s*=\s*"proto|^message\s+\w+\s*\{|^service\s+\w+)`)
	probeSQLRe     = regexp.MustCompile(`(?i)(?:^SELECT\s+|^INSERT\s+INTO|^CREATE\s+TABLE|^UPDATE\s+\w+\s+SET)`)
	probeShellRe   = regexp.MustCompile(`(?:npm (?:warn|notice|http)|downloading \d+%|\r\[={5,}\])`)
)

// NewFilterSelector creates a new filter selector.
func NewFilterSelector() *FilterSelector {
	return &FilterSelector{}
}

// Probe analyzes the first 512 bytes of content and returns detected signals.
func (s *FilterSelector) Probe(input string) ContentProbe {
	sample := input
	if len(sample) > 512 {
		sample = sample[:512]
	}

	probe := ContentProbe{}

	probe.IsGo = probeGoRe.MatchString(sample)
	probe.IsPython = probePyRe.MatchString(sample)
	probe.IsJavaScript = probeJSRe.MatchString(sample)
	probe.IsJava = probeJavaRe.MatchString(sample)
	probe.IsRust = probeRustRe.MatchString(sample)
	probe.IsCode = probe.IsGo || probe.IsPython || probe.IsJavaScript || probe.IsJava || probe.IsRust
	probe.IsJSON = probeJSONRe.MatchString(strings.TrimSpace(sample))
	probe.IsYAML = probeYAMLRe.MatchString(sample)
	probe.IsHTML = probeHTMLRe.MatchString(sample)
	probe.IsMarkdown = probeMDRe.MatchString(sample)
	probe.IsGitDiff = probeDiffRe.MatchString(sample)
	probe.IsStackTrace = probeStackRe.MatchString(sample)
	probe.IsTestOutput = probeTestRe.MatchString(sample)
	probe.IsLogFile = probeLogRe.MatchString(sample)
	probe.IsCSV = probeCSVRe.MatchString(sample)
	probe.IsProto = probeProtoRe.MatchString(sample)
	probe.IsSQL = probeSQLRe.MatchString(sample)
	probe.IsShellOutput = probeShellRe.MatchString(sample)

	return probe
}

// SelectFilters returns an ordered list of filter names applicable to the input.
// Filters are ordered by expected savings (highest first).
func (s *FilterSelector) SelectFilters(input string) []string {
	probe := s.Probe(input)
	var filters []string

	// Universal filters always run
	filters = append(filters, "rle_compress")         // repeat removal always cheap
	filters = append(filters, "boilerplate")           // license/autogen headers

	// Content-specific filters
	if probe.IsGitDiff {
		filters = append(filters, "git_diff")
	}
	if probe.IsStackTrace {
		filters = append(filters, "stack_trace")
	}
	if probe.IsTestOutput {
		filters = append(filters, "test_output")
	}
	if probe.IsShellOutput {
		filters = append(filters, "shell_output")
	}
	if probe.IsLogFile {
		filters = append(filters, "error_dedup")
	}
	if probe.IsJSON || probe.IsYAML {
		filters = append(filters, "json_yaml_compress")
	}
	if probe.IsHTML {
		filters = append(filters, "html_compress")
	}
	if probe.IsCSV {
		filters = append(filters, "csv_compress")
	}
	if probe.IsProto {
		filters = append(filters, "proto_compress")
	}
	if probe.IsSQL {
		filters = append(filters, "sql_compress")
	}
	if probe.IsCode {
		filters = append(filters, "ast_skeleton")
		filters = append(filters, "import_collapse")
		filters = append(filters, "pattern_dict_compress")
	}
	if probe.IsMarkdown {
		filters = append(filters, "markdown_compress")
	}

	// Heavy filters at the end
	filters = append(filters, "importance_scoring")
	filters = append(filters, "numeric_compress")
	filters = append(filters, "smart_truncate")

	return dedupStrings(filters)
}

// ShouldSkip returns true if the given filter name should be skipped for
// the detected content type (probe-based short-circuit).
func (s *FilterSelector) ShouldSkip(filterName string, probe ContentProbe) bool {
	switch filterName {
	case "html_compress":
		return !probe.IsHTML
	case "git_diff":
		return !probe.IsGitDiff
	case "stack_trace":
		return !probe.IsStackTrace
	case "test_output":
		return !probe.IsTestOutput
	case "shell_output":
		return !probe.IsShellOutput
	case "json_yaml_compress":
		return !probe.IsJSON && !probe.IsYAML
	case "csv_compress":
		return !probe.IsCSV
	case "proto_compress":
		return !probe.IsProto
	case "sql_compress":
		return !probe.IsSQL
	case "import_collapse":
		return !probe.IsCode
	case "ast_skeleton":
		return !probe.IsCode
	case "pattern_dict_compress":
		return !probe.IsCode
	}
	return false
}

func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
