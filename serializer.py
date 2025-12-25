import re
import html
from io import BytesIO
from datetime import datetime

from ebooklib import epub

from message import Message


class Serializer:
    def __init__(self):
        pass

    def serialize_messages(self, messages: list[Message]) -> bytes | str:
        raise NotImplementedError('Subclasses should implement this method.')


class TxtSerializer(Serializer):
    def __init__(self, max_newlines: int = 2, add_split_lines: bool = False):
        super().__init__()
        self.max_newlines = max_newlines
        self.add_split_lines = add_split_lines

    def serialize_messages(self, messages: list[Message]) -> str:
        txt_output = ''
        for msg in messages:
            role = msg['role']
            content = msg['content']
            txt_output += f'{role}：\n{content}\n\n'
            if self.add_split_lines:
                txt_output += '---\n\n'

        if self.max_newlines > 0:
            pattern = re.compile(r'\n{' + str(self.max_newlines + 1) + r',}')
            txt_output = pattern.sub('\n' * self.max_newlines, txt_output)

        return txt_output.strip()


class EpubSerializer(Serializer):
    def __init__(self, title: str = "對話記錄", author: str = "Chatlog Tool", max_newlines: int = 2,
                 chapter_mode: str = "batch", user_role_prefix: str = "您："):
        super().__init__()
        self.title = title
        self.author = author
        self.max_newlines = max_newlines
        self.chapter_mode = chapter_mode  # "batch", "per_message", "user_start"
        self.user_role_prefix = user_role_prefix

    def serialize_messages(self, messages: list[Message]) -> bytes:
        ''' Serialize a list of messages into an in-memory EPUB book.
        The provided messages are converted into one or more XHTML chapters
        (according to the configured chapter mode) and packaged as an EPUB
        document. The resulting EPUB file contents are returned as a bytes
        object, suitable for writing directly to disk or sending as a binary
        payload.

        :param messages: Ordered list of Message objects representing the
            conversation to include in the EPUB.
        :return: The complete EPUB file content encoded as bytes.
        :raises Exception: Any exception raised by ``ebooklib.epub`` or the
            underlying packaging/writing process during EPUB generation is
            propagated to the caller.
        '''

        # Create EPUB book
        book = epub.EpubBook()

        # Set metadata
        book.set_identifier('chatlog-' + str(int(datetime.now().timestamp())))
        book.set_title(self.title)
        book.set_language('zh-TW')
        book.add_author(self.author)

        # Create cover page
        cover_content = f'''<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>{html.escape(self.title)}</title>
    <meta charset="UTF-8"/>
</head>
<body>
    <div style="text-align: center; margin-top: 100px;">
        <h1>{html.escape(self.title)}</h1>
        <p>作者：{html.escape(self.author)}</p>
        <p>生成時間：{datetime.now().strftime('%Y年%m月%d日 %H:%M')}</p>
        <p>總共 {len(messages)} 條對話</p>
    </div>
</body>
</html>'''

        cover = epub.EpubHtml(
            title='封面', file_name='cover.xhtml', lang='zh-TW')
        cover.content = cover_content
        book.add_item(cover)

        # Create CSS for styling
        nav_css = epub.EpubItem(
            uid="nav_css",
            file_name="style/nav.css",
            media_type="text/css",
            content='''
.message-container {
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
            '''
        )
        book.add_item(nav_css)

        # Generate chapters based on selected mode
        chapters = self._create_chapters(messages)

        # Add chapters to book
        for chapter in chapters:
            book.add_item(chapter)

        # Create table of contents
        book.toc = (
            epub.Link("cover.xhtml", "封面", "cover"),
            (
                epub.Section("對話內容"),
                tuple(chapters)
            )
        )

        # Add navigation files
        book.add_item(epub.EpubNcx())
        book.add_item(epub.EpubNav())

        # Create spine
        book.spine = ['cover'] + chapters

        # Generate EPUB content as bytes
        epub_buffer = BytesIO()
        epub.write_epub(epub_buffer, book, {})
        epub_buffer.seek(0)

        return epub_buffer.read()

    def _create_chapters(self, messages: list[Message]) -> list:
        if self.chapter_mode == "per_message":
            return self._create_chapters_per_message(messages)
        elif self.chapter_mode == "user_start":
            return self._create_chapters_user_start(messages)
        else:  # default "batch"
            return self._create_chapters_batch(messages)

    def _create_chapters_batch(self, messages: list[Message]) -> list:
        chapter_size = 50
        chapters = []

        for i in range(0, len(messages), chapter_size):
            chapter_messages = messages[i:i + chapter_size]
            chapter_num = (i // chapter_size) + 1

            chapter = self._create_single_chapter(
                chapter_messages,
                f'第 {chapter_num} 章',
                f'chapter_{chapter_num}.xhtml'
            )
            chapters.append(chapter)

        return chapters

    def _create_chapters_per_message(self, messages: list[Message]) -> list:
        chapters = []

        for i, msg in enumerate(messages):
            chapter_num = i + 1
            chapter_title = f'對話 {chapter_num}'

            content_preview = msg['content'][:20].replace('\n', ' ')
            if len(msg['content']) > 20:
                content_preview += "..."
            chapter_title += f': {content_preview}'

            chapter = self._create_single_chapter(
                [msg],
                chapter_title,
                f'message_{chapter_num}.xhtml'
            )
            chapters.append(chapter)

        return chapters

    def _create_chapters_user_start(self, messages: list[Message]) -> list:
        chapters = []
        current_chapter_messages = []
        chapter_num = 1

        for msg in messages:
            if (msg['role'].startswith(self.user_role_prefix) or
                msg['role'].startswith(self.user_role_prefix.rstrip('：')) or
                    '您' in msg['role'] or 'User' in msg['role'] or '用戶' in msg['role']):

                if current_chapter_messages:
                    chapter = self._create_single_chapter(
                        current_chapter_messages,
                        f'第 {chapter_num} 章',
                        f'chapter_{chapter_num}.xhtml'
                    )
                    chapters.append(chapter)
                    chapter_num += 1

                current_chapter_messages = [msg]
            else:
                current_chapter_messages.append(msg)

        if current_chapter_messages:
            chapter = self._create_single_chapter(
                current_chapter_messages,
                f'第 {chapter_num} 章',
                f'chapter_{chapter_num}.xhtml'
            )
            chapters.append(chapter)

        return chapters

    def _create_single_chapter(self, chapter_messages: list[Message], title: str, filename: str) -> epub.EpubHtml:
        chapter_content = f'''<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>{html.escape(title)}</title>
    <meta charset="UTF-8"/>
    <link rel="stylesheet" type="text/css" href="../style/nav.css"/>
</head>
<body>
    <h2>{html.escape(title)}</h2>'''

        newline_limit_pattern = re.compile(
            r'\n{' + str(self.max_newlines + 1) + r',}')

        for msg in chapter_messages:
            role = html.escape(msg['role'])
            content = html.escape(msg['content'])

            # Limit consecutive newlines if max_newlines is set
            if self.max_newlines > 0:
                content = newline_limit_pattern.sub(
                    '\n' * self.max_newlines, content)

            # Convert newlines to <br> tags
            content = content.replace('\n', '<br/>')

            # Determine message type for styling
            css_class = 'message-container'
            if '您' in role or 'User' in role or '用戶' in role:
                css_class += ' user-message'
            elif 'AI' in role or 'Assistant' in role or '助手' in role:
                css_class += ' ai-message'
            else:
                css_class += ' other-message'

            chapter_content += f'''
    <div class="{css_class}">
        <div class="role">{role}</div>
        <div class="content">{content}</div>
    </div>'''

        chapter_content += '''
</body>
</html>'''

        # Create chapter
        chapter = epub.EpubHtml(
            title=title,
            file_name=filename,
            lang='zh-TW'
        )
        chapter.content = chapter_content
        return chapter
