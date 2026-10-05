package chatlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
)

// ErrNotSillyTavern means the content is not a SillyTavern chat (.jsonl).
var ErrNotSillyTavern = errors.New("不是 SillyTavern 聊天紀錄（.jsonl）格式")

// IsSillyTavernFile reports whether name has the .jsonl extension used by
// SillyTavern chat exports.
func IsSillyTavernFile(name string) bool {
	return strings.EqualFold(path.Ext(name), ".jsonl")
}

type stLine struct {
	ChatMetadata json.RawMessage `json:"chat_metadata"`
	Name         string          `json:"name"`
	IsUser       bool            `json:"is_user"`
	Mes          *string         `json:"mes"`
}

// ParseSillyTavern parses a SillyTavern chat export: one JSON object per
// line, the first being a chat_metadata header. Messages with empty text
// are dropped.
func ParseSillyTavern(content string) ([]Message, error) {
	var msgs []Message
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for i, line := range strings.Split(content, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		var l stLine
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			return nil, fmt.Errorf("%w：第 %d 行不是有效的 JSON", ErrNotSillyTavern, i+1)
		}
		if l.ChatMetadata != nil {
			continue
		}
		if l.Mes == nil {
			return nil, fmt.Errorf("%w：第 %d 行缺少 mes 欄位", ErrNotSillyTavern, i+1)
		}
		if strings.TrimSpace(*l.Mes) == "" {
			continue
		}
		msgs = append(msgs, Message{
			Role:    l.Name,
			Content: strings.TrimSpace(*l.Mes),
			IsUser:  l.IsUser,
		})
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("%w：找不到任何訊息", ErrNotSillyTavern)
	}
	return msgs, nil
}
