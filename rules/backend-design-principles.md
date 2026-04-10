# Backend Design Principles

- Keep session state, command execution, collection, and LLM analysis in separate packages.
- Do not execute arbitrary shell commands.
- Do not log secrets or OAuth tokens.
- Keep first-version Feishu operations read-only.
