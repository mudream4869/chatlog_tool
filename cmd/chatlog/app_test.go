package main

import (
	"testing"

	"github.com/voilelab/toolgui/toolgui/tgtest"
)

func TestMainPageEmpty(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if !p.HasText("請在左側上傳") {
		t.Error("no upload hint")
	}
}

func TestMainPageSample(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("沒有檔案？使用範例對話").Input(true)
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if !p.HasText("成功載入「範例對話」") {
		t.Error("sample not loaded")
	}
	if len(p.FindByName("chat_message_component")) == 0 {
		t.Error("no chat messages in preview")
	}
}

func TestMainPageUpload(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("上傳對話紀錄檔案").Upload("log.txt", []byte("您：你好\nAI：嗨\n"))
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if !p.HasText("共 2 筆訊息") {
		t.Error("upload not parsed")
	}
}

func TestMainPageBadPrefix(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("上傳對話紀錄檔案").Upload("log.txt", []byte("hello\n"))
	if !p.HasText("無法辨識對話格式") {
		t.Error("no parse error shown")
	}
}

func TestMainPageSillyTavern(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("上傳對話紀錄檔案").Upload("chat.jsonl", []byte(
		`{"chat_metadata":{}}`+"\n"+
			`{"name":"Seraphina","is_user":false,"mes":"Hi"}`+"\n"+
			`{"name":"帕秋莉","is_user":true,"mes":"hello?"}`+"\n"))
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if !p.HasText("SillyTavern 聊天紀錄（.jsonl）") || !p.HasText("共 2 筆訊息") {
		t.Error("SillyTavern jsonl not parsed")
	}
}

func TestMainPageBadJSONL(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("上傳對話紀錄檔案").Upload("data.jsonl", []byte(`{"messages":[]}`+"\n"))
	if !p.HasText("不是 SillyTavern 聊天紀錄") {
		t.Error("no SillyTavern format error shown")
	}
}

func TestMainPageToast(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	p.GetByLabel("沒有檔案？使用範例對話").Input(true)
	if len(p.FindByName("toast_component")) != 1 {
		t.Fatal("no load toast")
	}

	// Changing a setting must not re-toast the same file.
	p.GetByLabel("移除所有 HTML 標籤").Input(true)
	if len(p.FindByName("toast_component")) != 0 {
		t.Error("load toast fired again on rerun")
	}

	btns := p.Find(func(n *tgtest.Node) bool {
		return n.String("text") == "📥 下載整理後的 txt 檔案"
	})
	if len(btns) != 1 {
		t.Fatalf("got %d txt download buttons", len(btns))
	}
	btns[0].Click()
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	if !p.HasText("已下載 TXT 檔案") {
		t.Error("no download toast")
	}
}
