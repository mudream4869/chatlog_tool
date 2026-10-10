package chatlog

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
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
		{Role: "您", Content: "你好！"},
		{Role: "AI", Content: "嗨\n第二行"},
		{Role: "您", Content: "再見"},
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
	in := []Message{{Role: "AI", Content: "<b>x</b>"}}
	out := Apply(in, RemoveHTMLTags)
	if out[0].Content != "x" || in[0].Content != "<b>x</b>" {
		t.Errorf("in %q out %q", in[0].Content, out[0].Content)
	}
}

func TestToTxt(t *testing.T) {
	msgs := []Message{{Role: "您", Content: "a\n\n\n\nb"}, {Role: "AI", Content: "c"}}
	got := ToTxt(msgs, TxtOptions{MaxNewlines: 2, SplitLines: true})
	want := "您：\na\n\nb\n\n---\n\nAI：\nc\n\n---"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestCountRoles(t *testing.T) {
	msgs := []Message{{Role: "您", Content: ""}, {Role: "AI", Content: ""}, {Role: "您", Content: ""}}
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
		msgs = append(msgs, Message{Role: role, Content: strings.Repeat("字", 25)})
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
	msgs := []Message{{Role: "您", Content: "<hi> & bye"}, {Role: "AI", Content: "a\nb"}}
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

const stSample = `{"chat_metadata":{"integrity":"x"},"user_name":"unused","character_name":"unused"}
{"name":"Seraphina","is_user":false,"is_system":false,"mes":"*Hi.*","swipes":["*Hi.*"],"swipe_id":0}
{"name":"帕秋莉","is_user":true,"is_system":false,"mes":" hello? ","extra":{}}
{"name":"Seraphina","is_user":false,"mes":""}
`

func TestParseSillyTavern(t *testing.T) {
	got, err := ParseSillyTavern(strings.ReplaceAll(stSample, "\n", "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Message{
		{Role: "Seraphina", Content: "*Hi.*"},
		{Role: "帕秋莉", Content: "hello?", IsUser: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}

	for _, in := range []string{"您：你好", `{"chat_metadata":{}}`, `{"name":"a"}`} {
		if _, err := ParseSillyTavern(in); !errors.Is(err, ErrNotSillyTavern) {
			t.Errorf("%q: want ErrNotSillyTavern, got %v", in, err)
		}
	}
}

func TestIsSillyTavernFile(t *testing.T) {
	for name, want := range map[string]bool{"a.jsonl": true, "A.JSONL": true, "a.txt": false, "jsonl": false} {
		if got := IsSillyTavernFile(name); got != want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

func TestChaptersIsUser(t *testing.T) {
	msgs := []Message{{Role: "帕秋莉", IsUser: true}, {Role: "S"}, {Role: "帕秋莉", IsUser: true}}
	if n := len(Chapters(msgs, EpubOptions{Mode: ChapterUserStart})); n != 2 {
		t.Errorf("got %d chapters", n)
	}
}

func TestSuggestPrefixes(t *testing.T) {
	in := "說明：前言\r\n" +
		"玩家：你好\nGM：歡迎\n內容 12:30\nhttp://x.y\n" +
		"玩家：走吧\nGM：好\n他說：「嗯。」\n他說：「嗯。」\n" +
		"玩家：再見\nGM：再見\n 縮排：不算\n 縮排：不算\n"
	sugs := SuggestPrefixes(in)
	var got []string
	for _, s := range sugs {
		got = append(got, s.Prefix)
	}
	// "說明" appears once; "他說" lines are kept since the label itself is clean.
	if want := []string{"玩家：", "GM：", "他說："}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if sugs[0].Count != 3 || sugs[0].Samples[0] != "玩家：你好" {
		t.Errorf("got %+v", sugs[0])
	}
	if got := LikelyPrefixes(sugs); len(got) != 3 {
		t.Errorf("likely: got %q", got)
	}
	if got := SuggestPrefixes("12:30 起床\n12:31 刷牙\nnote http://a\nnote http://b"); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestLikelyPrefixes(t *testing.T) {
	sugs := []PrefixSuggestion{{Prefix: "您：", Count: 20}, {Prefix: "AI：", Count: 19}, {Prefix: "HP：", Count: 2}}
	if got := LikelyPrefixes(sugs); !reflect.DeepEqual(got, []string{"您：", "AI："}) {
		t.Errorf("got %q", got)
	}
}
