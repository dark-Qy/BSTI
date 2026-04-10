# Personality Agent Design

The first version supports both service-side Feishu app credentials and automatic app setup through `lark-cli config init --new`. Each user session receives an isolated local config directory and can only execute approved `lark-cli` read commands.

The analysis window defaults to the last 30 days. Collection covers chat messages, docs, calendar events, tasks, mail, and meeting notes when permissions allow. Domain failures are treated as partial failures and are included in the final report context.

Persona visual assets use canonical shorthand based on memorable common words such as `PRISM`, `SPARK`, and `HUMBLE`. Asset filenames under `photos/` follow the pattern `<SHORTHAND>.png`.

The analysis output is a strict 20-choice BSPI classification, not a free-form label. The LLM receives the raw authorized data plus a compact local catalog of the 20 official personas, and must return one Top1 shorthand plus structured analysis fields. The server validates that shorthand against the local catalog, enriches it with the official label, dimensions, image URL, one-liner, and canonical description, and then renders the result page.
