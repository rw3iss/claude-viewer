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

// Prompt is one user prompt + its time-to-first-response + token usage of
// the assistant reply (sum across all iterations).
type Prompt struct {
	Time     time.Time
	Text     string // cleaned single-line for list display
	FullText string // original with newlines preserved (for preview)
	Took     time.Duration
	Pending  bool // no assistant response yet

	// Token usage of the assistant turn following this prompt. Zeros when
	// Pending or when the JSONL didn't carry a usage block.
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	Model               string
}

// TotalInputTokens sums fresh + cache-read + cache-creation input.
func (p Prompt) TotalInputTokens() int {
	return p.InputTokens + p.CacheReadTokens + p.CacheCreationTokens
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
	type usage struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	}
	type assistantMessage struct {
		Model string `json:"model"`
		Usage usage  `json:"usage"`
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
					full := cleanFullText(content)
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
						p := Prompt{
							Time: pend.ts, Text: pend.text, FullText: pend.full,
							Took: ts.Sub(pend.ts),
						}
						if e.Message != nil {
							var am assistantMessage
							if jerr := json.Unmarshal(e.Message, &am); jerr == nil {
								p.Model = am.Model
								p.InputTokens = am.Usage.InputTokens
								p.OutputTokens = am.Usage.OutputTokens
								p.CacheReadTokens = am.Usage.CacheReadInputTokens
								p.CacheCreationTokens = am.Usage.CacheCreationInputTokens
							}
						}
						prompts = append(prompts, p)
						pend = nil
					} else if len(prompts) > 0 && e.Message != nil {
						// Continuation of the previous turn (multi-message
						// reply, e.g. tool use → result → final text).
						// Sum its tokens onto the existing Prompt.
						var am assistantMessage
						if jerr := json.Unmarshal(e.Message, &am); jerr == nil {
							last := &prompts[len(prompts)-1]
							last.InputTokens += am.Usage.InputTokens
							last.OutputTokens += am.Usage.OutputTokens
							last.CacheReadTokens += am.Usage.CacheReadInputTokens
							last.CacheCreationTokens += am.Usage.CacheCreationInputTokens
							if last.Model == "" {
								last.Model = am.Model
							}
						}
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
	// tagBlock = wholesale-strip these blocks (Claude Code system-emitted
	// metadata that the user never wrote and shouldn't see). Includes
	// <command-message> because it's redundant with <command-name>
	// (just the command name minus the leading slash).
	tagBlock = regexp.MustCompile(`(?s)<(system-reminder|local-command-stdout|local-command-stderr|local-command-caveat|command-message)>.*?</[^>]+>`)

	// commandTag = unwrap these to their inner text. <command-name> keeps
	// its leading slash; <command-args> carries any arguments; the
	// surrounding free-form text is the body the user sent along with
	// the command.
	commandTag = regexp.MustCompile(`<(command-name|command-args|command-stdout|command-stderr)>([^<]*)</[^>]+>`)

	multiNL = regexp.MustCompile(`\n+`)
	multiWS = regexp.MustCompile(`\s+`)
)

// cleanText is for the list-row preview: tags stripped, newlines collapsed
// to a visible "⏎" marker, all whitespace squeezed.
func cleanText(s string) string {
	s = tagBlock.ReplaceAllString(s, "")
	s = unwrapCommandTags(s)
	s = multiNL.ReplaceAllString(s, " ⏎ ")
	s = multiWS.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// cleanFullText is for the full-content preview pane: same tag handling
// but newlines/indentation are preserved so the text reads naturally.
func cleanFullText(s string) string {
	s = tagBlock.ReplaceAllString(s, "")
	s = unwrapCommandTags(s)
	return strings.TrimSpace(s)
}

// unwrapCommandTags replaces each <command-X>inner</command-X> with just
// its inner text. So
//   "<command-message>improve</command-message>\n<command-name>/improve</command-name>"
// becomes
//   "improve\n/improve"
// — i.e. what the user actually typed before Claude Code wrapped it.
func unwrapCommandTags(s string) string {
	return commandTag.ReplaceAllString(s, "$2")
}

func isSkippableUserText(s string) bool {
	return strings.HasPrefix(s, "Caveat:")
}
