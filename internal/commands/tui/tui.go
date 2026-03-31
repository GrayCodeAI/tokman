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

type model struct {
	width   int
	height  int
	tracker *tracking.Tracker
	ready   bool
	tab     int
	updated time.Time

	// Data
	sum      *Summary
	dayData  []DayData
	cmdData  []CmdStat
	teeData  []TeeFile
	discData []DiscItem

	// Viewports
	mainVP viewport.Model
}

type Summary struct {
	TotalCommands int
	TotalInput    int
	TotalOutput   int
	TotalSaved    int
	AvgSavings    float64
	Period        string
}

type DayData struct {
	Date  string
	Saved int
	Count int
}

type CmdStat struct {
	Command string
	Count   int
	Saved   int
	AvgPct  float64
}

type TeeFile struct {
	Filename string
	Command  string
	Date     string
}

type DiscItem struct {
	Command string
	Saving  int
	Suggest string
}

type tickMsg time.Time

type dataMsg struct {
	sum  *Summary
	day  []DayData
	cmd  []CmdStat
	tee  []TeeFile
	disc []DiscItem
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

func initialModel(tracker *tracking.Tracker) model {
	return model{
		tracker: tracker,
		updated: time.Now(),
		mainVP:  viewport.New(80, 20),
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
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
			m.mainVP.Width = msg.Width
			m.mainVP.Height = msg.Height - 4
		}
		return m, nil
	case tickMsg:
		m.updated = time.Now()
		cmds = append(cmds, tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) }))
		cmds = append(cmds, fetchData(m.tracker))
		return m, tea.Batch(cmds...)
	case dataMsg:
		m.sum = msg.sum
		m.dayData = msg.day
		m.cmdData = msg.cmd
		m.teeData = msg.tee
		m.discData = msg.disc
		m.render()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % 5
			m.render()
		case "left", "h":
			m.tab = (m.tab - 1 + 5) % 5
			m.render()
		case "1":
			m.tab = 0
			m.render()
		case "2":
			m.tab = 1
			m.render()
		case "3":
			m.tab = 2
			m.render()
		case "4":
			m.tab = 3
			m.render()
		case "5":
			m.tab = 4
			m.render()
		case "up", "k":
			m.mainVP.LineUp(1)
		case "down", "j":
			m.mainVP.LineDown(1)
		case "g":
			m.mainVP.GotoTop()
		case "G":
			m.mainVP.GotoBottom()
		case "r":
			cmds = append(cmds, fetchData(m.tracker))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  loading..."
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		m.tabBar(),
		m.mainVP.View(),
		m.footer(),
	)
}

func (m model) header() string {
	if m.sum == nil {
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Render(" tokman  ·  loading...")
	}

	s := m.sum
	left := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("tokman")
	mid := fmt.Sprintf("  %d commands  ·  %s saved  ·  %.0f%%",
		s.TotalCommands, fmtTok(s.TotalSaved), s.AvgSavings)
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(time.Now().Format("15:04:05"))

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			left, mid,
			lipgloss.NewStyle().Width(m.width-len(left)-len(mid)-len(right)).Render(""),
			right))
}

func (m model) tabBar() string {
	tabs := []string{"overview", "commands", "timeline", "discover", "tee"}
	var parts []string
	for i, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == m.tab {
			style = style.Bold(true).Foreground(lipgloss.Color("42"))
			parts = append(parts, style.Render("▸ "+t))
		} else {
			style = style.Foreground(lipgloss.Color("240"))
			parts = append(parts, style.Render(t))
		}
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(parts, ""))
}

func (m model) footer() string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Foreground(lipgloss.Color("240")).
		Render("  1-5: tabs  ↑↓: scroll  r: refresh  q: quit")
}

func (m *model) render() {
	var s string
	switch m.tab {
	case 0:
		s = m.viewOverview()
	case 1:
		s = m.viewCommands()
	case 2:
		s = m.viewTimeline()
	case 3:
		s = m.viewDiscover()
	case 4:
		s = m.viewTee()
	}
	m.mainVP.SetContent(s)
}

