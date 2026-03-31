package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/tee"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal dashboard",
	Long:  `Launch an interactive terminal dashboard for TokMan.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	registry.Add(func() { registry.Register(tuiCmd) })
}

const refreshInterval = 2 * time.Second

var (
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dim   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bold  = lipgloss.NewStyle().Bold(true)
)

type model struct {
	width   int
	height  int
	tracker *tracking.Tracker
	ready   bool

	summary  *Summary
	daily    []DayData
	cmds     []CmdStat
	tees     []TeeFile
	discover []DiscoverItem

	sv viewport.Model
	mv viewport.Model

	focus      string
	selected   int
	quitting   bool
	lastUpdate time.Time
}

type Summary struct {
	TotalCommands int
	TotalInput    int
	TotalOutput   int
	TotalSaved    int
	AvgSavings    float64
	Period        string
}

type DayData struct{ Date string; Saved int; Count int }
type CmdStat struct{ Command string; Count int; Saved int; AvgPct float64 }
type TeeFile struct{ Filename string; Command string; Date string }
type DiscoverItem struct{ Command string; Saving int; Suggest string }

type sideItem struct{ icon, title string }

type tickMsg time.Time
type dataMsg struct {
	summary  *Summary
	daily    []DayData
	cmds     []CmdStat
	tees     []TeeFile
	discover []DiscoverItem
}

func runTUI() error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		tracker = nil
	}
	p := tea.NewProgram(initialModel(tracker), tea.WithAltScreen())
	_, err = p.Run()
	return err
}

var sideItems = []sideItem{
	{"◆", "overview"},
	{"▸", "commands"},
	{"▸", "timeline"},
	{"▸", "discover"},
	{"▸", "tee files"},
	{"▸", "layers"},
}

func initialModel(tracker *tracking.Tracker) model {
	return model{
		tracker: tracker, focus: "side", lastUpdate: time.Now(),
		sv: viewport.New(28, 20), mv: viewport.New(80, 20),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) }),
		fetchData(m.tracker),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.ready {
			m.ready = true
			m.sv.Width, m.sv.Height = 28, msg.Height-5
			m.mv.Width, m.mv.Height = msg.Width-29, msg.Height-5
		}
		return m, nil
	case tickMsg:
		m.lastUpdate = time.Now()
		cmds = append(cmds, tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) }))
		cmds = append(cmds, fetchData(m.tracker))
		return m, tea.Batch(cmds...)
	case dataMsg:
		m.summary, m.daily, m.cmds, m.tees, m.discover = msg.summary, msg.daily, msg.cmds, msg.tees, msg.discover
		m.refreshMain()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			if m.focus == "side" { m.focus = "main" } else { m.focus = "side" }
		case "up", "k":
			if m.focus == "side" {
				if m.selected > 0 { m.selected--; m.refreshMain() }
			} else { m.mv.LineUp(1) }
		case "down", "j":
			if m.focus == "side" {
				if m.selected < len(sideItems)-1 { m.selected++; m.refreshMain() }
			} else { m.mv.LineDown(1) }
		case "g": m.mv.GotoTop()
		case "G": m.mv.GotoBottom()
		case "r": cmds = append(cmds, fetchData(m.tracker))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting { return "\n\n  bye.\n" }
	if !m.ready { return "\n\n  loading..." }

	side := m.sideView()
	main := m.mv.View()
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		m.topBar(),
		lipgloss.JoinHorizontal(lipgloss.Top, side, sep, main),
		m.bottomBar(),
	)
}

func (m model) topBar() string {
	left := bold.Render(" tokman ")
	var center string
	if m.summary != nil {
		center = fmt.Sprintf(" %d cmds · %s saved · %.0f%%",
			m.summary.TotalCommands, fmtTokens(m.summary.TotalSaved), m.summary.AvgSavings)
	} else {
		center = " loading..."
	}
	right := fmt.Sprintf(" %s ", time.Now().Format("15:04:05"))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		left, center,
		lipgloss.NewStyle().Width(m.width-len(left)-len(center)-len(right)).Render(""),
		dim.Render(right))
}

func (m model) sideView() string {
	var lines []string
	for i, item := range sideItems {
		s := lipgloss.NewStyle().Padding(0, 1).Width(26)
		if i == m.selected {
			s = s.Foreground(lipgloss.Color("235")).Background(lipgloss.Color("252")).Bold(true)
		} else {
			s = s.Foreground(lipgloss.Color("240"))
		}
		lines = append(lines, s.Render(" "+item.icon+" "+item.title))
	}
	m.sv.SetContent(strings.Join(lines, "\n"))
	return m.sv.View()
}

func (m model) refreshMain() {
	if m.selected >= len(sideItems) { return }
	title := sideItems[m.selected].title
	var content string
	switch title {
	case "overview": content = m.overviewContent()
	case "commands": content = m.commandsContent()
	case "timeline": content = m.timelineContent()
	case "discover": content = m.discoverContent()
	case "tee files": content = m.teeContent()
	case "layers": content = m.layersContent()
	}
	m.mv.SetContent(content)
}

func (m model) overviewContent() string {
	if m.summary == nil { return "\n  loading..." }
	s := m.summary
	meter := buildMeter(s.AvgSavings)
	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		"  commands     "+bold.Render(fmt.Sprintf("%d", s.TotalCommands)),
		"  input        "+bold.Render(fmtTokens(s.TotalInput)),
		"  output       "+bold.Render(fmtTokens(s.TotalOutput)),
		"  saved        "+green.Render(fmtTokens(s.TotalSaved)+"  ("+fmt.Sprintf("%.1f%%", s.AvgSavings)+")"),
		"  period       "+dim.Render(s.Period),
		"  updated      "+dim.Render(m.lastUpdate.Format("15:04:05")),
		"",
		"  efficiency   "+meter,
	)
}

func (m model) commandsContent() string {
	lines := []string{
		"",
		"  " + dim.Render(fmt.Sprintf("%-22s %6s %10s %6s", "command", "count", "saved", "avg%")),
		"  " + dim.Render(strings.Repeat("─", 48)),
	}
	for _, c := range m.cmds {
		name := c.Command
		if len(name) > 20 { name = name[:18]+".." }
		lines = append(lines, fmt.Sprintf("  %-22s %6d %10s %5.0f%%", name, c.Count, fmtTokens(c.Saved), c.AvgPct))
		if len(lines) > 30 { lines = append(lines, "  "+dim.Render("...")); break }
	}
	return lipgloss.JoinVertical(lipgloss.Left, "  "+bold.Render("commands"), strings.Join(lines, "\n"))
}

func (m model) timelineContent() string {
	if len(m.daily) == 0 { return "\n  no data" }
	maxS := 1
	for _, d := range m.daily { if d.Saved > maxS { maxS = d.Saved } }
	var lines []string
	for _, d := range m.daily {
		bl := int(math.Round(float64(d.Saved) / float64(maxS) * 34))
		bar := strings.Repeat("▸", bl)
		color := "240"
		if d.Saved > 100000 { color = "42" } else if d.Saved > 10000 { color = "220" }
		dt := d.Date
		if len(dt) > 10 { dt = dt[5:10] }
		lines = append(lines, fmt.Sprintf("  %s  %6s  %s  %s",
			dt, fmtTokens(d.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(bar),
			dim.Render(fmt.Sprintf("%d", d.Count))))
	}
	return strings.Join(lines, "\n")
}

func (m model) discoverContent() string {
	analyzer := core.NewDiscoverAnalyzer()
	results := analyzer.AnalyzeBatch([]string{
		"cat file.txt", "ls -la", "grep pattern .", "docker ps",
		"kubectl get pods", "curl http://api", "env", "npm test",
	})
	lines := []string{
		"",
		"  " + dim.Render(fmt.Sprintf("%-18s %8s  %s", "command", "saving", "suggestion")),
		"  " + dim.Render(strings.Repeat("─", 56)),
	}
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("  %-18s %6s  %s",
			r.Command, green.Render(fmt.Sprintf("+%d", r.EstSavings)), dim.Render(r.Suggestion)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, "  "+bold.Render("missed savings"), strings.Join(lines, "\n"))
}

func (m model) teeContent() string {
	entries, _ := tee.List(tee.DefaultConfig())
	lines := []string{""}
	if len(entries) == 0 {
		lines = append(lines, "  no saved outputs")
		lines = append(lines, "")
		lines = append(lines, "  tee saves full command output on failure.")
		lines = append(lines, "  use 'tokman tee list' to see files.")
	} else {
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				dim.Render(e.Timestamp.Format("01-02 15:04")),
				bold.Render(e.Command)))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, "  "+bold.Render("tee recovery"), strings.Join(lines, "\n"))
}

func (m model) layersContent() string {
	layers := []struct{ n, p, s string }{
		{"1  entropy", "Selective Context", "on"},
		{"2  perplexity", "LLMLingua", "on"},
		{"3  goal_driven", "SWE-Pruner", "?"},
		{"4  ast_preserve", "LongCodeZip", "on"},
		{"5  contrastive", "LongLLMLingua", "?"},
		{"6  ngram", "CompactPrompt", "on"},
		{"7  evaluator", "EHPC", "on"},
		{"8  gist", "Gisting", "on"},
		{"9  hierarchical", "AutoCompressor", "on"},
		{"11 compaction", "MemGPT", "off"},
		{"13 h2o", "Heavy-Hitter", "on"},
		{"14 attn_sink", "StreamingLLM", "on"},
		{"15 meta_token", "Meta-Tokens", "on"},
		{"23 swezze", "SWEzze 2026", "off"},
		{"24 mixed_dim", "MixedDimKV", "off"},
		{"25 beaver", "BEAVER", "off"},
		{"26 poc", "PoC", "off"},
		{"27 token_quant", "TurboQuant", "off"},
		{"28 retention", "Token Retention", "off"},
		{"29 acon", "ACON", "off"},
	}
	var lines []string
	for _, l := range layers {
		st := dim.Render(l.s)
		if l.s == "on" { st = green.Render("on") }
		lines = append(lines, fmt.Sprintf("  %-14s %-18s %s", l.n, dim.Render(l.p), st))
	}
	return lipgloss.JoinVertical(lipgloss.Left, "  "+bold.Render("layers"), "", strings.Join(lines, "\n"))
}

func (m model) bottomBar() string {
	f := "side"
	if m.focus == "main" { f = "main" }
	return dim.Render(fmt.Sprintf("  tab: switch  ↑↓: scroll  r: refresh  q: quit  [%s]", f))
}

func buildMeter(pct float64) string {
	w := 34; f := int(pct / 100.0 * float64(w))
	if f > w { f = w }
	bar := strings.Repeat("█", f) + strings.Repeat("░", w-f)
	c := "240"
	if pct >= 70 { c = "42" } else if pct >= 40 { c = "220" }
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar) + " " + fmt.Sprintf("%.0f%%", pct)
}

func fmtTokens(n int) string {
	if n >= 1_000_000 { return fmt.Sprintf("%.1fM", float64(n)/1_000_000) }
	if n >= 1_000 { return fmt.Sprintf("%.1fK", float64(n)/1_000) }
	return fmt.Sprintf("%d", n)
}

func fetchData(tracker *tracking.Tracker) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if tracker == nil { return dataMsg{} }
		s := &Summary{}
		sav, err := tracker.GetSavings("")
		if err == nil {
			s.TotalCommands, s.TotalSaved = sav.TotalCommands, sav.TotalSaved
			s.TotalInput, s.TotalOutput = sav.TotalOriginal, sav.TotalFiltered
			if sav.TotalOriginal > 0 { s.AvgSavings = float64(sav.TotalSaved) / float64(sav.TotalOriginal) * 100 }
		}
		daily, _ := tracker.GetDailySavings("", 30)
		var dd []DayData
		if len(daily) > 0 {
			s.Period = daily[len(daily)-1].Date + " → " + daily[0].Date
			for _, d := range daily { dd = append(dd, DayData{d.Date, d.Saved, d.Commands}) }
		}
		stats, _ := tracker.GetCommandStats("")
		var cs []CmdStat
		for _, c := range stats { cs = append(cs, CmdStat{c.Command, c.ExecutionCount, c.TotalSaved, c.ReductionPct}) }
		te, _ := tee.List(tee.DefaultConfig())
		var tf []TeeFile
		for _, e := range te { tf = append(tf, TeeFile{e.Filename, e.Command, e.Timestamp.Format("01-02 15:04")}) }
		an := core.NewDiscoverAnalyzer()
		res := an.AnalyzeBatch([]string{"cat file.txt", "ls -la", "grep pattern .", "docker ps", "kubectl get pods", "curl http://api", "env", "npm test"})
		var di []DiscoverItem
		for _, r := range res { di = append(di, DiscoverItem{r.Command, r.EstSavings, r.Suggestion}) }
		return dataMsg{s, dd, cs, tf, di}
	})
}
