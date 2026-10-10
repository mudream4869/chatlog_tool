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

const intro = `這個工具可以幫你把對話紀錄（尤其是 AI RPG 對話）整理成易讀、易分享的格式。

所有處理都在瀏覽器裡完成，檔案不會上傳到任何伺服器。`

func newApp() *tgframe.App {
	app := tgframe.NewApp()
	app.SetTitle("對話整理器")
	app.SetIcon("assets/favicon.ico")
	app.AddPageByConfig(&tgframe.PageConfig{
		Name: "index", Title: "對話整理器", Emoji: "💬",
	}, MainPage)
	return app
}

type settings struct {
	prefixes []string
	filters  []chatlog.Filter
}

// source is the loaded chat log.
type source struct {
	raw  []byte
	name string
	text string // decoded; empty for SillyTavern
	st   bool   // SillyTavern .jsonl
}

// key identifies the source across reruns.
func (s *source) key() string { return fmt.Sprintf("%s/%d", s.name, len(s.raw)) }

// loadSource reads the upload or the sample; nil if neither.
func loadSource(c *tgframe.Container) (*source, error) {
	file := tgcomp.FileUpload(c, "上傳對話紀錄檔案", ".txt,.jsonl,text/plain")
	tgcomp.Caption(c, "支援純文字對話（.txt）與 SillyTavern 聊天紀錄（.jsonl）。")
	useSample := tgcomp.Toggle(c, "沒有檔案？使用範例對話")
	tgcomp.Divider(c)

	var src *source
	switch {
	case file != nil:
		b, err := file.Bytes()
		if err != nil {
			return nil, err
		}
		src = &source{raw: b, name: file.Name, st: chatlog.IsSillyTavernFile(file.Name)}
	case useSample:
		src = &source{raw: sampleLog, name: "範例對話"}
	default:
		return nil, nil
	}
	if !src.st {
		src.text = chatlog.Decode(src.raw)
	}
	return src, nil
}

func sidebar(c *tgframe.Container, src *source) settings {
	tgcomp.Subtitle(c, "角色設定")
	var prefixes string
	if src != nil && src.st {
		tgcomp.Caption(c, "SillyTavern 聊天紀錄（.jsonl）會直接使用檔案中的角色名稱，不需設定角色前綴。")
	} else {
		def, resetKey := "您：\nAI：", ""
		var sugs []chatlog.PrefixSuggestion
		if src != nil {
			sugs = chatlog.SuggestPrefixes(src.text)
			if likely := chatlog.LikelyPrefixes(sugs); len(likely) > 0 {
				def = strings.Join(likely, "\n")
			}
			resetKey = src.key()
		}
		prefixes = tgcomp.Textarea(c, "角色前綴（每行一個）",
			&tgcomp.TextareaConf{Default: def, ResetKey: resetKey})
		tgcomp.Caption(c, "用於辨識對話中不同角色的前綴字串，前綴的最後一個字元需為冒號。")
		if src != nil {
			prefixSuggestions(c, sugs)
		}
	}
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

	return settings{
		prefixes: chatlog.ParsePrefixes(prefixes),
		filters:  filters,
	}
}

// prefixSuggestions lists the detected role prefixes.
func prefixSuggestions(c *tgframe.Container, sugs []chatlog.PrefixSuggestion) {
	if len(sugs) == 0 {
		tgcomp.Caption(c, "偵測不到角色前綴，使用預設值。")
		return
	}
	tgcomp.Caption(c, "🧪 已依檔案內容自動偵測角色前綴（實驗性功能），請確認是否正確。")
	exp := tgcomp.Expand(c, "偵測到的角色前綴", false)
	rows := make([][]tgcomp.Cell, len(sugs))
	for i, s := range sugs {
		rows[i] = []tgcomp.Cell{tgcomp.TextCell(s.Prefix), tgcomp.NumberCell(float64(s.Count)),
			tgcomp.TextCell(strings.Join(s.Samples, " / "))}
	}
	tgcomp.DataFrameCells(exp, []string{"前綴", "行數", "範例"}, rows, &tgcomp.DataFrameConf{
		Base:       tgframe.Base{ID: "prefix_suggestions"},
		ColumnConf: []tgcomp.DataFrameColumnConf{{}, {Type: tgcomp.ColumnTypeNumber}, {}},
	})
	tgcomp.Caption(exp, "列出行首出現兩次以上的「名稱：」。較少出現的前綴不會自動填入，可自行複製到上方使用。")
}

