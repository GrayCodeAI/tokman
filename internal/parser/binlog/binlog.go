package binlog

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// BinlogIssue represents a build error or warning
type BinlogIssue struct {
	Code    string
	File    string
	Line    uint32
	Column  uint32
	Message string
}

// BuildSummary represents the result of parsing a build binlog
type BuildSummary struct {
	Succeeded    bool
	ProjectCount int
	Errors       []BinlogIssue
	Warnings     []BinlogIssue
	Duration     time.Duration
	DurationText string
}

// TestSummary represents the result of parsing a test binlog
type TestSummary struct {
	Passed       int
	Failed       int
	Skipped      int
	Total        int
	ProjectCount int
	FailedTests  []FailedTest
	Duration     time.Duration
	DurationText string
}

// FailedTest represents a failed test case
type FailedTest struct {
	Name    string
	Details []string
}

// RestoreSummary represents the result of parsing a restore binlog
type RestoreSummary struct {
	RestoredProjects int
	Warnings         int
	Errors           int
	DurationText     string
}

// Record types from MSBuild binlog format
const (
	RecordBuildStarted         = 1
	RecordBuildFinished        = 2
	RecordProjectStarted       = 3
	RecordProjectFinished      = 4
	RecordError                = 9
	RecordWarning              = 10
	RecordMessage              = 11
	RecordCriticalBuildMessage = 13
	RecordNameValueList        = 23
	RecordString               = 24
)

// Flags for event fields
const (
	FlagBuildEventContext = 1 << 0
	FlagMessage           = 1 << 2
	FlagTimestamp         = 1 << 5
	FlagArguments         = 1 << 14
	FlagImportance        = 1 << 15
	FlagExtended          = 1 << 16
)

// Regex patterns for text parsing
var (
	issueRE          = regexp.MustCompile(`(?m)^\s*([^\r\n:(]+)\((\d+),(\d+)\):\s*(error|warning)\s*(?:( [A-Za-z]+\d+)\s*:\s*)?(.*)$`)
	buildSummaryRE   = regexp.MustCompile(`(?mi)^\s*(\d+)\s+(warning|error)\(s\)`)
	testResultRE     = regexp.MustCompile(`(?m)(?:Passed!|Failed!)\s*-\s*Failed:\s*(\d+),\s*Passed:\s*(\d+),\s*Skipped:\s*(\d+),\s*Total:\s*(\d+),\s*Duration:\s*([^\r\n-]+)`)
	testSummaryRE    = regexp.MustCompile(`(?mi)^\s*Test summary:\s*total:\s*(\d+),\s*failed:\s*(\d+),\s*(?:succeeded|passed):\s*(\d+),\s*skipped:\s*(\d+),\s*duration:\s*([^\r\n]+)$`)
	failedTestHeadRE = regexp.MustCompile(`(?m)^\s*Failed\s+([^\r\n\[]+)\s+\[[^\]\r\n]+\]\s*$`)
	durationRE       = regexp.MustCompile(`(?m)^\s*Time Elapsed\s+([^\r\n]+)$`)
	restoreProjectRE = regexp.MustCompile(`(?m)^\s*Restored\s+.+\.csproj\s*\(`)
)

