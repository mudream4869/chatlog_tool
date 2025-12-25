import time

import unifier
import filter
import serializer

from message import Message

import streamlit as st


def auto_decode(content: bytes) -> str:
    try:
        return content.decode('utf-8-sig', errors='replace')
    except UnicodeDecodeError:
        pass

    try:
        return content.decode('utf-8', errors='replace')
    except UnicodeDecodeError:
        pass

    try:
        return content.decode('big5', errors='replace')
    except UnicodeDecodeError:
        pass

    return content.decode('latin-1', errors='replace')


def try_unifiers(role_prefixes: list[str], content: str) -> list[Message]:
    unifiers: list[unifier.MessageUnifier] = [
        unifier.TextUnifier(role_prefixes=role_prefixes),
    ]

    last_exception = None

    for u in unifiers:
        try:
            messages = u.unify_messages_from_content(content)
            if messages:
                return messages
        except Exception as e:
            last_exception = e
            continue

    raise ValueError(f'無法辨識的對話紀錄格式。最後錯誤: {last_exception}')


def main():
    st.set_page_config(page_title='對話整理器', page_icon='💬')

    '''
    # 歡迎使用對話整理器！

    這個應用程式可以幫助你整理對話(尤其是 AI RPG 對話)成為易於閱讀和分享的格式。
    '''

    chatlog_file = st.sidebar.file_uploader('上傳對話紀錄檔案', type=['txt'])

    st.sidebar.markdown('---')

    st.sidebar.markdown('## 角色設定')

    role_prefixes_input = st.sidebar.text_area(
        '請輸入角色前綴，每行一個（例如 "您：" 和 "AI："）',
        value='您：\nAI：',
        help='用於辨識對話中不同角色的前綴字串。請確保每個前綴後面有冒號（:）'
    )
    role_prefixes = [line.strip()
                     for line in role_prefixes_input.splitlines() if line.strip()]

    st.sidebar.markdown('---')

    st.sidebar.markdown('### 清理選項')

    clear_html_comments = st.sidebar.checkbox(
        '移除 HTML 註解', value=True,
        help='移除對話內容中的 HTML 註解標籤(`<!-- ... -->`)')

    clear_html_details = st.sidebar.checkbox(
        '移除 <details> 標籤及其內容', value=True,
        help='移除對話內容中的 <details> 標籤及其內部內容')

    clear_html_tags = st.sidebar.checkbox(
        '移除所有 HTML 標籤', value=False,
        help='移除對話內容中的所有 HTML 標籤（例如 `<b>`, `<i>` 等）')

    if chatlog_file is None:
        st.warning('請上傳一個對話紀錄檔案以開始整理。')
        return

    content = auto_decode(chatlog_file.read())
    messages = try_unifiers(role_prefixes, content)
    st.text(f'成功載入對話，共 {len(messages)} 筆訊息。')

    tab_original_file_preview, \
        tab_after_cleanup_preview, \
        tab_export_txt, \
        tab_export_epub = st.tabs([
            '檔案預覽', '清理後預覽', '匯出格式 (txt)', '匯出格式 (epub)'
        ])

    def show_message_preview(msgs, k=10):
        for msg in msgs[:k]:
            with st.container():
                st.text(msg['role'] + ':')
                st.text(msg['content'])

            st.markdown('---')

    with tab_original_file_preview:
        st.text('前 10 筆對話預覽')
        show_message_preview(messages, 10)

    filters: list[filter.Filter] = []
    if clear_html_comments:
        filters.append(filter.HtmlCommentFilter())
    if clear_html_details:
        filters.append(filter.HtmlDetailsFilter())
    if clear_html_tags:
        filters.append(filter.HtmlTagFilter())

    msgs = messages
    for f in filters:
        msgs = f.filter_messages(msgs)

    with tab_after_cleanup_preview:
        st.text('清理後前 10 筆對話預覽')
        show_message_preview(msgs, 10)

    with tab_export_txt:
        add_split_lines = st.checkbox(
            '在訊息間加入分隔線', value=True,
            help='在每則訊息之間加入分隔線以增加可讀性')

        max_2_newlines = st.checkbox(
            '限制連續換行數量至兩行', value=True,
            help='將連續換行數量限制為最多兩行，以避免過多空白')

        max_newlines = 2 if max_2_newlines else 0
        file_serializer = serializer.TxtSerializer(
            max_newlines=max_newlines,
            add_split_lines=add_split_lines)
        file_extension = 'txt'
        mime_type = 'text/plain'

        output_content = file_serializer.serialize_messages(msgs)

        with st.expander('前 100 行輸出預覽'):
            st.text('\n'.join(output_content.splitlines()[:100]))

        timestamp = int(time.time())
        filename = f'dialogue_{timestamp}.{file_extension}'

        st.download_button(
            label=f'下載整理後的 txt 檔案',
            data=output_content,
            file_name=filename,
            mime=mime_type
        )

    with tab_export_epub:
        st.markdown('### EPUB 電子書設定')

        epub_title = st.text_input(
            '電子書標題',
            value='對話記錄',
            help='將顯示在電子書的封面和元數據中'
        )

        epub_author = st.text_input(
            '作者名稱',
            value='Chatlog Tool',
            help='將顯示在電子書的作者信息中'
        )

        epub_max_newlines = st.checkbox(
            '限制連續換行數量至兩行',
            value=True,
            key='epub_max_newlines',
            help='將連續換行數量限制為最多兩行，以避免過多空白'
        )

        st.markdown('---')

        max_newlines = 2 if epub_max_newlines else 0
        epub_serializer = serializer.EpubSerializer(
            title=epub_title,
            author=epub_author,
            max_newlines=max_newlines
        )

        try:
            epub_content = epub_serializer.serialize_messages(msgs)

            st.success(f'✅ EPUB 電子書生成成功！')
            st.info(f'📚 包含 {len(msgs)} 條對話，分為 {(len(msgs) + 49) // 50} 章')

            timestamp = int(time.time())
            epub_filename = f'dialogue_{timestamp}.epub'

            st.download_button(
                label='📥 下載 EPUB 電子書',
                data=epub_content,
                file_name=epub_filename,
                mime='application/epub+zip'
            )

        except Exception as e:
            st.error(f'❌ EPUB 生成失敗：{str(e)}')
            st.text('請檢查是否已正確安裝相關依賴套件。')


main()
