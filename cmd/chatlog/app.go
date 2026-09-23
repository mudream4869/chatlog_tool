package main

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mudream4869/chatlog_tool/chatlog"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

//go:embed sample.txt
var sampleLog []byte

//go:embed app.go
var appSource string

const intro = `這個工具可以幫你把對話紀錄（尤其是 AI RPG 對話）整理成易讀、易分享的格式。

所有處理都在瀏覽器裡完成，檔案不會上傳到任何伺服器。`

func newApp() *tgframe.App {
	app := tgframe.NewApp()
	app.SetTitle("對話整理器")
	// No page Emoji: toolgui v0.7.2's wasm index.html has no <link rel="icon">,
	// and setting one crashes the frontend.
	app.AddPage("index", "對話整理器", MainPage)
	app.AddPage("source", "原始碼", SourcePage)
	return app
}

func SourcePage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "原始碼")
	tgcomp.Markdown(p.Main, "這個頁面用 [ToolGUI](https://github.com/voilelab/toolgui) 寫成，"+
		"以下是 UI 的完整程式碼。")
	tgcomp.Code(p.Main, appSource)
	return nil
}

type settings struct {
	prefixes []string
	filters  []chatlog.Filter
}

func sidebar(c *tgframe.Container) (*tgcomp.FileObject, bool, settings) {
	file := tgcomp.FileUpload(c, "上傳對話紀錄檔案", ".txt,text/plain")
	useSample := tgcomp.Toggle(c, "沒有檔案？使用範例對話")
	tgcomp.Divider(c)

	tgcomp.Subtitle(c, "角色設定")
	prefixes := tgcomp.Textarea(c, "角色前綴（每行一個）",
		&tgcomp.TextareaConf{Default: "您：\nAI："})
	tgcomp.Caption(c, "用於辨識對話中不同角色的前綴字串，前綴的最後一個字元需為冒號。")
	tgcomp.Divider(c)

	tgcomp.Subtitle(c, "清理選項")
	var filters []chatlog.Filter
	if tgcomp.Checkbox(c, "移除 HTML 註解 <!-- ... -->",
		&tgcomp.CheckboxConf{Default: true}) {
		filters = append(filters, chatlog.RemoveHTMLComments)
	}
	if tgcomp.Checkbox(c, "移除 <details> 標籤及其內容",
		&tgcomp.CheckboxConf{Default: true}) {
		filters = append(filters, chatlog.RemoveDetails)
	}
	if tgcomp.Checkbox(c, "移除所有 HTML 標籤") {
		filters = append(filters, chatlog.RemoveHTMLTags)
	}

	return file, useSample, settings{
		prefixes: chatlog.ParsePrefixes(prefixes),
		filters:  filters,
	}
}

func MainPage(p *tgframe.Params) error {
	file, useSample, s := sidebar(p.Sidebar)

	tgcomp.Title(p.Main, "💬 對話整理器")
	tgcomp.Markdown(p.Main, intro)

	var (
		raw  []byte
		name string
	)
	switch {
	case file != nil:
		b, err := file.Bytes()
		if err != nil {
			return err
		}
		raw, name = b, file.Name
	case useSample:
		raw, name = sampleLog, "範例對話"
	default:
		tgcomp.MessageWarning(p.Main, "請在左側上傳一個對話紀錄檔案（.txt），或開啟「使用範例對話」。")
		return nil
	}

	msgs, err := chatlog.ParseText(chatlog.Decode(raw), s.prefixes)
	if err != nil {
		tgcomp.MessageDanger(p.Main, err.Error()+"，請確認角色前綴設定。",
			&tgcomp.MessageConf{Title: "無法辨識對話格式"})
		return nil
	}
	cleaned := chatlog.Apply(msgs, s.filters...)

	overview(p.Main, name, msgs, cleaned)

	tabs := tgcomp.Tab(p.Main, []string{
		"檔案預覽", "清理後預覽", "匯出 TXT", "匯出 EPUB",
	})
	preview(tabs[0], "raw", msgs)
	preview(tabs[1], "cleaned", cleaned)
	exportTxt(tabs[2], cleaned)
	exportEpub(tabs[3], cleaned)
	return nil
}

func overview(c *tgframe.Container, name string, msgs, cleaned []chatlog.Message) {
	tgcomp.MessageSuccess(c, fmt.Sprintf("成功載入「%s」，共 %d 筆訊息。", name, len(msgs)))

	roles := chatlog.CountRoles(msgs)
	before, after := chatlog.CountRunes(msgs), chatlog.CountRunes(cleaned)

	c1, c2, c3 := tgcomp.EqColumn3(c)
	tgcomp.Metric(c1, "訊息數", strconv.Itoa(len(msgs)))
	tgcomp.Metric(c2, "角色數", strconv.Itoa(len(roles)))
	delta := ""
	if after != before {
		delta = fmt.Sprintf("%+d", after-before)
	}
	tgcomp.Metric(c3, "清理後字數", strconv.Itoa(after),
		&tgcomp.MetricConf{Delta: delta, DeltaColorInverse: true})

	labels := make([]string, len(roles))
	values := make([]float64, len(roles))
	for i, r := range roles {
		labels[i], values[i] = r.Role, float64(r.Count)
	}
	exp := tgcomp.Expand(c, "各角色訊息數", false)
	tgcomp.BarChart(exp, labels, []tgcomp.ChartSeries{{Name: "訊息數", Values: values}},
		&tgcomp.ChartConf{Height: "240px"})
}

