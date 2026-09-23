// Package chatlog parses, cleans and exports chat logs.
package chatlog

import "strings"

// Message is one turn of a conversation.
type Message struct {
	Role    string
	Content string
}

// IsUserRole reports whether role belongs to the user side. Roles have their
// colon removed, so userPrefix is matched without its trailing colon.
func IsUserRole(role, userPrefix string) bool {
	p := strings.TrimRight(userPrefix, "：:")
	return (p != "" && strings.HasPrefix(role, p)) ||
		strings.Contains(role, "您") ||
		strings.Contains(role, "User") ||
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
