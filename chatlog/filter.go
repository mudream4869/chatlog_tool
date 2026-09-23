package chatlog

import (
	"regexp"
	"strconv"
	"strings"
)

// Filter rewrites a message's content.
type Filter func(string) string

var (
	reHTMLComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	reDetails     = regexp.MustCompile(`(?s)<details>.*?</details>`)
	reBr          = regexp.MustCompile(`(?i)<br\s*/?>`)
	reCloseP      = regexp.MustCompile(`(?i)</p>`)
	reTag         = regexp.MustCompile(`<[^>]+>`)
)

// RemoveHTMLComments removes <!-- ... -->.
func RemoveHTMLComments(s string) string {
	return reHTMLComment.ReplaceAllString(s, "")
}

// RemoveDetails removes <details> blocks with their content.
func RemoveDetails(s string) string {
	return reDetails.ReplaceAllString(s, "")
}

// RemoveHTMLTags strips all tags; <br> and </p> become newlines.
func RemoveHTMLTags(s string) string {
	s = reBr.ReplaceAllString(s, "\n")
	s = reCloseP.ReplaceAllString(s, "\n")
	return reTag.ReplaceAllString(s, "")
}

// LimitNewlines caps consecutive newlines at max; max <= 0 is a no-op.
func LimitNewlines(s string, max int) string {
	if max <= 0 {
		return s
	}
	re := regexp.MustCompile(`\n{` + strconv.Itoa(max+1) + `,}`)
	return re.ReplaceAllString(s, strings.Repeat("\n", max))
}

// Apply runs filters over every message and returns new messages, with
// surrounding whitespace left by removed markup trimmed.
func Apply(msgs []Message, filters ...Filter) []Message {
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		for _, f := range filters {
			m.Content = f(m.Content)
		}
		m.Content = strings.TrimSpace(m.Content)
		out[i] = m
	}
	return out
}