func (m model) viewOverview() string {
	if m.sum == nil {
		return "\n  loading..."
	}
	s := m.sum

	// Build efficiency meter
	w := 40
	f := int(s.AvgSavings / 100.0 * float64(w))
	if f > w {
		f = w
	}
	bar := strings.Repeat("█", f) + strings.Repeat("░", w-f)
	meterColor := "240"
	if s.AvgSavings >= 70 {
		meterColor = "42"
	} else if s.AvgSavings >= 40 {
		meterColor = "220"
	}
	meter := lipgloss.NewStyle().Foreground(lipgloss.Color(meterColor)).Render(bar) + " " + fmt.Sprintf("%.0f%%", s.AvgSavings)

	// Build sparkline for daily data
	spark := m.sparkline()

	content := strings.Join([]string{
		"",
		"  ┌─ summary ──────────────────────────────────────────────────┐",
		fmt.Sprintf("  │  commands    %-48d │", s.TotalCommands),
		fmt.Sprintf("  │  input       %-48s │", fmtTok(s.TotalInput)),
		fmt.Sprintf("  │  output      %-48s │", fmtTok(s.TotalOutput)),
		fmt.Sprintf("  │  saved       %-48s │", fmtTok(s.TotalSaved)+" ("+fmt.Sprintf("%.1f%%", s.AvgSavings)+")"),
		fmt.Sprintf("  │  period      %-48s │", s.Period),
		fmt.Sprintf("  │  updated     %-48s │", m.updated.Format("15:04:05")),
		"  └────────────────────────────────────────────────────────────┘",
		"",
		"  ┌─ efficiency ───────────────────────────────────────────────┐",
		fmt.Sprintf("  │  %s │", meter),
		"  └────────────────────────────────────────────────────────────┘",
	}, "\n")

	if spark != "" {
		content += "\n\n  ┌─ daily savings ──────────────────────────────────────────────┐\n"
		content += spark
		content += "\n  └────────────────────────────────────────────────────────────┘"
	}

	return content
}

func (m model) sparkline() string {
	if len(m.dayData) == 0 {
		return ""
	}
	maxS := 1
	for _, d := range m.dayData {
		if d.Saved > maxS {
			maxS = d.Saved
		}
	}
	var lines []string
	for _, d := range m.dayData {
		bl := int(math.Round(float64(d.Saved) / float64(maxS) * 48))
		bar := strings.Repeat("▸", bl)
		c := "240"
		if d.Saved > 100000 {
			c = "42"
		} else if d.Saved > 10000 {
			c = "220"
		}
		dt := d.Date
		if len(dt) > 10 {
			dt = dt[5:10]
		}
		lines = append(lines, fmt.Sprintf("  │  %s  %6s  %s  %s │",
			dt, fmtTok(d.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%d cmds", d.Count))))
	}
	return strings.Join(lines, "\n")
}

func (m model) viewCommands() string {
	lines := []string{
		"",
		"  ┌─ command history ──────────────────────────────────────────┐",
		"  │  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%-22s %6s %10s %6s", "command", "count", "saved", "avg%")) + " │",
		"  │  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", 52)) + " │",
	}
	for _, c := range m.cmdData {
		n := c.Command
		if len(n) > 20 {
			n = n[:18] + ".."
		}
		pctColor := "240"
		if c.AvgPct >= 70 {
			pctColor = "42"
		} else if c.AvgPct >= 40 {
			pctColor = "220"
		}
		lines = append(lines, fmt.Sprintf("  │  %-22s %6d %10s %s │",
			n, c.Count, fmtTok(c.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(pctColor)).Render(fmt.Sprintf("%5.0f%%", c.AvgPct))))
		if len(lines) > 25 {
			lines = append(lines, "  │  "+lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("...")+" │")
			break
		}
	}
	lines = append(lines, "  └────────────────────────────────────────────────────────────┘")
	return strings.Join(lines, "\n")
}

