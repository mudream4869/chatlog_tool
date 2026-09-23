package chatlog

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/traditionalchinese"
)

func TestDecode(t *testing.T) {
	big5, err := traditionalchinese.Big5.NewEncoder().String("您：你好")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"utf8":    []byte("您：你好"),
		"utf8bom": append([]byte{0xEF, 0xBB, 0xBF}, "您：你好"...),
		"big5":    []byte(big5),
	}
	for name, in := range cases {
		if got := Decode(in); got != "您：你好" {
			t.Errorf("%s: got %q", name, got)
		}
	}
}

func TestParseText(t *testing.T) {
	in := "preamble\r\n您：你好！\nAI：嗨\n第二行\n\n您：再見"
	got, err := ParseText(in, []string{"您：", "AI："})
	if err != nil {
		t.Fatal(err)
	}
	want := []Message{
		{"您", "你好！"},
		{"AI", "嗨\n第二行"},
		{"您", "再見"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}

	if _, err := ParseText("no roles", []string{"您："}); err != ErrNoMessage {
		t.Errorf("want ErrNoMessage, got %v", err)
	}
}

func TestParsePrefixes(t *testing.T) {
	got := ParsePrefixes(" 您：\n\nAI： \n")
	if !reflect.DeepEqual(got, []string{"您：", "AI："}) {
		t.Errorf("got %q", got)
	}
}

func TestFilters(t *testing.T) {
	cases := []struct {
		f       Filter
		in, out string
	}{
		{RemoveHTMLComments, "a<!-- x\ny -->b", "ab"},
		{RemoveDetails, "a<details>\n<summary>s</summary>x</details>b", "ab"},
		{RemoveHTMLTags, "<b>a</b><BR/>b</p>c", "a\nb\nc"},
		{func(s string) string { return LimitNewlines(s, 2) }, "a\n\n\n\nb", "a\n\nb"},
		{func(s string) string { return LimitNewlines(s, 0) }, "a\n\n\nb", "a\n\n\nb"},
	}
	for _, c := range cases {
		if got := c.f(c.in); got != c.out {
			t.Errorf("%q: got %q, want %q", c.in, got, c.out)
		}
	}
}

func TestApplyKeepsInput(t *testing.T) {
	in := []Message{{"AI", "<b>x</b>"}}
	out := Apply(in, RemoveHTMLTags)
	if out[0].Content != "x" || in[0].Content != "<b>x</b>" {
		t.Errorf("in %q out %q", in[0].Content, out[0].Content)
	}
}

func TestToTxt(t *testing.T) {
	msgs := []Message{{"您", "a\n\n\n\nb"}, {"AI", "c"}}
	got := ToTxt(msgs, TxtOptions{MaxNewlines: 2, SplitLines: true})
	want := "您：\na\n\nb\n\n---\n\nAI：\nc\n\n---"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestCountRoles(t *testing.T) {
	msgs := []Message{{"您", ""}, {"AI", ""}, {"您", ""}}
	want := []RoleCount{{"您", 2}, {"AI", 1}}
	if got := CountRoles(msgs); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestChapters(t *testing.T) {
	msgs := make([]Message, 0, 120)
	for i := range 120 {
		role := "AI"
		if i%3 == 0 {
			role = "您"
		}
		msgs = append(msgs, Message{role, strings.Repeat("字", 25)})
	}

	if n := len(Chapters(msgs, EpubOptions{Mode: ChapterBatch})); n != 3 {
		t.Errorf("batch: %d", n)
	}
	per := Chapters(msgs, EpubOptions{Mode: ChapterPerMessage})
	if len(per) != 120 || per[0].Title != "對話 1: "+strings.Repeat("字", 20)+"..." {
		t.Errorf("per message: %d %q", len(per), per[0].Title)
	}
	if n := len(Chapters(msgs, EpubOptions{Mode: ChapterUserStart, UserPrefix: "您："})); n != 40 {
		t.Errorf("user start: %d", n)
	}
}

func TestToEpub(t *testing.T) {
	msgs := []Message{{"您", "<hi> & bye"}, {"AI", "a\nb"}}
	b, err := ToEpub(msgs, EpubOptions{
		Title: "T&T", Author: "me", Now: time.Unix(0, 0),
	})
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if f := zr.File[0]; f.Name != "mimetype" || f.Method != zip.Store || len(f.Extra) != 0 {
		t.Fatalf("first entry %q method %d extra %d", f.Name, f.Method, len(f.Extra))
	}

	var ch string
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()

		// Every non-css file must be well-formed XML.
		if strings.HasSuffix(f.Name, ".xhtml") || strings.HasSuffix(f.Name, ".opf") ||
			strings.HasSuffix(f.Name, ".ncx") || strings.HasSuffix(f.Name, ".xml") {
			d := xml.NewDecoder(bytes.NewReader(data))
			d.Strict = true
			for {
				if _, err := d.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("%s: %v", f.Name, err)
				}
			}
		}
		if f.Name == "EPUB/chapter_1.xhtml" {
			ch = string(data)
		}
	}

	for _, s := range []string{"&lt;hi&gt; &amp; bye", "a<br/>b", "user-message", "ai-message"} {
		if !strings.Contains(ch, s) {
			t.Errorf("chapter missing %q", s)
		}
	}
}
