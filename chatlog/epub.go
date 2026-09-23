package chatlog

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"strings"
	"time"
)

// ChapterMode decides how messages are split into chapters.
type ChapterMode int

const (
	ChapterBatch      ChapterMode = iota // every BatchSize messages
	ChapterPerMessage                    // one message per chapter
	ChapterUserStart                     // a user message starts a chapter
)

// BatchSize is the chapter size of ChapterBatch.
const BatchSize = 50

// EpubOptions controls ToEpub.
type EpubOptions struct {
	Title       string
	Author      string
	MaxNewlines int // <= 0 means unlimited
	Mode        ChapterMode
	UserPrefix  string // for ChapterUserStart and styling
	Now         time.Time
}

// Chapter is one chapter of the book.
type Chapter struct {
	Title string
	File  string
	Msgs  []Message
}

// Chapters splits msgs per opt.Mode.
func Chapters(msgs []Message, opt EpubOptions) []Chapter {
	var chs []Chapter
	add := func(ms []Message) {
		n := len(chs) + 1
		chs = append(chs, Chapter{
			Title: fmt.Sprintf("第 %d 章", n),
			File:  fmt.Sprintf("chapter_%d.xhtml", n),
			Msgs:  ms,
		})
	}

	switch opt.Mode {
	case ChapterPerMessage:
		for i, m := range msgs {
			preview := []rune(m.Content)
			more := len(preview) > 20
			if more {
				preview = preview[:20]
			}
			title := fmt.Sprintf("對話 %d: %s", i+1,
				strings.ReplaceAll(string(preview), "\n", " "))
			if more {
				title += "..."
			}
			chs = append(chs, Chapter{
				Title: title,
				File:  fmt.Sprintf("message_%d.xhtml", i+1),
				Msgs:  []Message{m},
			})
		}
	case ChapterUserStart:
		var cur []Message
		for _, m := range msgs {
			if IsUserRole(m.Role, opt.UserPrefix) && len(cur) > 0 {
				add(cur)
				cur = nil
			}
			cur = append(cur, m)
		}
		if len(cur) > 0 {
			add(cur)
		}
	default:
		for i := 0; i < len(msgs); i += BatchSize {
			add(msgs[i:min(i+BatchSize, len(msgs))])
		}
	}
	return chs
}

// ToEpub packs msgs into an EPUB 3 book.
func ToEpub(msgs []Message, opt EpubOptions) ([]byte, error) {
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	chs := Chapters(msgs, opt)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// mimetype must be first, uncompressed, and without extra fields
	// (setting Modified would add one).
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte("application/epub+zip")); err != nil {
		return nil, err
	}

	files := []struct{ name, body string }{
		{"META-INF/container.xml", containerXML},
		{"EPUB/content.opf", contentOPF(chs, opt)},
		{"EPUB/nav.xhtml", navXHTML(chs, opt)},
		{"EPUB/toc.ncx", tocNCX(chs, opt)},
		{"EPUB/style/nav.css", styleCSS},
		{"EPUB/cover.xhtml", coverXHTML(len(msgs), opt)},
	}
	for _, ch := range chs {
		files = append(files, struct{ name, body string }{
			"EPUB/" + ch.File, chapterXHTML(ch, opt),
		})
	}

	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name: f.name, Method: zip.Deflate, Modified: opt.Now,
		})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var esc = html.EscapeString

const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`

func bookID(opt EpubOptions) string {
	return fmt.Sprintf("chatlog-%d", opt.Now.Unix())
}

func contentOPF(chs []Chapter, opt EpubOptions) string {
	var items, spine strings.Builder
	for i, ch := range chs {
		fmt.Fprintf(&items, "    <item id=\"ch%d\" href=\"%s\" media-type=\"application/xhtml+xml\"/>\n", i+1, ch.File)
		fmt.Fprintf(&spine, "    <itemref idref=\"ch%d\"/>\n", i+1)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id" xml:lang="zh-TW">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="id">%s</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:language>zh-TW</dc:language>
    <dc:creator>%s</dc:creator>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="css" href="style/nav.css" media-type="text/css"/>
    <item id="cover" href="cover.xhtml" media-type="application/xhtml+xml"/>
%s  </manifest>
  <spine toc="ncx">
    <itemref idref="cover"/>
%s  </spine>
</package>
`, bookID(opt), esc(opt.Title), esc(opt.Author),
		opt.Now.UTC().Format("2006-01-02T15:04:05Z"),
		items.String(), spine.String())
}

