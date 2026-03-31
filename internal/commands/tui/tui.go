package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI dashboard",
	Long:  `Launch an interactive terminal dashboard for TokMan with real-time analytics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	registry.Add(func() { registry.Register(tuiCmd) })
}

const refreshInterval = 2 * time.Second

type tab int

const (
	tabOverview tab = iota
	tabCommands
	tabLayers
	tabTimeline
)

type model struct {
	width     int
	height    int
	activeTab tab
	tracker   *tracking.Tracker
	ready     bool

	summary   *Summary
	cmdTable  table.Model
	layerTable table.Model
	viewport  viewport.Model

	quitting bool
	showHelp bool
	lastTick time.Time
}

type Summary struct {
	TotalCommands int
	TotalInput    int
	TotalOutput   int
	TotalSaved    int
	AvgSavings    float64
	Period        string
	LastUpdated   time.Time
}

type DayData struct {
	Date  string
	Saved int
	Count int
}

type tickMsg time.Time
type dataUpdatedMsg struct {
	summary   *Summary
	cmdRows   []table.Row
	layerRows []table.Row
	dailyData []DayData
}

func runTUI() error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		tracker = nil
	}
	p := tea.NewProgram(initialModel(tracker), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

func initialModel(tracker *tracking.Tracker) model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Command", Width: 26},
			{Title: "Count", Width: 7},
			{Title: "Saved", Width: 12},
			{Title: "Avg%", Width: 8},
			{Title: "Last Seen", Width: 14},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	l := table.New(
		table.WithColumns([]table.Column{
			{Title: "Layer", Width: 22},
			{Title: "Paper", Width: 24},
			{Title: "Status", Width: 12},
		}),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)
	l.SetStyles(ts)

	v := viewport.New(80, 20)

	return model{
		tracker:    tracker,
		cmdTable:   t,
		layerTable: l,
		viewport:   v,
		lastTick:   time.Now(),
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
			m.cmdTable.SetWidth(msg.Width - 4)
			m.layerTable.SetWidth(msg.Width - 4)
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 12
		}
		return m, nil

	case tickMsg:
		m.lastTick = time.Now()
		cmds = append(cmds, tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) }))
		cmds = append(cmds, fetchData(m.tracker))
		return m, tea.Batch(cmds...)

	case dataUpdatedMsg:
		m.summary = msg.summary
		m.cmdTable.SetRows(msg.cmdRows)
		m.layerTable.SetRows(msg.layerRows)
		m.viewport.SetContent(m.renderTimeline(msg.dailyData))
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "tab", "right", "l":
			m.activeTab = (m.activeTab + 1) % 4
		case "left", "h":
			m.activeTab = (m.activeTab - 1 + 4) % 4
		case "1":
			m.activeTab = tabOverview
		case "2":
			m.activeTab = tabCommands
		case "3":
			m.activeTab = tabLayers
		case "4":
			m.activeTab = tabTimeline
		case "r":
			cmds = append(cmds, fetchData(m.tracker))
		}
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case tabCommands:
		m.cmdTable, cmd = m.cmdTable.Update(msg)
	case tabLayers:
		m.layerTable, cmd = m.layerTable.Update(msg)
	case tabTimeline:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("TokMan TUI closed. Goodbye!\n")
	}
	if m.showHelp {
		return m.helpView()
	}
	if !m.ready {
		return "\n  Initializing..."
	}

	var content string
	switch m.activeTab {
	case tabOverview:
		content = m.overviewView()
	case tabCommands:
		content = m.commandsView()
	case tabLayers:
		content = m.layersView()
	case tabTimeline:
		content = m.viewport.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.statusBar(),
		m.tabBar(),
		"",
		content,
		"",
		m.helpBar(),
	)
}

func (m model) statusBar() string {
	left := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		Padding(0, 1).
		Render(" TokMan ")

	var center string
	if m.summary != nil {
		center = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Render(fmt.Sprintf(" %d commands · %s saved · %.1f%% avg ",
				m.summary.TotalCommands,
				formatTokens(m.summary.TotalSaved),
				m.summary.AvgSavings))
	} else {
		center = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" Loading... ")
	}

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(fmt.Sprintf(" %s ", time.Now().Format("15:04:05")))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		left,
		center,
		lipgloss.NewStyle().Width(m.width-len(left)-len(center)-len(right)).Render(""),
		right,
	)
}

func (m model) tabBar() string {
	tabs := []string{"Overview", "Commands", "Layers", "Timeline"}
	var rendered []string
	for i, t := range tabs {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 2)
		if tab(i) == m.activeTab {
			style = style.
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(true)
			t = "▸ " + t
		}
		rendered = append(rendered, style.Render(t))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m model) helpBar() string {
	keys := []string{"1-4 tabs", "←→ nav", "↑↓ scroll", "r refresh", "? help", "q quit"}
	var styled []string
	for _, k := range keys {
		styled = append(styled, lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(k))
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderTop(true).
		Padding(0, 1).
		Render(strings.Join(styled, "  "))
}

func (m model) helpView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		Padding(0, 2).
		Render(" Keyboard Shortcuts ")

	help := strings.Join([]string{
		"",
		"  Navigation:",
		"    1-4 / Tab / ←→    Switch tabs",
		"    ↑↓ / j/k          Scroll list",
		"    g / G             Go to top / bottom",
		"",
		"  Actions:",
		"    r                 Refresh data",
		"    ?                 Toggle this help",
		"    q / Ctrl+C        Quit",
		"",
		"  Tabs:",
		"    1 Overview        Dashboard with KPIs + sparkline",
		"    2 Commands        Command history with timestamps",
		"    3 Layers          Compression layer status",
		"    4 Timeline        Daily savings trend",
		"",
	}, "\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		help,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  Press ? to close"),
	)
}

func (m model) overviewView() string {
	if m.summary == nil {
		return "  Loading data..."
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	kpis := []string{
		fmt.Sprintf("  Commands:  %s", cyan.Render(fmt.Sprintf("%d", m.summary.TotalCommands))),
		fmt.Sprintf("  Input:     %s", cyan.Render(formatTokens(m.summary.TotalInput))),
		fmt.Sprintf("  Output:    %s", cyan.Render(formatTokens(m.summary.TotalOutput))),
		fmt.Sprintf("  Saved:     %s", green.Render(fmt.Sprintf("%s (%.1f%%)", formatTokens(m.summary.TotalSaved), m.summary.AvgSavings))),
		fmt.Sprintf("  Period:    %s", dim.Render(m.summary.Period)),
		fmt.Sprintf("  Updated:   %s", dim.Render(m.summary.LastUpdated.Format("15:04:05"))),
	}

	meter := buildEfficiencyMeter(m.summary.AvgSavings)

	left := lipgloss.JoinVertical(lipgloss.Left,
		strings.Join(kpis, "\n"),
		"",
		"  Efficiency: "+meter,
	)

	return left
}

func (m model) commandsView() string {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := cyan.Render("Command History")
	divider := dim.Render(strings.Repeat("─", m.width-4))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		m.cmdTable.View(),
	)
}

func (m model) layersView() string {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := cyan.Render("Compression Layers (37 total)")
	divider := dim.Render(strings.Repeat("─", m.width-4))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		m.layerTable.View(),
	)
}

func (m model) renderTimeline(data []DayData) string {
	if len(data) == 0 {
		return "  No timeline data available."
	}

	var lines []string
	maxSaved := 1
	for _, d := range data {
		if d.Saved > maxSaved {
			maxSaved = d.Saved
		}
	}

	for _, d := range data {
		barLen := int(math.Round(float64(d.Saved) / float64(maxSaved) * 40))
		bar := strings.Repeat("█", barLen)
		var color string
		if d.Saved > 100000 {
			color = "42"
		} else if d.Saved > 10000 {
			color = "220"
		} else {
			color = "240"
		}
		coloredBar := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(bar)
		dateShort := d.Date
		if len(dateShort) > 10 {
			dateShort = dateShort[5:10]
		}
		lines = append(lines, fmt.Sprintf("  %s  %6s  %s  %s",
			dateShort,
			formatTokens(d.Saved),
			coloredBar,
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%d cmds", d.Count))))
	}

	return strings.Join(lines, "\n")
}

func fetchData(tracker *tracking.Tracker) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if tracker == nil {
			return dataUpdatedMsg{}
		}

		summary := &Summary{LastUpdated: time.Now()}
		savings, err := tracker.GetSavings("")
		if err == nil {
			summary.TotalCommands = savings.TotalCommands
			summary.TotalSaved = savings.TotalSaved
			summary.TotalInput = savings.TotalOriginal
			summary.TotalOutput = savings.TotalFiltered
			if savings.TotalOriginal > 0 {
				summary.AvgSavings = float64(savings.TotalSaved) / float64(savings.TotalOriginal) * 100
			}
		}

		daily, _ := tracker.GetDailySavings("", 30)
		if len(daily) > 0 {
			summary.Period = daily[len(daily)-1].Date + " → " + daily[0].Date
		}

		var dayData []DayData
		for _, d := range daily {
			dayData = append(dayData, DayData{Date: d.Date, Saved: d.Saved, Count: d.Commands})
		}

		stats, _ := tracker.GetCommandStats("")
		recent, _ := tracker.GetRecentCommands("", 100)
		tsMap := make(map[string]string)
		for _, r := range recent {
			if _, exists := tsMap[r.Command]; !exists {
				tsMap[r.Command] = r.Timestamp.Format("01-02 15:04")
			}
		}

		var cmdRows []table.Row
		for _, cs := range stats {
			cmdRows = append(cmdRows, table.Row{
				cs.Command,
				fmt.Sprintf("%d", cs.ExecutionCount),
				formatTokens(cs.TotalSaved),
				fmt.Sprintf("%.1f%%", cs.ReductionPct),
				tsMap[cs.Command],
			})
		}

		layerRows := getLayerRows()

		return dataUpdatedMsg{
			summary:   summary,
			cmdRows:   cmdRows,
			layerRows: layerRows,
			dailyData: dayData,
		}
	})
}

func buildEfficiencyMeter(pct float64) string {
	width := 40
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pctStr := fmt.Sprintf("%.1f%%", pct)

	if pct >= 70 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(bar) + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render(pctStr)
	} else if pct >= 40 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(bar) + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Render(pctStr)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(bar) + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(pctStr)
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func getLayerRows() []table.Row {
	return []table.Row{
		{"1_entropy", "Selective Context (Mila)", "enabled"},
		{"2_perplexity", "LLMLingua (Microsoft)", "enabled"},
		{"3_goal_driven", "SWE-Pruner (SJTU)", "query-only"},
		{"4_ast_preserve", "LongCodeZip (NUS)", "enabled"},
		{"5_contrastive", "LongLLMLingua (MS)", "query-only"},
		{"6_ngram", "CompactPrompt", "enabled"},
		{"7_evaluator", "EHPC (Tsinghua)", "enabled"},
		{"8_gist", "Gisting (Stanford)", "enabled"},
		{"9_hierarchical", "AutoCompressor (Princeton)", "enabled"},
		{"11_compaction", "MemGPT (Berkeley)", "optional"},
		{"13_h2o", "Heavy-Hitter Oracle", "enabled"},
		{"14_attention_sink", "StreamingLLM (MIT)", "enabled"},
		{"15_meta_token", "Meta-Tokens (arXiv)", "enabled"},
		{"23_swezze", "SWEzze (PKU/UCL 2026)", "optional"},
		{"24_mixed_dim", "MixedDimKV (2026)", "optional"},
		{"25_beaver", "BEAVER (2026)", "optional"},
		{"26_poc", "PoC (2026)", "optional"},
		{"27_token_quant", "TurboQuant (Google)", "optional"},
		{"28_token_retention", "Token Retention (Yale)", "optional"},
		{"29_acon", "ACON (ICLR 2026)", "optional"},
	}
}
