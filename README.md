# 對話整理器 💬

一個簡單的工具，用於整理和清理對話紀錄（特別適合 AI RPG 對話）。

## 功能

- 📤 上傳 TXT 格式的對話紀錄
- 🎭 自訂角色前綴識別
- 🧹 清理 HTML 標籤和註解
- 📝 匯出為純文字或 Markdown 格式
- 🔍 預覽原始和清理後的內容

## 安裝

```bash
# 建立虛擬環境
python -m venv venv
source venv/bin/activate  # macOS/Linux
# 或 venv\Scripts\activate  # Windows

# 安裝套件
pip install streamlit
```

## 使用方式

```bash
streamlit run main.py
```

然後在瀏覽器中打開顯示的網址（通常是 `http://localhost:8501`）。

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

- **純文字 (TXT)**：簡單的文字格式
