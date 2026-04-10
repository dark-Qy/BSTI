# Personality Agent Design

The first version supports both service-side Feishu app credentials and automatic app setup through `lark-cli config init --new`. Each user session receives an isolated local config directory and can only execute approved `lark-cli` read commands.

The analysis window defaults to the last 30 days. Collection covers chat messages, docs, calendar events, tasks, mail, and meeting notes when permissions allow. Domain failures are treated as partial failures and are included in the final report context.