const rawPreviewLines = 100

// rawPreview shows the start of the text so users can find the prefixes.
func rawPreview(c *tgframe.Container, text string, open bool) {
	lines := strings.Split(text, "\n")
	exp := tgcomp.Expand(c, fmt.Sprintf("原始文字（前 %d 行）", rawPreviewLines), open)
	tgcomp.Code(exp, strings.Join(lines[:min(rawPreviewLines, len(lines))], "\n"),
		&tgcomp.CodeConf{Base: tgframe.Base{ID: "raw_preview"}, Language: "text"})
}

func MainPage(p *tgframe.Params) error {
	src, err := loadSource(p.Sidebar)
	if err != nil {
		return err
	}
	s := sidebar(p.Sidebar, src)

	tgcomp.Title(p.Main, "💬 對話整理器")
	tgcomp.Markdown(p.Main, intro)

	if src == nil {
		p.State.Delete(loadedKey)
		tgcomp.MessageWarning(p.Main, "請在左側上傳一個對話紀錄檔案（.txt 或 SillyTavern .jsonl），或開啟「使用範例對話」。")
		return nil
	}

	var msgs []chatlog.Message
	if src.st {
		msgs, err = chatlog.ParseSillyTavern(chatlog.Decode(src.raw))
		if err != nil {
			tgcomp.MessageDanger(p.Main, err.Error()+"。目前 .jsonl 只支援 SillyTavern 匯出的聊天紀錄。",
				&tgcomp.MessageConf{Title: "無法辨識對話格式"})
			return nil
		}
	} else {
		msgs, err = chatlog.ParseText(src.text, s.prefixes)
		if err != nil {
			tgcomp.MessageDanger(p.Main, err.Error()+"，請對照下方原始文字修改左側的角色前綴設定。",
				&tgcomp.MessageConf{Title: "無法辨識對話格式"})
			rawPreview(p.Main, src.text, true)
			return nil
		}
	}
	cleaned := chatlog.Apply(msgs, s.filters...)

	toastLoaded(p, src, len(msgs))
	overview(p.Main, src.name, msgs, cleaned)
	if !src.st {
		rawPreview(p.Main, src.text, false)
	}

	tabs := tgcomp.Tab(p.Main, []string{
		"檔案預覽", "清理後預覽", "匯出 TXT", "匯出 EPUB",
	})
	preview(tabs[0].Scope("raw"), msgs)
	preview(tabs[1].Scope("cleaned"), cleaned)
	exportTxt(tabs[2].Scope("txt"), cleaned)
	exportEpub(tabs[3].Scope("epub"), cleaned, src.st)
	return nil
}

const loadedKey = "chatlog.loaded"

// toastLoaded toasts only when the source changes, not on every rerun.
func toastLoaded(p *tgframe.Params, src *source, n int) {
	key := src.key()
	if last, _ := p.State.Get[string](loadedKey); last == key {
		return
	}
	p.State.Set(loadedKey, key)
	text := fmt.Sprintf("成功載入「%s」，共 %d 筆訊息。", src.name, n)
	if src.st {
		text = "已依 SillyTavern 聊天紀錄（.jsonl）格式" + text
	}
	tgcomp.Toast(p.Main, text, &tgcomp.ToastConf{Icon: "✅"})
}

