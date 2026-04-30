package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/rw3iss/claude-viewer/internal/events"
	"github.com/rw3iss/claude-viewer/internal/clipboard"
	"github.com/rw3iss/claude-viewer/internal/components"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	dbg "github.com/rw3iss/claude-viewer/internal/debug"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Chat = the session detail screen. Lists prompts (newest-first, wrap-N) on
// the left/top, full content of selected prompt on the right/bottom.
type Chat struct {
	repo  data.Repository
	cfg   *config.Config
	theme theme.Theme
	keys  keys.Map

	width, height int
	session       data.Session
	dir           data.ClaudeDir

	prompts []data.Prompt
	cursor  int

	layout       string // "bottom" | "right"
	previewSize  int    // 30..80
	previewRows  int    // 1..8 (wrapped lines per prompt in list)
	searchActive bool
	searchInput  textinput.Model
	filter       string

	preview viewport.Model

	alert       components.Alert
	watcher     *fsnotify.Watcher
	watchEvents chan struct{}
	helpVisible bool
}

// NewChat constructs a chat-detail screen.
func NewChat(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map, s data.Session, d data.ClaudeDir) *Chat {
	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Width = 40
	c := &Chat{
		repo:        repo,
		cfg:         cfg,
		theme:       t,
		keys:        k,
		session:     s,
		dir:         d,
		layout:      cfg.Layout,
		previewSize: cfg.PreviewSize,
		previewRows: cfg.PreviewRows,
		searchInput: ti,
	}
	if c.layout == "" {
		c.layout = "bottom"
	}
	if c.previewSize == 0 {
		c.previewSize = 60
	}
	if c.previewRows == 0 {
		c.previewRows = 2
	}
	c.preview = viewport.New(40, 10)
	c.loadPrompts()
	c.startWatcher()
	return c
}

