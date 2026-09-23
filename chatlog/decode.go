package chatlog

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/encoding/traditionalchinese"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Decode turns raw file bytes into text: UTF-8 (with or without BOM) if
// valid, otherwise Big5.
func Decode(b []byte) string {
	b = bytes.TrimPrefix(b, utf8BOM)
	if utf8.Valid(b) {
		return string(b)
	}

	out, err := traditionalchinese.Big5.NewDecoder().Bytes(b)
	if err != nil {
		return string(bytes.ToValidUTF8(b, []byte("�")))
	}
	return string(out)
}
