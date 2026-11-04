# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Llama of Fate, please report it responsibly by emailing **c.ross.oss+llamaoffate@outlook.com**. Do not open a public GitHub issue for security vulnerabilities.

You should expect an initial response within 7 days acknowledging your report. We will work with you to understand and address the issue before any public disclosure.

## Scope

### In scope

- Server-side vulnerabilities (e.g., path traversal, command injection)
- Exposure of secrets or credentials
- Dependency vulnerabilities

### Out of scope: LLM prompt injection

LLM prompt injection — where a player crafts input to manipulate the game's AI responses — is a **known limitation of all LLM-based applications**, not a traditional security vulnerability. We track this as an ongoing research area:

- [#84](https://github.com/C-Ross/LlamaOfFate/issues/84) — LLM prompt escape testing
- [#83](https://github.com/C-Ross/LlamaOfFate/issues/83) — Abuse testing

We welcome contributions and research in this area, but please report these findings through normal GitHub issues rather than the security email.

## Deployment Considerations

Llama of Fate is designed for **local play**. The WebSocket endpoint (`/ws`) has no authentication. If you expose the server to the public internet, be aware that anyone can connect and start a game session. We recommend running behind a reverse proxy with authentication for any non-local deployment.