func (c *Chat) loadPrompts() {
	t := time.Now()
	prompts, err := data.LoadPrompts(c.session.Path)
	if err != nil {
		dbg.Logf("Chat.loadPrompts: %v", err)
		c.alert = components.Alert{Text: fmt.Sprintf("load: %v", err), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
		return
	}
	dbg.Logf("Chat.loadPrompts: session=%s n=%d in=%s", c.session.UUID, len(prompts), time.Since(t).Truncate(time.Millisecond))
	c.prompts = prompts
	if c.cursor >= len(c.prompts) {
		c.cursor = 0
	}
	c.refreshPreview()
}

func (c *Chat) refreshPreview() {
	if c.cursor >= len(c.filteredPrompts()) {
		c.preview.SetContent("")
		return
	}
	p := c.filteredPrompts()[c.cursor]

	wrapW := c.preview.Width - 4
	if wrapW < 20 {
		wrapW = 20
	}
	wrapStyle := lipgloss.NewStyle().Width(wrapW)

	var b strings.Builder
	if meta := previewMetaLine(c.theme, p); meta != "" {
		b.WriteString("  ")
		b.WriteString(meta)
		b.WriteString("\n\n")
	}
	for i, line := range strings.Split(p.FullText, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		wrapped := wrapStyle.Render(line)
		for j, sub := range strings.Split(wrapped, "\n") {
			if j > 0 {
				b.WriteString("\n")
			}
			b.WriteString("  ")
			b.WriteString(sub)
		}
	}
	c.preview.SetContent(b.String())
	c.preview.GotoTop()
}

// previewMetaLine renders a dim 1-line summary atop the preview pane:
// "claude-sonnet-4-6 · 21s · ↑3.2k (cache 18k) · ↓329".
func previewMetaLine(t theme.Theme, p data.Prompt) string {
	parts := []string{}
	if p.Model != "" {
		parts = append(parts, t.AccentAlt().Render(p.Model))
	}
	if !p.Pending && p.Took > 0 {
		parts = append(parts, t.Subtitle().Render(formatDelta(p.Took, false)))
	}
	if p.InputTokens > 0 || p.CacheReadTokens > 0 || p.CacheCreationTokens > 0 {
		in := fmt.Sprintf("↑%s", fmtTokens(p.TotalInputTokens()))
		if p.CacheReadTokens > 0 || p.CacheCreationTokens > 0 {
			in += t.Dim().Render(fmt.Sprintf(" (cache %s)", fmtTokens(p.CacheReadTokens+p.CacheCreationTokens)))
		}
		parts = append(parts, in)
	}
	if p.OutputTokens > 0 {
		parts = append(parts, fmt.Sprintf("↓%s", fmtTokens(p.OutputTokens)))
	}
	if len(parts) == 0 {
		return ""
	}
	sep := t.Dim().Render(" · ")
	return t.Dim().Render(strings.Join(parts, sep))
}

// fmtTokens formats an int as 1.2k / 3.4M / 999.
func fmtTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (c *Chat) filteredPrompts() []data.Prompt {
	if c.filter == "" {
		return c.prompts
	}
	q := strings.ToLower(c.filter)
	out := make([]data.Prompt, 0, len(c.prompts))
	for _, p := range c.prompts {
		if strings.Contains(strings.ToLower(p.Text), q) || strings.Contains(strings.ToLower(p.FullText), q) {
			out = append(out, p)
		}
	}
	return out
}

// Init satisfies tea.Model.
func (c *Chat) Init() tea.Cmd { return waitFsTick(c.watchEvents) }

// SetSize updates dimensions.
func (c *Chat) SetSize(w, h int) {
	c.width, c.height = w, h
	c.recomputePanes()
}

func (c *Chat) recomputePanes() {
	listW, listH, prevW, prevH := c.paneSizes()
	c.preview.Width = prevW
	c.preview.Height = prevH
	_ = listW
	_ = listH
}

func (c *Chat) paneSizes() (listW, listH, prevW, prevH int) {
	bodyH := c.height - 5
	if bodyH < 5 {
		bodyH = 5
	}
	if c.layout == "right" {
		prevW = c.width * c.previewSize / 100
		listW = c.width - prevW - 1
		prevH = bodyH
		listH = bodyH
	} else {
		prevH = bodyH * c.previewSize / 100
		listH = bodyH - prevH - 1
		prevW = c.width - 2
		listW = c.width - 2
	}
	return
}

// Update handles input.
func (c *Chat) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if c.searchActive {
		var cmd tea.Cmd
		if m, ok := msg.(tea.KeyMsg); ok {
			switch m.String() {
			case "esc":
				c.searchActive = false
				c.searchInput.Blur()
				c.searchInput.SetValue("")
				c.filter = ""
				c.cursor = 0
				c.refreshPreview()
				return c, nil
			case "enter":
				c.searchActive = false
				c.searchInput.Blur()
				c.cursor = 0
				c.refreshPreview()
				return c, nil
			}
		}
		c.searchInput, cmd = c.searchInput.Update(msg)
		c.filter = c.searchInput.Value()
		c.cursor = 0
		c.refreshPreview()
		return c, cmd
	}

	switch msg := msg.(type) {
	case fsTickMsg:
		c.loadPrompts()
		return c, waitFsTick(c.watchEvents)
	case components.AlertExpiredMsg:
		c.alert = components.Alert{}
	case tea.KeyMsg:
		if c.helpVisible {
			if key.Matches(msg, c.keys.Help) || key.Matches(msg, c.keys.Esc) {
				c.helpVisible = false
			}
			return c, nil
		}
		filtered := c.filteredPrompts()
		switch {
		case key.Matches(msg, c.keys.Help):
			c.helpVisible = true
			return c, nil
		case key.Matches(msg, c.keys.Quit):
			c.stopWatcher()
			return c, func() tea.Msg { return events.QuitAppMsg{} }
		case key.Matches(msg, c.keys.Esc):
			c.stopWatcher()
			return c, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenMenu} }
		case key.Matches(msg, c.keys.Up):
			if c.cursor > 0 {
				c.cursor--
				c.refreshPreview()
			}
		case key.Matches(msg, c.keys.Down):
			if c.cursor < len(filtered)-1 {
				c.cursor++
				c.refreshPreview()
			}
		case key.Matches(msg, c.keys.Home):
			c.cursor = 0
			c.refreshPreview()
		case key.Matches(msg, c.keys.End):
			c.cursor = len(filtered) - 1
			if c.cursor < 0 {
				c.cursor = 0
			}
			c.refreshPreview()
		case key.Matches(msg, c.keys.PageUp):
			c.cursor -= 10
			if c.cursor < 0 {
				c.cursor = 0
			}
			c.refreshPreview()
		case key.Matches(msg, c.keys.PageDown):
			c.cursor += 10
			if c.cursor >= len(filtered) {
				c.cursor = len(filtered) - 1
			}
			c.refreshPreview()
		case key.Matches(msg, c.keys.Reload):
			c.loadPrompts()
			c.alert = components.Alert{Text: "reloaded", Level: components.AlertOK, Expires: time.Now().Add(2 * time.Second)}
			return c, components.AlertCmd(time.Now().UnixNano(), 2*time.Second)
		case key.Matches(msg, c.keys.Layout):
			if c.layout == "bottom" {
				c.layout = "right"
			} else {
				c.layout = "bottom"
			}
			c.cfg.Layout = c.layout
			_ = config.Save(c.cfg)
			c.recomputePanes()
		case key.Matches(msg, c.keys.PaneUp):
			if c.previewSize < 80 {
				c.previewSize += 5
				c.cfg.PreviewSize = c.previewSize
				_ = config.Save(c.cfg)
				c.recomputePanes()
			}
		case key.Matches(msg, c.keys.PaneDown):
			if c.previewSize > 30 {
				c.previewSize -= 5
				c.cfg.PreviewSize = c.previewSize
				_ = config.Save(c.cfg)
				c.recomputePanes()
			}
		case key.Matches(msg, c.keys.RowsUp):
			if c.previewRows < 8 {
				c.previewRows++
				c.cfg.PreviewRows = c.previewRows
				_ = config.Save(c.cfg)
			}
		case key.Matches(msg, c.keys.RowsDown):
			if c.previewRows > 1 {
				c.previewRows--
				c.cfg.PreviewRows = c.previewRows
				_ = config.Save(c.cfg)
			}
		case key.Matches(msg, c.keys.Search):
			c.searchActive = true
			c.searchInput.SetValue(c.filter)
			c.searchInput.Focus()
			return c, textinput.Blink
		case key.Matches(msg, c.keys.Copy):
			if c.cursor < len(filtered) {
				if err := clipboard.Copy(filtered[c.cursor].FullText); err != nil {
					c.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				} else {
					c.alert = components.Alert{Text: "copied to clipboard", Level: components.AlertOK, Expires: time.Now().Add(2 * time.Second)}
				}
				return c, components.AlertCmd(time.Now().UnixNano(), 2*time.Second)
			}
		case key.Matches(msg, c.keys.Save):
			if c.cursor < len(filtered) {
				name := fmt.Sprintf("claude-prompt-%s.txt", filtered[c.cursor].Time.Format("2006-01-02-150405"))
				cwd, _ := os.Getwd()
				out := filepath.Join(cwd, name)
				if err := os.WriteFile(out, []byte(filtered[c.cursor].FullText), 0o644); err != nil {
					c.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				} else {
					c.alert = components.Alert{Text: "saved " + name, Level: components.AlertOK, Expires: time.Now().Add(3 * time.Second)}
				}
				return c, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
			}
		}
	}
	return c, nil
}

