package app

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/events"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/screens"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Model is the root tea.Model. It holds the currently-active screen plus
// shared dependencies (repo, theme, config) and handles SwitchScreenMsg.
// memTickMsg fires every memTickInterval to refresh the mem-usage footer.
type memTickMsg struct{}

const memTickInterval = 2 * time.Second

func memTickCmd() tea.Cmd {
	return tea.Tick(memTickInterval, func(time.Time) tea.Msg { return memTickMsg{} })
}

type Model struct {
	repo  data.Repository
	cfg   *config.Config
	theme theme.Theme
	keys  keys.Map

	width, height int
	current       screens.Screen

	// Live process memory usage (heap-alloc bytes), refreshed via memTickCmd.
	memBytes uint64
}

// Deps bundles the dependencies needed to construct Model.
type Deps struct {
	Repo  data.Repository
	Cfg   *config.Config
	Theme theme.Theme
	Keys  keys.Map

	// InitialSession, if non-nil, opens straight into the chat screen.
	InitialSession *data.Session
	InitialDir     *data.ClaudeDir
}

// New creates the root model.
func New(d Deps) *Model {
	m := &Model{repo: d.Repo, cfg: d.Cfg, theme: d.Theme, keys: d.Keys}
	if d.InitialSession != nil && d.InitialDir != nil {
		m.current = screens.NewChat(d.Repo, d.Cfg, d.Theme, d.Keys, *d.InitialSession, *d.InitialDir)
	} else {
		m.current = screens.NewMenu(d.Repo, d.Cfg, d.Theme, d.Keys)
	}
	return m
}

// Init satisfies tea.Model.
func (m *Model) Init() tea.Cmd {
	m.refreshMem()
	return tea.Batch(m.current.Init(), memTickCmd())
}

func (m *Model) refreshMem() {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	m.memBytes = s.HeapAlloc
}

// Update routes messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.current.SetSize(m.width, m.height)
		return m, nil
	case memTickMsg:
		m.refreshMem()
		return m, memTickCmd()
	case events.QuitAppMsg:
		return m, tea.Quit
	case events.SwitchScreenMsg:
		switch msg.To {
		case events.ScreenMenu:
			m.current = screens.NewMenu(m.repo, m.cfg, m.theme, m.keys)
		case events.ScreenAllOrgs:
			m.current = screens.NewAllOrgs(m.repo, m.cfg, m.theme, m.keys)
		case events.ScreenSettings:
			m.current = screens.NewSettings(m.repo, m.cfg, m.theme, m.keys)
		case events.ScreenChat:
			if msg.Session != nil && msg.Dir != nil {
				m.current = screens.NewChat(m.repo, m.cfg, m.theme, m.keys, *msg.Session, *msg.Dir)
			}
		}
		m.current.SetSize(m.width, m.height)
		return m, m.current.Init()
	}
	next, cmd := m.current.Update(msg)
	m.current = next
	return m, cmd
}

// View renders the current screen and overlays the mem-usage footer in the
// bottom-right corner.
func (m *Model) View() string {
	body := m.current.View()
	if m.width <= 0 {
		return body
	}
	memText := m.theme.Dim().Render(fmt.Sprintf("mem: %s", fmtMemory(m.memBytes)))
	memW := lipgloss.Width(memText)

	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return body
	}
	lastIdx := len(lines) - 1
	last := lines[lastIdx]
	lastW := lipgloss.Width(last)

	// If the existing last line has space, right-align mem on it. Otherwise
	// append on a fresh line.
	if lastW+memW+2 <= m.width {
		pad := m.width - lastW - memW
		lines[lastIdx] = last + strings.Repeat(" ", pad) + memText
	} else {
		pad := m.width - memW
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, strings.Repeat(" ", pad)+memText)
	}
	return strings.Join(lines, "\n")
}

// fmtMemory formats bytes as KB/MB/GB.
func fmtMemory(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
