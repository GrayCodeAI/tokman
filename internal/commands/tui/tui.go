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
	tab     int
	updated time.Time
	vp      viewport.Model

	sum      *Summary
	dayData  []DayData
	cmdData  []CmdStat
	teeData  []TeeFile
	discData []DiscItem
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
		vp:      viewport.New(80, 20),
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
			m.vp.Width = msg.Width - 2
			m.vp.Height = msg.Height - 6
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
			m.tab = -1
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
			m.vp.LineUp(1)
		case "down", "j":
			m.vp.LineDown(1)
		case "g":
			m.vp.GotoTop()
		case "G":
			m.vp.GotoBottom()
		case "r":
			cmds = append(cmds, fetchData(m.tracker))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.tab == -1 {
		return "\n\n  bye.\n"
	}
	if !m.ready {
		return "\n\n  loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		m.header(),
		m.tabs(),
		"",
		m.vp.View(),
		"",
		m.footer(),
	)
}

func (m model) header() string {
	left := bold.Render("tokman")
	var mid string
	if m.sum != nil {
		mid = fmt.Sprintf("  %d commands  ·  %s saved  ·  %.0f%%",
			m.sum.TotalCommands, fmtTok(m.sum.TotalSaved), m.sum.AvgSavings)
	} else {
		mid = "  loading..."
	}
	right := dim.Render(time.Now().Format("15:04"))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		left, mid,
		lipgloss.NewStyle().Width(m.width-len(left)-len(mid)-len(right)).Render(""),
		right)
}

func (m model) tabs() string {
	names := []string{"overview", "commands", "timeline", "discover", "tee"}
	var parts []string
	for i, n := range names {
		s := lipgloss.NewStyle().Padding(0, 1)
		if i == m.tab {
			s = s.Bold(true)
			parts = append(parts, s.Render(n))
		} else {
			s = s.Foreground(lipgloss.Color("240"))
			parts = append(parts, s.Render(n))
		}
	}
	return strings.Join(parts, "  ")
}

func (m model) footer() string {
	return dim.Render("  1-5: tabs  ↑↓: scroll  r: refresh  q: quit")
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
	m.vp.SetContent(s)
}

func (m model) viewOverview() string {
	if m.sum == nil {
		return "\n  loading..."
	}
	s := m.sum
	meter := buildMeter(s.AvgSavings)
	return strings.Join([]string{
		"",
		"  commands    " + bold.Render(fmt.Sprintf("%d", s.TotalCommands)),
		"  input       " + bold.Render(fmtTok(s.TotalInput)),
		"  output      " + bold.Render(fmtTok(s.TotalOutput)),
		"  saved       " + green.Render(fmtTok(s.TotalSaved)+"  ("+fmt.Sprintf("%.1f%%", s.AvgSavings)+")"),
		"  period      " + dim.Render(s.Period),
		"  updated     " + dim.Render(m.updated.Format("15:04:05")),
		"",
		"  efficiency  " + meter,
	}, "\n")
}

func (m model) viewCommands() string {
	lines := []string{
		"",
		"  " + dim.Render(fmt.Sprintf("%-22s %6s %10s %6s", "command", "count", "saved", "avg%")),
		"  " + dim.Render(strings.Repeat("─", 48)),
	}
	for _, c := range m.cmdData {
		n := c.Command
		if len(n) > 20 {
			n = n[:18] + ".."
		}
		lines = append(lines, fmt.Sprintf("  %-22s %6d %10s %5.0f%%", n, c.Count, fmtTok(c.Saved), c.AvgPct))
		if len(lines) > 30 {
			lines = append(lines, "  "+dim.Render("..."))
			break
		}
	}
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
	var lines []string
	for _, d := range m.dayData {
		bl := int(math.Round(float64(d.Saved) / float64(maxS) * 34))
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
		lines = append(lines, fmt.Sprintf("  %s  %6s  %s  %s",
			dt, fmtTok(d.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar),
			dim.Render(fmt.Sprintf("%d", d.Count))))
	}
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
		"  " + dim.Render(fmt.Sprintf("%-18s %8s  %s", "command", "saving", "suggestion")),
		"  " + dim.Render(strings.Repeat("─", 56)),
	}
	for _, r := range res {
		lines = append(lines, fmt.Sprintf("  %-18s %6s  %s",
			r.Command, green.Render(fmt.Sprintf("+%d", r.EstSavings)), dim.Render(r.Suggestion)))
	}
	return strings.Join(lines, "\n")
}

func (m model) viewTee() string {
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
	return strings.Join(lines, "\n")
}

func buildMeter(pct float64) string {
	w := 34
	f := int(pct / 100.0 * float64(w))
	if f > w {
		f = w
	}
	bar := strings.Repeat("█", f) + strings.Repeat("░", w-f)
	c := "240"
	if pct >= 70 {
		c = "42"
	} else if pct >= 40 {
		c = "220"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar) + " " + fmt.Sprintf("%.0f%%", pct)
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
