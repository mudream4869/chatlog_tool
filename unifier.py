from message import Message


class MessageUnifier:
    def __init__(self):
        pass

    def unify_messages_from_content(self, content: str) -> list[Message]:
        raise NotImplementedError('Subclasses should implement this method.')


class TextUnifier(MessageUnifier):
    GENERAL_ROLE_PREFIXES = [
        '你：', 'AI：', '玩家：', '系統：',
        'User:', 'Assistant:', 'Player:', 'System:', 'AI:'
    ]

    def __init__(self, role_prefixes: list[str] | None = None):
        super().__init__()

        if role_prefixes is None:
            role_prefixes = self.GENERAL_ROLE_PREFIXES

        self.role_prefixes = role_prefixes

    def unify_messages_from_content(self, content: str) -> list[Message]:
        messages: list[Message] = []
        lines = content.splitlines()
        current_role = None
        current_content = []

        for line in lines:
            for role_prefix in self.role_prefixes:
                if line.startswith(role_prefix):
                    if current_role:
                        messages.append(
                            {'role': current_role,
                             'content': '\n'.join(current_content).strip()})
                    current_role = role_prefix[:-1]  # Remove the colon
                    current_content = [line[len(role_prefix):].strip()]
                    break
            else:
                current_content.append(line)

        if current_role:
            messages.append(
                {'role': current_role, 'content': '\n'.join(current_content).strip()})

        return messages