// View renders the screen.
func (c *Chat) View() string {
	// Pre-WindowSizeMsg or absurdly small terminal — show a placeholder
	// rather than running geometry math with negative numbers.
	if c.width < 20 || c.height < 8 {
		return c.theme.Dim().Render("claude-viewer: initializing…")
	}
	if c.helpVisible {
		return components.RenderHelp(c.theme, components.HelpInput{
			Title:    "Chat — Help",
			Subtitle: c.session.Display(),
			Sections: helpForChat(),
			Width:    c.width, Height: c.height,
		})
	}
	hint := "↑/↓ nav · enter open · ctrl+f search · ctrl+y copy · ctrl+o save · ctrl+l layout · ctrl+↑/↓ rows · alt+↑/↓ pane · h help · ctrl+r reload · esc menu"
	header := components.Header(c.theme, *c.cfg, components.HeaderInput{
		Session: &c.session,
		Dir:     &c.dir,
		HintRow: hint,
		Width:   c.width,
	})

	listW, listH, prevW, prevH := c.paneSizes()
	listView := c.renderList(listW, listH)
	previewView := c.renderPreview(prevW, prevH)

	var body string
	if c.layout == "right" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, listView, c.theme.Border().Render("│"), previewView)
	} else {
		borderW := c.width - 2
		if borderW < 1 {
			borderW = 1
		}
		body = listView + "\n" + c.theme.Border().Render(strings.Repeat("─", borderW)) + "\n" + previewView
	}

	var footer string
	if c.searchActive {
		footer = c.theme.Highlight().Render("/ "+c.searchInput.View())
	} else {
		footer = components.RenderAlert(c.theme, c.alert)
	}
	return header + "\n\n" + body + "\n" + footer
}