// ParseBuild parses an MSBuild binary log file and extracts build summary
func ParseBuild(binlogPath string) (*BuildSummary, error) {
	file, err := os.Open(binlogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open binlog: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	summary := &BuildSummary{}
	reader := newBinReader(gzReader)

	// Parse binlog format
	parsed, err := parseBinlogEvents(reader)
	if err != nil {
		// Fallback to text parsing from string records
		return parseBuildFromText(parsed.stringRecords), nil
	}

	summary.Succeeded = parsed.buildSucceeded
	summary.ProjectCount = len(parsed.projectFiles)
	summary.Errors = parsed.errors
	summary.Warnings = parsed.warnings

	if parsed.buildStartedTicks > 0 && parsed.buildFinishedTicks > parsed.buildStartedTicks {
		ticks := parsed.buildFinishedTicks - parsed.buildStartedTicks
		summary.Duration = ticksToDuration(ticks)
		summary.DurationText = formatDuration(summary.Duration)
	}

	// Merge with text fallback for better issue quality
	textSummary := parseBuildFromText(parsed.stringRecords)
	if len(summary.Errors) == 0 && len(textSummary.Errors) > 0 {
		summary.Errors = textSummary.Errors
	}
	if len(summary.Warnings) == 0 && len(textSummary.Warnings) > 0 {
		summary.Warnings = textSummary.Warnings
	}
	if summary.ProjectCount == 0 {
		summary.ProjectCount = textSummary.ProjectCount
	}

	return summary, nil
}

// ParseTest parses an MSBuild binary log file and extracts test summary
func ParseTest(binlogPath string) (*TestSummary, error) {
	file, err := os.Open(binlogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open binlog: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	reader := newBinReader(gzReader)

	parsed, err := parseBinlogEvents(reader)
	if err != nil {
		return parseTestFromText(parsed.stringRecords), nil
	}

	summary := parseTestFromText(parsed.stringRecords)
	summary.ProjectCount = len(parsed.projectFiles)

	return summary, nil
}

// ParseRestore parses an MSBuild binary log file and extracts restore summary
func ParseRestore(binlogPath string) (*RestoreSummary, error) {
	file, err := os.Open(binlogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open binlog: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	reader := newBinReader(gzReader)

	parsed, err := parseBinlogEvents(reader)
	if err != nil {
		return parseRestoreFromText(parsed.stringRecords), nil
	}

	summary := parseRestoreFromText(parsed.stringRecords)
	return summary, nil
}

// ParsedBinlog holds the parsed binlog data
type ParsedBinlog struct {
	buildStartedTicks  int64
	buildFinishedTicks int64
	buildSucceeded     bool
	projectFiles       map[string]bool
	errors             []BinlogIssue
	warnings           []BinlogIssue
	messages           []string
	stringRecords      string
}

func newParsedBinlog() *ParsedBinlog {
	return &ParsedBinlog{
		projectFiles: make(map[string]bool),
	}
}

// binReader wraps an io.Reader for binary reading
type binReader struct {
	reader io.Reader
	buf    []byte
}

func newBinReader(r io.Reader) *binReader {
	return &binReader{
		reader: r,
		buf:    make([]byte, 8),
	}
}

func (r *binReader) readByte() (byte, error) {
	_, err := io.ReadFull(r.reader, r.buf[:1])
	if err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

func (r *binReader) readInt32() (int32, error) {
	_, err := io.ReadFull(r.reader, r.buf[:4])
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(r.buf[:4])), nil
}

func (r *binReader) readInt64() (int64, error) {
	_, err := io.ReadFull(r.reader, r.buf[:8])
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(r.buf[:8])), nil
}

func (r *binReader) read7BitInt32() (int32, error) {
	var result int32
	var shift uint
	for {
		b, err := r.readByte()
		if err != nil {
			return 0, err
		}
		result |= int32(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

func (r *binReader) readBool() (bool, error) {
	b, err := r.readByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (r *binReader) readBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r.reader, buf)
	return buf, err
}

func parseBinlogEvents(reader *binReader) (*ParsedBinlog, error) {
	parsed := newParsedBinlog()

	// Read file format version
	version, err := reader.readInt32()
	if err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	if version < 1 || version > 20 {
		return nil, fmt.Errorf("unsupported binlog version: %d", version)
	}

	var stringRecords []string

	for {
		recordType, err := reader.readInt32()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record type: %w", err)
		}

		length, err := reader.read7BitInt32()
		if err != nil {
			return nil, fmt.Errorf("failed to read record length: %w", err)
		}
		if length < 0 {
			return nil, fmt.Errorf("negative record length: %d", length)
		}

		payload, err := reader.readBytes(int(length))
		if err != nil {
			return nil, fmt.Errorf("failed to read record payload: %w", err)
		}

		eventReader := newBinReader(strings.NewReader(string(payload)))

		switch recordType {
		case RecordString:
			if str, err := readStringRecord(eventReader, int(version)); err == nil && str != "" {
				stringRecords = append(stringRecords, str)
			}

		case RecordBuildStarted:
			fields, err := readEventFields(eventReader, int(version), parsed)
			if err == nil {
				parsed.buildStartedTicks = fields.timestampTicks
			}

		case RecordBuildFinished:
			fields, err := readEventFields(eventReader, int(version), parsed)
			if err == nil {
				parsed.buildFinishedTicks = fields.timestampTicks
			}
			if success, err := eventReader.readBool(); err == nil {
				parsed.buildSucceeded = success
			}

		case RecordProjectStarted:
			if _, err := readEventFields(eventReader, int(version), parsed); err != nil {
				utils.Warn("binlog: failed to read project started event fields", "error", err)
				continue
			}
			hasContext, err := eventReader.readBool()
			if err != nil {
				utils.Warn("binlog: failed to read build context flag", "error", err)
				continue
			}
			if hasContext {
				skipBuildContext(eventReader, int(version))
			}
			if projectFile, err := readOptionalString(eventReader, parsed); err == nil && projectFile != "" {
				parsed.projectFiles[projectFile] = true
			}

		case RecordProjectFinished:
			if _, err := readEventFields(eventReader, int(version), parsed); err != nil {
				utils.Warn("binlog: failed to read project finished event fields", "error", err)
				continue
			}
			if projectFile, err := readOptionalString(eventReader, parsed); err == nil && projectFile != "" {
				parsed.projectFiles[projectFile] = true
			}

		case RecordError, RecordWarning:
			fields, err := readEventFields(eventReader, int(version), parsed)
			if err != nil {
				continue
			}
			issue, err := readIssue(eventReader, parsed, fields.message)
			if err != nil {
				continue
			}
			if recordType == RecordError {
				parsed.errors = append(parsed.errors, issue)
			} else {
				parsed.warnings = append(parsed.warnings, issue)
			}

		case RecordMessage, RecordCriticalBuildMessage:
			fields, err := readEventFields(eventReader, int(version), parsed)
			if err == nil && fields.message != "" {
				parsed.messages = append(parsed.messages, fields.message)
			}
		}
	}

	parsed.stringRecords = strings.Join(stringRecords, "\n")
	return parsed, nil
}

// ParsedEventFields holds parsed event field data
type ParsedEventFields struct {
	message        string
	timestampTicks int64
}

func readEventFields(reader *binReader, version int, parsed *ParsedBinlog) (ParsedEventFields, error) {
	flags, err := reader.read7BitInt32()
	if err != nil {
		return ParsedEventFields{}, err
	}

	result := ParsedEventFields{}

	if flags&FlagMessage != 0 {
		if msg, err := readDeduplicatedString(reader, parsed); err == nil {
			result.message = msg
		}
	}

	if flags&FlagBuildEventContext != 0 {
		skipBuildContext(reader, version)
	}

	if flags&FlagTimestamp != 0 {
		if ticks, err := reader.readInt64(); err == nil {
			result.timestampTicks = ticks
		}
		if _, err := reader.read7BitInt32(); err != nil { // timestamp kind
			utils.Warn("binlog: failed to read timestamp kind", "error", err)
		}
	}

	return result, nil
}

func skipBuildContext(reader *binReader, version int) {
	count := 6
	if version > 1 {
		count = 7
	}
	for i := 0; i < count; i++ {
		if _, err := reader.read7BitInt32(); err != nil {
			utils.Warn("binlog: failed to skip build context field", "error", err, "field", i)
			return
		}
	}
}

func readStringRecord(reader *binReader, version int) (string, error) {
	strLen, err := reader.read7BitInt32()
	if err != nil {
		return "", err
	}
	if strLen < 0 {
		return "", nil
	}
	buf := make([]byte, strLen)
	_, err = io.ReadFull(reader.reader, buf)
	return string(buf), err
}

func readOptionalString(reader *binReader, parsed *ParsedBinlog) (string, error) {
	strLen, err := reader.read7BitInt32()
	if err != nil {
		return "", err
	}
	if strLen < 0 {
		return "", nil
	}

	// Check if it's a deduplicated string reference
	if int(strLen) < len(parsed.stringRecords) && strLen < 100 {
		// Likely a reference to the string table
		// For now, read the actual string
	}

	buf := make([]byte, strLen)
	_, err = io.ReadFull(reader.reader, buf)
	return string(buf), err
}

func readDeduplicatedString(reader *binReader, parsed *ParsedBinlog) (string, error) {
	return readOptionalString(reader, parsed)
}

func readIssue(reader *binReader, parsed *ParsedBinlog, message string) (BinlogIssue, error) {
	if _, err := readOptionalString(reader, parsed); err != nil { // subcategory
		utils.Warn("binlog: failed to read issue subcategory", "error", err)
	}
	code, err := readOptionalString(reader, parsed)
	if err != nil {
		utils.Warn("binlog: failed to read issue code", "error", err)
	}
	file, err := readOptionalString(reader, parsed)
	if err != nil {
		utils.Warn("binlog: failed to read issue file", "error", err)
	}
	if _, err := readOptionalString(reader, parsed); err != nil { // project file
		utils.Warn("binlog: failed to read issue project file", "error", err)
	}
	line, err := reader.read7BitInt32()
	if err != nil {
		utils.Warn("binlog: failed to read issue line", "error", err)
	}
	column, err := reader.read7BitInt32()
	if err != nil {
		utils.Warn("binlog: failed to read issue column", "error", err)
	}
	if _, err := reader.read7BitInt32(); err != nil { // end line
		utils.Warn("binlog: failed to read issue end line", "error", err)
	}
	if _, err := reader.read7BitInt32(); err != nil { // end column
		utils.Warn("binlog: failed to read issue end column", "error", err)
	}

	if line < 0 {
		line = 0
	}
	if column < 0 {
		column = 0
	}

	return BinlogIssue{
		Code:    code,
		File:    file,
		Line:    uint32(line),
		Column:  uint32(column),
		Message: message,
	}, nil
}

// Text parsing fallback functions

func parseBuildFromText(text string) *BuildSummary {
	summary := &BuildSummary{
		ProjectCount: 1,
	}

	// Parse errors and warnings
	matches := issueRE.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		issue := BinlogIssue{
			File:    match[1],
			Line:    parseUint32(match[2]),
			Column:  parseUint32(match[3]),
			Code:    strings.TrimSpace(match[5]),
			Message: match[6],
		}
		if match[4] == "error" {
			summary.Errors = append(summary.Errors, issue)
		} else {
			summary.Warnings = append(summary.Warnings, issue)
		}
	}

	// Parse duration
	if match := durationRE.FindStringSubmatch(text); match != nil {
		summary.DurationText = strings.TrimSpace(match[1])
	}

	// Determine success
	summary.Succeeded = len(summary.Errors) == 0

	return summary
}

func parseTestFromText(text string) *TestSummary {
	summary := &TestSummary{}

	// Try test result format
	if match := testResultRE.FindStringSubmatch(text); match != nil {
		summary.Failed = parseInt(match[1])
		summary.Passed = parseInt(match[2])
		summary.Skipped = parseInt(match[3])
		summary.Total = parseInt(match[4])
		summary.DurationText = strings.TrimSpace(match[5])
		return summary
	}

	// Try test summary format
	if match := testSummaryRE.FindStringSubmatch(text); match != nil {
		summary.Total = parseInt(match[1])
		summary.Failed = parseInt(match[2])
		summary.Passed = parseInt(match[3])
		summary.Skipped = parseInt(match[4])
		summary.DurationText = strings.TrimSpace(match[5])
		return summary
	}

	// Parse failed tests
	matches := failedTestHeadRE.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		summary.FailedTests = append(summary.FailedTests, FailedTest{
			Name: strings.TrimSpace(match[1]),
		})
	}

	return summary
}

func parseRestoreFromText(text string) *RestoreSummary {
	summary := &RestoreSummary{}

	// Count restored projects
	summary.RestoredProjects = len(restoreProjectRE.FindAllString(text, -1))

	// Parse duration
	if match := durationRE.FindStringSubmatch(text); match != nil {
		summary.DurationText = strings.TrimSpace(match[1])
	}

	return summary
}

// Helper functions

func ticksToDuration(ticks int64) time.Duration {
	// MSBuild ticks are 100-nanosecond intervals
	return time.Duration(ticks * 100)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	if d >= time.Hour {
		h := d / time.Hour
		d -= h * time.Hour
		m := d / time.Minute
		d -= m * time.Minute
		s := d / time.Second
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	if d >= time.Minute {
		m := d / time.Minute
		d -= m * time.Minute
		s := d / time.Second
		return fmt.Sprintf("%d:%02d", m, s)
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}

func parseInt(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
}

func parseUint32(s string) uint32 {
	var n uint32
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
}

// FormatBuildSummary creates a compact string representation of a build summary
func FormatBuildSummary(summary *BuildSummary, command string) string {
	status := "✓"
	if !summary.Succeeded {
		status = "✗"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%s %s", status, command))

	if summary.ProjectCount > 0 {
		parts = append(parts, fmt.Sprintf("%d projects", summary.ProjectCount))
	}

	parts = append(parts, fmt.Sprintf("%d errors", len(summary.Errors)))
	parts = append(parts, fmt.Sprintf("%d warnings", len(summary.Warnings)))

	if summary.DurationText != "" {
		parts = append(parts, fmt.Sprintf("(%s)", summary.DurationText))
	}

	return strings.Join(parts, ", ")
}

// FormatTestSummary creates a compact string representation of a test summary
func FormatTestSummary(summary *TestSummary, command string) string {
	status := "✓"
	if summary.Failed > 0 {
		status = "✗"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%s %s:", status, command))
	parts = append(parts, fmt.Sprintf("%d passed", summary.Passed))
	parts = append(parts, fmt.Sprintf("%d failed", summary.Failed))
	parts = append(parts, fmt.Sprintf("%d skipped", summary.Skipped))

	if summary.DurationText != "" {
		parts = append(parts, fmt.Sprintf("(%s)", summary.DurationText))
	}

	return strings.Join(parts, ", ")
}

// IsBinlogFile checks if a file is an MSBuild binary log
func IsBinlogFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".binlog"
}
