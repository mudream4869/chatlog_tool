package chatlog

import "strings"

// TxtOptions controls ToTxt.
type TxtOptions struct {
	MaxNewlines int  // <= 0 means unlimited
	SplitLines  bool // add "---" between messages
}

// ToTxt renders messages as plain text.
func ToTxt(msgs []Message, opt TxtOptions) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Role + "：\n" + m.Content + "\n\n")
		if opt.SplitLines {
			b.WriteString("---\n\n")
		}
	}
	return strings.TrimSpace(LimitNewlines(b.String(), opt.MaxNewlines))
}
