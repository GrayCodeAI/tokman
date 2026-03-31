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

	sum      *Summary
	dayData  []DayData
	cmdData  []CmdStat
	teeData  []TeeFile
	discData []DiscItem

	vp viewport.Model
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
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - 2
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
	if !m.ready {
		return "\n  loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		m.vp.View(),
		m.footer(),
	)
}

func (m model) header() string {
	if m.sum == nil {
		return "tokman  loading..."
	}
	tabs := []string{"overview", "commands", "timeline", "discover", "tee"}
	var tabStrs []string
	for i, t := range tabs {
		if i == m.tab {
			tabStrs = append(tabStrs, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render(t))
		} else {
			tabStrs = append(tabStrs, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(t))
		}
	}
	return fmt.Sprintf("tokman  %s", strings.Join(tabStrs, "  "))
}

func (m model) footer() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  1-5:tabs  ↑↓:scroll  r:refresh  q:quit")
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

func panel(title string, content string, w int) string {
	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(w).
		Padding(0, 1)
	return border.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render(" "+title),
		content))
}

func (m model) viewOverview() string {
	if m.sum == nil {
		return "\n  loading..."
	}
	s := m.sum
	w := m.width / 2 - 2

	// Summary panel
	sumContent := strings.Join([]string{
		fmt.Sprintf("commands   %d", s.TotalCommands),
		fmt.Sprintf("input      %s", fmtTok(s.TotalInput)),
		fmt.Sprintf("output     %s", fmtTok(s.TotalOutput)),
		fmt.Sprintf("saved      %s", fmtTok(s.TotalSaved)),
		fmt.Sprintf("period     %s", s.Period),
	}, "\n")

	// Efficiency panel
	w2 := 40
	f := int(s.AvgSavings / 100.0 * float64(w2))
	if f > w2 {
		f = w2
	}
	bar := strings.Repeat("█", f) + strings.Repeat("░", w2-f)
	c := "240"
	if s.AvgSavings >= 70 {
		c = "42"
	} else if s.AvgSavings >= 40 {
		c = "220"
	}
	effContent := lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(bar) + "\n" + fmt.Sprintf("%.0f%%", s.AvgSavings)

	// Daily sparkline
	var sparkLines []string
	if len(m.dayData) > 0 {
		maxS := 1
		for _, d := range m.dayData {
			if d.Saved > maxS {
				maxS = d.Saved
			}
		}
		for _, d := range m.dayData {
			bl := int(math.Round(float64(d.Saved) / float64(maxS) * 30))
			bar := strings.Repeat("▸", bl)
			dc := "240"
			if d.Saved > 100000 {
				dc = "42"
			} else if d.Saved > 10000 {
				dc = "220"
			}
			dt := d.Date
			if len(dt) > 10 {
				dt = dt[5:10]
			}
			sparkLines = append(sparkLines, fmt.Sprintf("%s  %6s  %s  %d",
				dt, fmtTok(d.Saved),
				lipgloss.NewStyle().Foreground(lipgloss.Color(dc)).Render(bar),
				d.Count))
		}
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		panel("summary", sumContent, w),
		"",
		panel("efficiency", effContent, w),
	)

	right := panel("daily savings", strings.Join(sparkLines, "\n"), w)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m model) viewCommands() string {
	lines := []string{
		fmt.Sprintf("  %-24s %6s %10s %6s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("command"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("count"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("saved"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("avg%")),
		"  " + strings.Repeat("─", 52),
	}
	for _, c := range m.cmdData {
		n := c.Command
		if len(n) > 22 {
			n = n[:20] + ".."
		}
		pctC := "240"
		if c.AvgPct >= 70 {
			pctC = "42"
		} else if c.AvgPct >= 40 {
			pctC = "220"
		}
		lines = append(lines, fmt.Sprintf("  %-24s %6d %10s %s",
			n, c.Count, fmtTok(c.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(pctC)).Render(fmt.Sprintf("%5.0f%%", c.AvgPct))))
		if len(lines) > 30 {
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("..."))
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
		bl := int(math.Round(float64(d.Saved) / float64(maxS) * 44))
		bar := strings.Repeat("▸", bl)
		dc := "240"
		if d.Saved > 100000 {
			dc = "42"
		} else if d.Saved > 10000 {
			dc = "220"
		}
		dt := d.Date
		if len(dt) > 10 {
			dt = dt[5:10]
		}
		lines = append(lines, fmt.Sprintf("  %s  %6s  %s  %d",
			dt, fmtTok(d.Saved),
			lipgloss.NewStyle().Foreground(lipgloss.Color(dc)).Render(bar),
			d.Count))
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
		fmt.Sprintf("  %-20s %8s  %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("command"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("saving"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("suggestion")),
		"  " + strings.Repeat("─", 56),
	}
	for _, r := range res {
		lines = append(lines, fmt.Sprintf("  %-20s %s  %s",
			r.Command,
			lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(fmt.Sprintf("+%d", r.EstSavings)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(r.Suggestion)))
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
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(e.Timestamp.Format("01-02 15:04")),
				lipgloss.NewStyle().Bold(true).Render(e.Command)))
		}
	}
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
