import re

from message import Message


class Filter:
    def __init__(self):
        pass

    def filter_messages(self, messages: list[Message]) -> list[Message]:
        raise NotImplementedError('Subclasses should implement this method.')


class HtmlCommentFilter(Filter):
    def __init__(self):
        super().__init__()

    def filter_messages(self, messages: list[Message]) -> list[Message]:
        filtered_messages: list[Message] = []
        for msg in messages:
            content = re.sub(r'<!--.*?-->', '',
                             msg['content'], flags=re.DOTALL)
            filtered_messages.append({'role': msg['role'], 'content': content})

        return filtered_messages


class HtmlTagFilter(Filter):
    def __init__(self):
        super().__init__()

    def filter_messages(self, messages: list[Message]) -> list[Message]:
        filtered_messages: list[Message] = []
        for msg in messages:
            content = re.sub(r'<br\s*/?>', '\n', msg['content'], flags=re.IGNORECASE)
            content = re.sub(r'</p>', '\n', content, flags=re.IGNORECASE)
            content = re.sub(r'<[^>]+>', '', content)
            filtered_messages.append({'role': msg['role'], 'content': content})

        return filtered_messages


class HtmlDetailsFilter(Filter):
    def __init__(self):
        super().__init__()

    def filter_messages(self, messages: list[Message]) -> list[Message]:
        filtered_messages: list[Message] = []
        for msg in messages:
            content = re.sub(r'<details>.*?</details>', '',
                             msg['content'], flags=re.DOTALL)
            filtered_messages.append({'role': msg['role'], 'content': content})

        return filtered_messages


class MaxNewlineFilter(Filter):
    def __init__(self, max_newlines: int = 2):
        super().__init__()
        self.max_newlines = max_newlines

    def filter_messages(self, messages: list[Message]) -> list[Message]:
        filtered_messages: list[Message] = []
        pattern = re.compile(r'\n{' + str(self.max_newlines + 1) + r',}')
        for msg in messages:
            content = pattern.sub('\n' * self.max_newlines, msg['content'])
            filtered_messages.append({'role': msg['role'], 'content': content})

        return filtered_messages
