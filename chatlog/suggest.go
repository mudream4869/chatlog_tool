package chatlog

import (
	"slices"
	"strings"
	"unicode"
)

// PrefixSuggestion is a likely role prefix found in a text log.
type PrefixSuggestion struct {
	Prefix  string   // incl. its colon
	Count   int      // lines starting with it
	Samples []string // first such lines
}

const (
	maxLabelLen   = 12 // runes before the colon
	minPrefixN    = 2
	maxSuggest    = 8
	maxSamples    = 3
	maxSampleLen  = 40
	minShareOfTop = 0.2 // Likely drops prefixes rarer than this share of the top one
)

// labelBreak may not appear in a role name; such lines are prose or URLs.
const labelBreak = "，,。.！!？?；;、「」『』\"“”'‘’()（）[]【】<>《》/\\=*#"

// SuggestPrefixes finds line-start "名稱：" / "Name:" prefixes used at least
// twice, most frequent first.
func SuggestPrefixes(content string) []PrefixSuggestion {
	idx := map[string]int{}
	var out []PrefixSuggestion
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for line := range strings.SplitSeq(content, "\n") {
		p, ok := prefixOf(line)
		if !ok {
			continue
		}
		i, seen := idx[p]
		if !seen {
			i = len(out)
			idx[p] = i
			out = append(out, PrefixSuggestion{Prefix: p})
		}
		s := &out[i]
		s.Count++
		if len(s.Samples) < maxSamples {
			s.Samples = append(s.Samples, clip(line, maxSampleLen))
		}
	}

	out = slices.DeleteFunc(out, func(s PrefixSuggestion) bool { return s.Count < minPrefixN })
	// Stable: ties keep first appearance.
	slices.SortStableFunc(out, func(a, b PrefixSuggestion) int { return b.Count - a.Count })
	return out[:min(len(out), maxSuggest)]
}

// LikelyPrefixes keeps suggestions common enough to be roles.
func LikelyPrefixes(sugs []PrefixSuggestion) []string {
	var out []string
	for _, s := range sugs {
		if float64(s.Count) >= minShareOfTop*float64(sugs[0].Count) {
			out = append(out, s.Prefix)
		}
	}
	return out
}

// prefixOf returns the role prefix a line starts with, if it looks like one.
func prefixOf(line string) (string, bool) {
	rs := []rune(line)
	for i, r := range rs {
		if i > maxLabelLen {
			return "", false
		}
		if r == '：' || r == ':' {
			url := i+1 < len(rs) && rs[i+1] == '/' // "http://"
			return string(rs[:i+1]), !url && validLabel(rs[:i])
		}
	}
	return "", false
}

func validLabel(rs []rune) bool {
	rs = []rune(unwrap(string(rs))) // "[AI]"
	if len(rs) == 0 || unicode.IsSpace(rs[0]) || unicode.IsSpace(rs[len(rs)-1]) {
		return false
	}
	digits := true
	for _, r := range rs {
		if strings.ContainsRune(labelBreak, r) || unicode.IsControl(r) {
			return false
		}
		if !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			digits = false
		}
	}
	return !digits // e.g. "12:30"
}

func clip(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n]) + "…"
}