func navXHTML(chs []Chapter, opt EpubOptions) string {
	var li strings.Builder
	for _, ch := range chs {
		fmt.Fprintf(&li, "          <li><a href=\"%s\">%s</a></li>\n", ch.File, esc(ch.Title))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="zh-TW" xml:lang="zh-TW">
<head><title>%s</title></head>
<body>
  <nav epub:type="toc" id="toc">
    <h1>%s</h1>
    <ol>
      <li><a href="cover.xhtml">封面</a></li>
      <li><span>對話內容</span>
        <ol>
%s        </ol>
      </li>
    </ol>
  </nav>
</body>
</html>
`, esc(opt.Title), esc(opt.Title), li.String())
}

func tocNCX(chs []Chapter, opt EpubOptions) string {
	var pts strings.Builder
	for i, ch := range chs {
		fmt.Fprintf(&pts, `    <navPoint id="ch%d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="%s"/></navPoint>
`, i+1, i+2, esc(ch.Title), ch.File)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head><meta name="dtb:uid" content="%s"/></head>
  <docTitle><text>%s</text></docTitle>
  <navMap>
    <navPoint id="cover" playOrder="1"><navLabel><text>封面</text></navLabel><content src="cover.xhtml"/></navPoint>
%s  </navMap>
</ncx>
`, bookID(opt), esc(opt.Title), pts.String())
}

func xhtmlPage(title, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" lang="zh-TW" xml:lang="zh-TW">
<head>
  <title>%s</title>
  <link rel="stylesheet" type="text/css" href="style/nav.css"/>
</head>
<body>
%s
</body>
</html>
`, esc(title), body)
}

func coverXHTML(n int, opt EpubOptions) string {
	return xhtmlPage(opt.Title, fmt.Sprintf(`  <div style="text-align: center; margin-top: 100px;">
    <h1>%s</h1>
    <p>作者：%s</p>
    <p>生成時間：%s</p>
    <p>總共 %d 條對話</p>
  </div>`, esc(opt.Title), esc(opt.Author),
		opt.Now.Format("2006年01月02日 15:04"), n))
}

func chapterXHTML(ch Chapter, opt EpubOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  <h2>%s</h2>\n", esc(ch.Title))
	for _, m := range ch.Msgs {
		content := esc(LimitNewlines(m.Content, opt.MaxNewlines))
		content = strings.ReplaceAll(content, "\n", "<br/>")

		class := "other-message"
		switch {
		case IsUserRole(m.Role, opt.UserPrefix):
			class = "user-message"
		case strings.Contains(m.Role, "AI") ||
			strings.Contains(m.Role, "Assistant") ||
			strings.Contains(m.Role, "助手"):
			class = "ai-message"
		}
		fmt.Fprintf(&b, `  <div class="message-container %s">
    <div class="role">%s</div>
    <div class="content">%s</div>
  </div>
`, class, esc(m.Role), content)
	}
	return xhtmlPage(ch.Title, b.String())
}

const styleCSS = `.message-container {
    margin: 20px 0;
    padding: 15px;
    border-left: 4px solid #ddd;
    background-color: #f9f9f9;
}
.role {
    font-weight: bold;
    color: #333;
    margin-bottom: 10px;
}
.content {
    line-height: 1.6;
}
.user-message {
    border-left-color: #4CAF50;
}
.ai-message {
    border-left-color: #2196F3;
}
.other-message {
    border-left-color: #FF9800;
}
`
