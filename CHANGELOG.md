# Changelog

All notable changes to this project will be documented in this file.

## v0.8.0 (2026-06-18)

### Features

- Render replies as Bot API 10.1 rich messages (#19)
- Run scheduled tasks across threads in parallel (#17)
- Include day of week in prompt timestamp
- Send slash commands to Claude Code without metadata

### Bug Fixes

- Require double tildes for strikethrough formatting (#18)

## v0.7.0 (2026-04-14)

### Features

- Add configurable effort level with `/effort` command (#15)
- Format MCP tool names in status updates (#14)
- Add per-thread system prompts (#16)

### Improvements

- Add system prompts section

## v0.6.0 (2026-04-09)

### Features

- Add Markdown to Telegram HTML converter (#13)
- Show skill name in status updates (#12)

## v0.5.0 (2026-04-04)

### Features

- Inject current time per message with configurable timezone (#11)

### Improvements

- Add md language identifier to sub-agent prompt code blocks

## v0.4.0 (2026-03-26)

### Features

- Add typing indicators and status tracking to scheduled tasks (#10)

### Bug Fixes

- Remove hardcoded user-specific paths from skills for portability

### Improvements

- Delegate /remember and /voice skills to sub-agents

## v0.3.0 (2026-03-23)

### Features

- Add /clear command to reset session context (#9)
- Add /usage command to show context and cost tracking (#8)

## v0.2.0 (2026-03-22)

### Features

- Intermediate text status, three-tier log levels, and status truncation (#7)
- Add isolated session for scheduled tasks (#5)

### Bug Fixes

- Include CLI error details in Telegram error messages (#6)
- Send restart confirmation to the invoking thread
- Update /release issues

### Improvements

- Update /logs description for three-tier status levels
- Add table formatting rule to CLAUDE.md

## v0.1.0 (2026-03-16)

Initial release.
