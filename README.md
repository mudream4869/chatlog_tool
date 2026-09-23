# 對話整理器 💬

一個簡單的工具，用於整理和清理對話紀錄（特別適合 AI RPG 對話）。

以 [ToolGUI](https://github.com/voilelab/toolgui) 寫成，可編譯成 WebAssembly 在瀏覽器中執行，檔案不會離開你的電腦。

**線上使用：<https://mudream4869.github.io/chatlog_tool/>**

## 功能

- 📤 上傳 TXT 格式的對話紀錄（自動辨識 UTF-8 / Big5）
- 🎭 自訂角色前綴識別
- 🧹 清理 HTML 標籤和註解
- 📊 訊息數、角色分布、清理前後字數統計
- 📝 匯出為純文字或 EPUB 電子書
- 🔍 預覽原始和清理後的內容
- 🧪 內建範例對話，不用準備檔案就能試用

## 使用方式

需要 Go 1.27.1 以上（或設定 `GOTOOLCHAIN=auto` 讓 Go 自動下載）。

```bash
# 本機伺服器模式
go run ./cmd/chatlog
# 開啟 http://127.0.0.1:3000

# 瀏覽器模式（WebAssembly）
go tool toolgui-wasm serve ./cmd/chatlog

# 產生靜態網站到 dist/
go tool toolgui-wasm build -o dist ./cmd/chatlog
```

推送到 `main` 時，GitHub Actions 會自動建置並部署到 GitHub Pages（需在 repo 設定中將 Pages 來源設為 GitHub Actions）。

## 對話格式範例

```
您：你好！
AI：你好！有什麼我可以幫忙的嗎？
您：請介紹一下你自己。
AI：我是 AI 助手，很高興認識你！
```

## 清理選項

- **移除 HTML 註解**：移除 `<!-- ... -->` 標籤
- **移除 details 標籤**：移除 `<details>` 標籤及其內容
- **移除所有 HTML 標籤**：清除所有 HTML 格式

## 匯出格式

- **純文字 (TXT)**：可選擇訊息間分隔線、限制連續換行
- **EPUB**：可設定標題、作者，章節可依批次（每 50 則）、每則訊息或用戶訊息分割

## 專案結構

- `chatlog/`：解析、清理與匯出邏輯（與 UI 無關，有單元測試）
- `cmd/chatlog/`：ToolGUI 介面；`main_server.go` 為伺服器模式，`main_wasm.go` 為瀏覽器模式
