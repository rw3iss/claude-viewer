package screens

import "github.com/rw3iss/claude-viewer/internal/components"

// helpForMenu returns the help-overlay sections for the main menu.
func helpForMenu() []components.HelpSection {
	return []components.HelpSection{
		{Title: "Navigation", Entries: []components.HelpEntry{
			{Key: "←/→", Desc: "switch organization (page through claude dirs)"},
			{Key: "↑/↓", Desc: "select session"},
			{Key: "home/end", Desc: "first / last session"},
			{Key: "enter", Desc: "open session in chat view"},
		}},
		{Title: "Other screens", Entries: []components.HelpEntry{
			{Key: "a", Desc: "all-orgs view (every dir as side-by-side columns)"},
			{Key: "o", Desc: "settings (enable/disable dirs, add custom)"},
		}},
		{Title: "Misc", Entries: []components.HelpEntry{
			{Key: "r", Desc: "reload current page from disk"},
			{Key: "h / ?", Desc: "this help"},
			{Key: "q / esc", Desc: "quit"},
		}},
	}
}

// helpForAllOrgs returns help for the side-by-side multi-column view.
func helpForAllOrgs() []components.HelpSection {
	return []components.HelpSection{
		{Title: "Navigation", Entries: []components.HelpEntry{
			{Key: "←/→", Desc: "move focus between columns (orgs)"},
			{Key: "↑/↓", Desc: "select session in focused column"},
			{Key: "enter", Desc: "open session"},
		}},
		{Title: "Misc", Entries: []components.HelpEntry{
			{Key: "r", Desc: "reload all dirs"},
			{Key: "h / ?", Desc: "this help"},
			{Key: "a / esc", Desc: "back to menu"},
			{Key: "q", Desc: "quit"},
		}},
	}
}

// helpForSettings returns help for the settings screen.
func helpForSettings() []components.HelpSection {
	return []components.HelpSection{
		{Title: "Directories", Entries: []components.HelpEntry{
			{Key: "↑/↓", Desc: "select row"},
			{Key: "space", Desc: "toggle enabled/disabled for selected dir"},
			{Key: "n", Desc: "add a custom claude config dir (input prompt)"},
			{Key: "d", Desc: "remove selected custom dir (auto-detected ones can't be removed)"},
		}},
		{Title: "Preferences", Entries: []components.HelpEntry{
			{Key: "enter", Desc: "edit selected preference (e.g. Active session window)"},
		}},
		{Title: "Misc", Entries: []components.HelpEntry{
			{Key: "h / ?", Desc: "this help"},
			{Key: "esc", Desc: "menu (back)"},
			{Key: "q", Desc: "quit"},
		}},
	}
}

// helpForChat returns help for the session detail view.
func helpForChat() []components.HelpSection {
	return []components.HelpSection{
		{Title: "Navigation", Entries: []components.HelpEntry{
			{Key: "↑/↓", Desc: "select prompt"},
			{Key: "pgup/pgdn", Desc: "jump 10"},
			{Key: "home/end", Desc: "first / last"},
			{Key: "enter", Desc: "open the highlighted prompt full-screen"},
		}},
		{Title: "Actions", Entries: []components.HelpEntry{
			{Key: "f / /", Desc: "toggle search filter"},
			{Key: "c", Desc: "copy highlighted prompt to clipboard"},
			{Key: "e", Desc: "export highlighted prompt to current directory"},
			{Key: "r", Desc: "reload from disk"},
		}},
		{Title: "Layout", Entries: []components.HelpEntry{
			{Key: "l", Desc: "toggle bottom ↔ right preview layout"},
			{Key: "ctrl+↑/↓", Desc: "wrap rows per prompt (1–8)"},
			{Key: "alt+↑/↓", Desc: "grow/shrink the preview pane"},
		}},
		{Title: "Misc", Entries: []components.HelpEntry{
			{Key: "h / ?", Desc: "this help"},
			{Key: "esc", Desc: "menu (back) — closes full-screen view first if open"},
			{Key: "q", Desc: "quit"},
		}},
	}
}