func (c *Chat) renderList(w, h int) string {
	filtered := c.filteredPrompts()
	if len(filtered) == 0 {
		return c.theme.Dim().Render("(no prompts)")
	}
	prefixW := 18 // "HH:MM:SS  TOOK  "
	textW := w - prefixW
	if textW < 20 {
		textW = 20
	}
	var lines []string
	rowsUsed := 0
	lastDate := ""
	cursorAdj := c.cursor

	// Determine view window centered around cursor.
	// Build all rendered blocks first, then take the slice that fits.
	type block struct {
		lines    []string
		isCursor bool
	}
	var blocks []block
	cursorBlockIdx := 0
	for i, p := range filtered {
		date := p.Time.Format("2006-01-02")
		var rendered []string
		// firstMsgIdx points at the first line of the prompt itself within
		// `rendered`. If a date-separator line was emitted, that's at index
		// 0 and the prompt starts at 1 — otherwise the prompt is at 0.
		firstMsgIdx := 0
		if date != lastDate {
			rendered = append(rendered, c.theme.Dim().Render("── "+date+" ──"))
			lastDate = date
			firstMsgIdx = 1
		}
		took := formatDelta(p.Took, p.Pending)
		prefix := c.theme.Dim().Render(p.Time.Format("15:04:05")) + "  " + c.theme.Subtitle().Render(fmt.Sprintf("%-7s", took)) + "  "
		wrapped := wrapText(p.Text, textW, c.previewRows)
		for j, wl := range wrapped {
			if j == 0 {
				rendered = append(rendered, prefix+wl)
			} else {
				rendered = append(rendered, strings.Repeat(" ", prefixW)+wl)
			}
		}
		rendered = append(rendered, "")
		if i == cursorAdj {
			cursorBlockIdx = len(blocks)
			rendered[firstMsgIdx] = c.theme.Selected().Render(rendered[firstMsgIdx])
		}
		blocks = append(blocks, block{lines: rendered, isCursor: i == cursorAdj})
	}

	// Render blocks until height hit, scrolling so cursor stays visible.
	startBlock := 0
	if cursorBlockIdx > 0 {
		// Estimate roughly: cursor near top
		startBlock = 0
	}
	for i := startBlock; i < len(blocks); i++ {
		blockHeight := len(blocks[i].lines)
		if rowsUsed+blockHeight > h {
			break
		}
		lines = append(lines, blocks[i].lines...)
		rowsUsed += blockHeight
	}
	return strings.Join(lines, "\n")
}

func (c *Chat) renderPreview(w, h int) string {
	box := lipgloss.NewStyle().
		Width(w).
		Height(h)
	c.preview.Width = w
	c.preview.Height = h
	return box.Render(c.preview.View())
}

func wrapText(text string, w, max int) []string {
	if w < 10 {
		w = 10
	}
	var out []string
	for len(text) > 0 && len(out) < max {
		if len(text) <= w {
			out = append(out, text)
			text = ""
			break
		}
		// find break point near w (last space within 20 chars before w)
		bp := w
		for i := w; i > w-20 && i > 0; i-- {
			if text[i] == ' ' {
				bp = i
				break
			}
		}
		out = append(out, strings.TrimRight(text[:bp], " "))
		text = strings.TrimLeft(text[bp:], " ")
	}
	if len(text) > 0 && len(out) > 0 {
		last := out[len(out)-1]
		if len(last) > w-1 {
			out[len(out)-1] = last[:w-1] + "…"
		} else {
			out[len(out)-1] = last + "…"
		}
	}
	return out
}

func formatDelta(d time.Duration, pending bool) string {
	if pending {
		return "…"
	}
	if d <= 0 {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) - m*60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	return fmt.Sprintf("%dh%dm", h, m)
}

// fsnotify integration ------------------------------------------------------

type fsTickMsg struct{}

func (c *Chat) startWatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	c.watcher = w
	c.watchEvents = make(chan struct{}, 8)
	if err := w.Add(c.session.Path); err != nil {
		w.Close()
		c.watcher = nil
		return
	}
	go func() {
		debounce := time.NewTimer(time.Hour)
		debounce.Stop()
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					debounce.Reset(300 * time.Millisecond)
				}
			case <-debounce.C:
				select {
				case c.watchEvents <- struct{}{}:
				default:
				}
			case <-w.Errors:
			}
		}
	}()
}

func (c *Chat) stopWatcher() {
	if c.watcher != nil {
		c.watcher.Close()
		c.watcher = nil
	}
}

func waitFsTick(events chan struct{}) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		<-events
		return fsTickMsg{}
	}
}