func preview(c *tgframe.Container, id string, msgs []chatlog.Message) {
	maxN := max(1, min(len(msgs), 50))
	n := tgcomp.Slider(c, "預覽筆數", &tgcomp.SliderConf[int]{
		Base:    tgcomp.Base{ID: "preview_" + id},
		Default: ptr(min(10, maxN)),
		Min:     ptr(1),
		Max:     ptr(maxN),
	})
	for _, m := range msgs[:min(n, len(msgs))] {
		box := tgcomp.Box(c)
		tgcomp.Badge(box, m.Role)
		multilineText(box, m.Content)
	}
}

// multilineText keeps line breaks: toolgui's Text collapses them.
func multilineText(c *tgframe.Container, s string) {
	for line := range strings.SplitSeq(s, "\n") {
		if line == "" {
			line = "\u00a0" // keep blank lines' height
		}
		tgcomp.Text(c, line)
	}
}

func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func exportTxt(c *tgframe.Container, msgs []chatlog.Message) {
	split := tgcomp.Checkbox(c, "在訊息間加入分隔線", &tgcomp.CheckboxConf{
		Base: tgcomp.Base{ID: "txt_split"}, Default: true,
	})
	limit := tgcomp.Checkbox(c, "限制連續換行數量至兩行", &tgcomp.CheckboxConf{
		Base: tgcomp.Base{ID: "txt_limit"}, Default: true,
	})

	opt := chatlog.TxtOptions{SplitLines: split}
	if limit {
		opt.MaxNewlines = 2
	}
	out := chatlog.ToTxt(msgs, opt)

	lines := strings.Split(out, "\n")
	exp := tgcomp.Expand(c, "前 100 行輸出預覽", false)
	tgcomp.Code(exp, strings.Join(lines[:min(100, len(lines))], "\n"),
		&tgcomp.CodeConf{Language: "text"})

	tgcomp.DownloadFile(c, "📥 下載整理後的 txt 檔案", []byte(out), &tgcomp.DownloadFileConf{
		Filename: "dialogue_" + timestamp() + ".txt",
		MIME:     "text/plain",
	})
}

var chapterModes = []string{
	fmt.Sprintf("批次分割（每 %d 則訊息一章）", chatlog.BatchSize),
	"每則訊息一章",
	"用戶訊息開始新章節",
}

func exportEpub(c *tgframe.Container, msgs []chatlog.Message) {
	tgcomp.Subtitle(c, "EPUB 電子書設定")

	c1, c2 := tgcomp.EqColumn2(c)
	opt := chatlog.EpubOptions{
		Title:      tgcomp.Textbox(c1, "電子書標題", &tgcomp.TextboxConf{Default: "對話記錄"}),
		Author:     tgcomp.Textbox(c2, "作者名稱", &tgcomp.TextboxConf{Default: "Chatlog Tool"}),
		UserPrefix: "您：",
	}
	if tgcomp.Checkbox(c, "限制連續換行數量至兩行", &tgcomp.CheckboxConf{
		Base: tgcomp.Base{ID: "epub_limit"}, Default: true,
	}) {
		opt.MaxNewlines = 2
	}

	// Select rather than Radio: toolgui's Radio doesn't show its label.
	mode := tgcomp.Select(c, "章節分割方式", chapterModes,
		(&tgcomp.SelectConf{}).SetDefault(0))
	if mode != nil {
		opt.Mode = chatlog.ChapterMode(*mode)
	}
	if opt.Mode == chatlog.ChapterUserStart {
		opt.UserPrefix = tgcomp.Textbox(c, "用戶角色前綴", &tgcomp.TextboxConf{Default: "您："})
		tgcomp.Caption(c, "遇到此前綴（或含「您」「User」「用戶」）的訊息時開始新章節。")
	}
	tgcomp.Divider(c)

	book, err := chatlog.ToEpub(msgs, opt)
	if err != nil {
		tgcomp.MessageDanger(c, err.Error(), &tgcomp.MessageConf{Title: "EPUB 生成失敗"})
		return
	}

	chapters := len(chatlog.Chapters(msgs, opt))
	tgcomp.MessageInfo(c, fmt.Sprintf("📚 包含 %d 則對話，分為 %d 章。", len(msgs), chapters))
	tgcomp.DownloadFile(c, "📥 下載 EPUB 電子書", book, &tgcomp.DownloadFileConf{
		Filename: "dialogue_" + timestamp() + ".epub",
		MIME:     "application/epub+zip",
	})
}

func ptr[T any](v T) *T { return &v }
