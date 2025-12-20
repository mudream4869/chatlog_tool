import re

from message import Message


class Serializer:
    def __init__(self):
        pass

    def serialize_messages(self, messages: list[Message]) -> str:
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
