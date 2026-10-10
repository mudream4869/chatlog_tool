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
	Field   bool     // looks like a status field, e.g. "姓名："
}

const (
	maxLabelLen   = 12 // runes before the colon
	minPrefixN    = 2
	maxSuggest    = 8
	maxSamples    = 3
	maxSampleLen  = 40
	minShareOfTop = 0.2 // Likely drops prefixes rarer than this share of the top one
	minListLen    = 3   // distinct labels in a row that make a field list
	minFieldShare = 0.8 // share of a label's lines in field lists
)

// labelBreak may not appear in a role name; such lines are prose or URLs.
const labelBreak = "，,。.！!？?；;、「」『』\"“”'‘’()（）[]【】<>《》/\\=*#"

// knownRoles are never taken for status fields.
var knownRoles = []string{
	"您", "你", "玩家", "用戶", "用户", "系統", "系统", "旁白",
	"ai", "user", "assistant", "system", "player", "human", "narrator", "gm",
}

// SuggestPrefixes finds line-start "名稱：" / "Name:" prefixes used at least
// twice, most frequent first. Lines in <details>, HTML comments and code
// fences are skipped, as status blocks live there.
func SuggestPrefixes(content string) []PrefixSuggestion {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	labels := make([]string, len(lines)) // prefix per line, "" if none
	skip := blockSkipper()
	for i, line := range lines {
		if skip(line) {
			continue
		}
		if p, ok := prefixOf(line); ok {
			labels[i] = p
		}
	}
	inList := fieldLists(lines, labels)

	idx := map[string]int{}
	var out []PrefixSuggestion
	listed := map[string]int{}
	for i, p := range labels {
		if p == "" {
			continue
		}
		j, seen := idx[p]
		if !seen {
			j = len(out)
			idx[p] = j
			out = append(out, PrefixSuggestion{Prefix: p})
		}
		s := &out[j]
		s.Count++
		if len(s.Samples) < maxSamples {
			s.Samples = append(s.Samples, clip(lines[i], maxSampleLen))
		}
		if inList[i] {
			listed[p]++
		}
	}
	for i := range out {
		s := &out[i]
		s.Field = float64(listed[s.Prefix]) >= minFieldShare*float64(s.Count) &&
			!slices.Contains(knownRoles, strings.ToLower(RoleOf(s.Prefix)))
	}

	out = slices.DeleteFunc(out, func(s PrefixSuggestion) bool { return s.Count < minPrefixN })
	// Stable: ties keep first appearance.
	slices.SortStableFunc(out, func(a, b PrefixSuggestion) int { return b.Count - a.Count })
	return out[:min(len(out), maxSuggest)]
}

// blockSkipper reports, line by line, whether a line is inside a <details>,
// an HTML comment or a code fence.
func blockSkipper() func(string) bool {
	details, comment, fence := 0, false, false
	return func(line string) bool {
		in := details > 0 || comment || fence
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fence = !fence
			return true
		}
		details = max(0, details+strings.Count(line, "<details")-strings.Count(line, "</details>"))
		if comment {
			comment = !strings.Contains(line, "-->")
		} else if i := strings.LastIndex(line, "<!--"); i >= 0 {
			comment = !strings.Contains(line[i:], "-->")
		}
		return in
	}
}

// fieldLists marks lines in runs of at least minListLen prefixed lines, blank
// lines aside, with no label repeated: a status list, not a dialogue. A
// repeated label starts a new run.
func fieldLists(lines, labels []string) []bool {
	in := make([]bool, len(lines))
	var run []int
	seen := map[string]bool{}
	flush := func() {
		if len(run) >= minListLen {
			for _, i := range run {
				in[i] = true
			}
		}
		run = run[:0]
		clear(seen)
	}
	for i, l := range labels {
		switch {
		case l != "":
			if seen[l] {
				flush()
			}
			run = append(run, i)
			seen[l] = true
		case strings.TrimSpace(lines[i]) != "":
			flush()
		}
	}
	flush()
	return in
}

// LikelyPrefixes keeps suggestions common enough to be roles, leaving out
// status fields unless nothing else is left.
func LikelyPrefixes(sugs []PrefixSuggestion) []string {
	roles := slices.DeleteFunc(slices.Clone(sugs), func(s PrefixSuggestion) bool { return s.Field })
	if len(roles) == 0 {
		roles = sugs
	}
	var out []string
	for _, s := range roles {
		if float64(s.Count) >= minShareOfTop*float64(roles[0].Count) {
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