func overview(c *tgframe.Container, name string, msgs, cleaned []chatlog.Message) {
	tgcomp.Caption(c, fmt.Sprintf("目前檔案：%s", name))

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

func preview(c *tgframe.Container, msgs []chatlog.Message) {
	maxN := max(1, min(len(msgs), 50))
	n := tgcomp.Slider(c, "預覽筆數", &tgcomp.SliderConf[int]{
		Default: new(min(10, maxN)),
		Min:     new(1),
		Max:     new(maxN),
	})
	for _, m := range msgs[:min(n, len(msgs))] {
		tgcomp.Text(tgcomp.ChatMessage(c, m.Role, &tgcomp.ChatMessageConf{Avatar: avatar(m)}), m.Content)
	}
}

// avatar gives user-side roles such as "您" a user icon; "" keeps toolgui's
// default (its own icons for user/assistant, else the first letter).
func avatar(m chatlog.Message) string {
	switch strings.ToLower(m.Role) {
	case "user", "human":
		return ""
	}
	if m.IsUser || chatlog.IsUserRole(m.Role, "") {
		return "🧑"
	}
	return ""
}

func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func exportTxt(c *tgframe.Container, msgs []chatlog.Message) {
	split := tgcomp.Checkbox(c, "在訊息間加入分隔線", &tgcomp.CheckboxConf{Default: true})
	limit := tgcomp.Checkbox(c, "限制連續換行數量至兩行", &tgcomp.CheckboxConf{Default: true})

	opt := chatlog.TxtOptions{SplitLines: split}
	if limit {
		opt.MaxNewlines = 2
	}
	out := chatlog.ToTxt(msgs, opt)

	lines := strings.Split(out, "\n")
	exp := tgcomp.Expand(c, "前 100 行輸出預覽", false)
	tgcomp.Code(exp, strings.Join(lines[:min(100, len(lines))], "\n"),
		&tgcomp.CodeConf{Language: "text"})

	if tgcomp.DownloadFile(c, "📥 下載整理後的 txt 檔案", []byte(out), &tgcomp.DownloadFileConf{
		Filename: "dialogue_" + timestamp() + ".txt",
		MIME:     "text/plain",
	}) {
		tgcomp.Toast(c, "已下載 TXT 檔案", &tgcomp.ToastConf{Icon: "📄"})
	}
}

var chapterModes = []string{
	fmt.Sprintf("批次分割（每 %d 則訊息一章）", chatlog.BatchSize),
	"每則訊息一章",
	"用戶訊息開始新章節",
}

func exportEpub(c *tgframe.Container, msgs []chatlog.Message, st bool) {
	tgcomp.Subtitle(c, "EPUB 電子書設定")

	c1, c2 := tgcomp.EqColumn2(c)
	opt := chatlog.EpubOptions{
		Title:      tgcomp.Textbox(c1, "電子書標題", &tgcomp.TextboxConf{Default: "對話記錄"}),
		Author:     tgcomp.Textbox(c2, "作者名稱", &tgcomp.TextboxConf{Default: "Chatlog Tool"}),
		UserPrefix: "您：",
	}
	if tgcomp.Checkbox(c, "限制連續換行數量至兩行", &tgcomp.CheckboxConf{Default: true}) {
		opt.MaxNewlines = 2
	}

	mode := tgcomp.Radio(c, "章節分割方式", chapterModes,
		(&tgcomp.RadioConf{}).SetDefault(0))
	if mode != nil {
		opt.Mode = chatlog.ChapterMode(*mode)
	}
	if st {
		// Users are marked by is_user, not by prefix.
		opt.UserPrefix = ""
	}
	if opt.Mode == chatlog.ChapterUserStart {
		if st {
			tgcomp.Caption(c, "依 SillyTavern 紀錄中標記為用戶（is_user）的訊息開始新章節。")
		} else {
			opt.UserPrefix = tgcomp.Textbox(c, "用戶角色前綴", &tgcomp.TextboxConf{Default: "您："})
			tgcomp.Caption(c, "遇到此前綴（或含「您」「User」「用戶」）的訊息時開始新章節。")
		}
	}
	tgcomp.Divider(c)

	chapters := len(chatlog.Chapters(msgs, opt))
	tgcomp.MessageInfo(c, fmt.Sprintf("📚 包含 %d 則對話，分為 %d 章。", len(msgs), chapters))
	// Build the EPUB only on click, not on every rerun.
	var genErr error
	gen := func() ([]byte, error) {
		b, err := chatlog.ToEpub(msgs, opt)
		genErr = err
		return b, err
	}
	// gen's error is already shown under the button.
	if tgcomp.DownloadFileFunc(c, "📥 下載 EPUB 電子書", gen, &tgcomp.DownloadFileConf{
		Filename: "dialogue_" + timestamp() + ".epub",
		MIME:     "application/epub+zip",
	}) && genErr == nil {
		tgcomp.Toast(c, fmt.Sprintf("已產生 EPUB 電子書（%d 章）", chapters), &tgcomp.ToastConf{Icon: "📚"})
	}
}
