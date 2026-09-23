package chatlog

import (
	"errors"
	"strings"
)

// DefaultRolePrefixes is used when no prefix is given.
var DefaultRolePrefixes = []string{
	"你：", "AI：", "玩家：", "系統：",
	"User:", "Assistant:", "Player:", "System:", "AI:",
}

// ErrNoMessage means no line starts with any role prefix.
var ErrNoMessage = errors.New("找不到任何符合角色前綴的訊息")

// ParseText splits content into messages. A message starts at a line
// beginning with one of prefixes; the role is the prefix minus its last
// character (the colon). Text before the first message is dropped.
func ParseText(content string, prefixes []string) ([]Message, error) {
	if len(prefixes) == 0 {
		prefixes = DefaultRolePrefixes
	}

	var (
		msgs []Message
		role string
		buf  []string
	)
	flush := func() {
		if role != "" {
			msgs = append(msgs, Message{
				Role:    role,
				Content: strings.TrimSpace(strings.Join(buf, "\n")),
			})
		}
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	for line := range strings.SplitSeq(content, "\n") {
		matched := false
		for _, p := range prefixes {
			if !strings.HasPrefix(line, p) {
				continue
			}
			flush()
			r := []rune(p)
			role = string(r[:len(r)-1])
			buf = []string{strings.TrimSpace(line[len(p):])}
			matched = true
			break
		}
		if !matched {
			buf = append(buf, line)
		}
	}
	flush()

	if len(msgs) == 0 {
		return nil, ErrNoMessage
	}
	return msgs, nil
}

// ParsePrefixes reads one prefix per non-empty line.
func ParsePrefixes(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