func (m model) viewTimeline() string {
	if len(m.dayData) == 0 {
		return "\n  no data"
	}
	maxS := 1
	for _, d := range m.dayData {
		if d.Saved > maxS {
			maxS = d.Saved
		}
	}
	lines := []string{
		"",
		"  ┌─ daily savings trend ──────────────────────────────────────┐",
	}
	for _, d := range m.dayData {
		bl := int(math.Round(float64(d.Saved) / float64(maxS) * 46))
		bar := strings.Repeat("▸", bl)
		c := "240"
		if d.Saved > 100000 {
			c = "42"
		} else if d.Saved > 10000 {
			c = "220"
		}
		dt := d.Date
		if len(dt) > 10 {
			dt = dt[5:10]
		}
		lines = append(lines, fmt.Sprintf("  │  %s  %6s  %s  %s │",
			dt, fmtTok(d.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%d cmds", d.Count))))
	}
	lines = append(lines, "  └────────────────────────────────────────────────────────────┘")
	return strings.Join(lines, "\n")
}

func (m model) viewDiscover() string {
	an := core.NewDiscoverAnalyzer()
	res := an.AnalyzeBatch([]string{
		"cat file.txt", "ls -la", "grep pattern .", "docker ps",
		"kubectl get pods", "curl http://api", "env", "npm test",
	})
	lines := []string{
		"",
		"  ┌─ missed savings opportunities ─────────────────────────────┐",
		"  │  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%-18s %8s  %s", "command", "saving", "suggestion")) + " │",
		"  │  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", 52)) + " │",
	}
	for _, r := range res {
		lines = append(lines, fmt.Sprintf("  │  %-18s %s  %s │",
			r.Command,
			lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(fmt.Sprintf("+%d", r.EstSavings)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(r.Suggestion)))
	}
	lines = append(lines, "  └────────────────────────────────────────────────────────────┘")
	return strings.Join(lines, "\n")
}

func (m model) viewTee() string {
	entries, _ := tee.List(tee.DefaultConfig())
	lines := []string{
		"",
		"  ┌─ full output recovery (tee) ───────────────────────────────┐",
	}
	if len(entries) == 0 {
		lines = append(lines, "  │  no saved outputs                                         │")
		lines = append(lines, "  │                                                           │")
		lines = append(lines, "  │  tee saves full command output on failure.                │")
		lines = append(lines, "  │  use 'tokman tee list' to see files.                      │")
	} else {
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("  │  %s  %s │",
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(e.Timestamp.Format("01-02 15:04")),
				lipgloss.NewStyle().Bold(true).Render(e.Command)))
		}
	}
	lines = append(lines, "  └────────────────────────────────────────────────────────────┘")
	return strings.Join(lines, "\n")
}

func fmtTok(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func fetchData(tracker *tracking.Tracker) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if tracker == nil {
			return dataMsg{}
		}
		s := &Summary{}
		sav, err := tracker.GetSavings("")
		if err == nil {
			s.TotalCommands = sav.TotalCommands
			s.TotalSaved = sav.TotalSaved
			s.TotalInput = sav.TotalOriginal
			s.TotalOutput = sav.TotalFiltered
			if sav.TotalOriginal > 0 {
				s.AvgSavings = float64(sav.TotalSaved) / float64(sav.TotalOriginal) * 100
			}
		}
		daily, _ := tracker.GetDailySavings("", 30)
		var dd []DayData
		if len(daily) > 0 {
			s.Period = daily[len(daily)-1].Date + " → " + daily[0].Date
			for _, d := range daily {
				dd = append(dd, DayData{Date: d.Date, Saved: d.Saved, Count: d.Commands})
			}
		}
		stats, _ := tracker.GetCommandStats("")
		var cs []CmdStat
		for _, c := range stats {
			cs = append(cs, CmdStat{Command: c.Command, Count: c.ExecutionCount, Saved: c.TotalSaved, AvgPct: c.ReductionPct})
		}
		te, _ := tee.List(tee.DefaultConfig())
		var tf []TeeFile
		for _, e := range te {
			tf = append(tf, TeeFile{Filename: e.Filename, Command: e.Command, Date: e.Timestamp.Format("01-02 15:04")})
		}
		an := core.NewDiscoverAnalyzer()
		res := an.AnalyzeBatch([]string{"cat file.txt", "ls -la", "grep pattern .", "docker ps", "kubectl get pods", "curl http://api", "env", "npm test"})
		var di []DiscItem
		for _, r := range res {
			di = append(di, DiscItem{Command: r.Command, Saving: r.EstSavings, Suggest: r.Suggestion})
		}
		return dataMsg{sum: s, day: dd, cmd: cs, tee: tf, disc: di}
	})
}
