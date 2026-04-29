package data

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// Prompt is one user prompt + its time-to-first-response.
type Prompt struct {
	Time      time.Time
	Text      string // cleaned single-line for list display
	FullText  string // original with newlines preserved (for preview)
	Took      time.Duration
	Pending   bool // no assistant response yet
}

// LoadPrompts parses a JSONL session file and pairs user prompts with the
// next assistant message, returning newest-first.
func LoadPrompts(path string) ([]Prompt, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type entry struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		Message   json.RawMessage `json:"message"`
	}
	type userMessage struct {
		Content json.RawMessage `json:"content"`
	}

	var prompts []Prompt
	type pending struct {
		ts   time.Time
		text string
		full string
	}
	var pend *pending

	br := bufio.NewReaderSize(f, 256*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			var e entry
			if jerr := json.Unmarshal(line, &e); jerr == nil {
				ts, ok := parseTime(e.Timestamp)
				if !ok {
					goto cont
				}
				switch e.Type {
				case "user":
					if e.Message == nil {
						goto cont
					}
					var um userMessage
					if jerr := json.Unmarshal(e.Message, &um); jerr != nil {
						goto cont
					}
					var content string
					if jerr := json.Unmarshal(um.Content, &content); jerr != nil {
						// non-string content (tool result) → skip
						goto cont
					}
					if isSkippableUserText(content) {
						goto cont
					}
					full := content
					cleaned := cleanText(content)
					if cleaned == "" {
						goto cont
					}
					if pend != nil {
						prompts = append(prompts, Prompt{
							Time: pend.ts, Text: pend.text, FullText: pend.full, Pending: true,
						})
					}
					pend = &pending{ts: ts, text: cleaned, full: full}
				case "assistant":
					if pend != nil {
						prompts = append(prompts, Prompt{
							Time: pend.ts, Text: pend.text, FullText: pend.full,
							Took: ts.Sub(pend.ts),
						})
						pend = nil
					}
				}
			}
		}
	cont:
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	if pend != nil {
		prompts = append(prompts, Prompt{
			Time: pend.ts, Text: pend.text, FullText: pend.full, Pending: true,
		})
	}

	// Reverse for newest-first
	for i, j := 0, len(prompts)-1; i < j; i, j = i+1, j-1 {
		prompts[i], prompts[j] = prompts[j], prompts[i]
	}
	return prompts, nil
}

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var (
	tagBlock = regexp.MustCompile(`(?s)<(system-reminder|local-command-stdout|local-command-stderr|local-command-caveat)>.*?</[^>]+>`)
	tagInline = regexp.MustCompile(`<(command-message|command-args)>[^<]*</[^>]+>`)
	cmdName  = regexp.MustCompile(`<command-name>([^<]*)</command-name>`)
	multiNL  = regexp.MustCompile(`\n+`)
	multiWS  = regexp.MustCompile(`\s+`)
)

func cleanText(s string) string {
	s = tagBlock.ReplaceAllString(s, "")
	s = tagInline.ReplaceAllString(s, "")
	s = cmdName.ReplaceAllString(s, "/$1")
	s = multiNL.ReplaceAllString(s, " ⏎ ")
	s = multiWS.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func isSkippableUserText(s string) bool {
	return strings.HasPrefix(s, "Caveat:")
}
