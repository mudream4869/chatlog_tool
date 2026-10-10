// Package chatlog parses, cleans and exports chat logs.
package chatlog

import "strings"

// Message is one turn of a conversation.
type Message struct {
	Role    string
	Content string
	IsUser  bool // known user side, e.g. SillyTavern is_user
}

// IsUserRole reports whether role belongs to the user side. Roles have their
// colon and brackets removed, so userPrefix is matched the same way.
func IsUserRole(role, userPrefix string) bool {
	p := unwrap(strings.TrimRight(userPrefix, "：:"))
	return (p != "" && strings.HasPrefix(role, p)) ||
		strings.Contains(role, "您") ||
		strings.Contains(strings.ToLower(role), "user") ||
		strings.Contains(role, "用戶")
}

// RoleCount is the number of messages a role has.
type RoleCount struct {
	Role  string
	Count int
}

// CountRoles counts messages per role, in order of first appearance.
func CountRoles(msgs []Message) []RoleCount {
	idx := map[string]int{}
	var out []RoleCount
	for _, m := range msgs {
		i, ok := idx[m.Role]
		if !ok {
			i = len(out)
			idx[m.Role] = i
			out = append(out, RoleCount{Role: m.Role})
		}
		out[i].Count++
	}
	return out
}

// CountRunes returns the total number of characters in all contents.
func CountRunes(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += len([]rune(m.Content))
	}
	return n
}
