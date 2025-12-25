#!/usr/bin/env python3
"""
Test script to verify EPUB generation functionality
"""

from serializer import EpubSerializer
from message import Message
import zipfile
import tempfile
import os


def test_epub_generation():
    """Test EPUB generation with sample data"""

    # Create more comprehensive test messages
    messages = [
        {"role": "您", "content": "你好，我想了解一下人工智能的發展歷史。"},
        {"role": "AI", "content": "人工智能的發展可以追溯到1950年代。阿蘭·圖靈提出了著名的圖靈測試，這被認為是人工智能領域的起點之一。\n\n在1956年，達特茅斯會議正式確立了人工智能這一學科名稱。"},
        {"role": "您", "content": "那麼現在的AI技術有哪些主要應用呢？"},
        {"role": "AI", "content": "現在的AI技術應用非常廣泛，主要包括：\n\n1. **自然語言處理**：如ChatGPT、翻譯軟件\n2. **計算機視覺**：人臉識別、自動駕駛\n3. **推薦系統**：電商、視頻平台的個性化推薦\n4. **語音識別**：智能音箱、語音助手"},
        {"role": "您", "content": "AI會取代人類的工作嗎？"},
        {"role": "AI", "content": "這是一個複雜的問題。AI確實會改變就業市場：\n\n• 某些重複性工作可能被自動化\n• 但同時會創造新的工作機會\n• 關鍵是要適應變化，持續學習新技能\n\n重要的是AI應該作為工具來增強人類能力，而不是簡單的替代。"},
    ]

    # Generate EPUB
    epub_serializer = EpubSerializer(
        title="AI對話測試記錄",
        author="Chatlog Tool 測試"
    )

    try:
        epub_content = epub_serializer.serialize_messages(messages)
        print(f"✅ EPUB 生成成功！")
        print(f"📊 文件大小：{len(epub_content)} bytes")
        print(f"💬 對話數量：{len(messages)} 條")

        # Verify it's a valid ZIP file (EPUB is essentially a ZIP)
        with tempfile.NamedTemporaryFile(suffix='.epub', delete=False) as tmp_file:
            tmp_file.write(epub_content)
            tmp_file_path = tmp_file.name

        try:
            with zipfile.ZipFile(tmp_file_path, 'r') as zip_ref:
                file_list = zip_ref.namelist()
                print(f"📁 EPUB 內部文件：")
                for file_name in sorted(file_list):
                    print(f"   • {file_name}")

                # Check for required EPUB files
                required_files = [
                    'META-INF/container.xml', 'OEBPS/content.opf']
                missing_files = [
                    f for f in required_files if f not in file_list]

                if missing_files:
                    print(f"⚠️  缺少必要文件：{missing_files}")
                else:
                    print("✅ EPUB 結構檢查通過！")

        finally:
            # Clean up temp file
            os.unlink(tmp_file_path)

        return True

    except Exception as e:
        print(f"❌ EPUB 生成失敗：{e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    print("🧪 開始 EPUB 功能測試...")
    success = test_epub_generation()
    print(f"\n{'✅ 測試通過！' if success else '❌ 測試失敗！'}")
