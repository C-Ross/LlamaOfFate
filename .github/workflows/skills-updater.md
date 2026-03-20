---
description: Detects significant repository changes on push to main and creates a PR to update existing Copilot skill files.
# on:
#   push:
#     branches: [main]
#   workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    toolsets: [default]
  edit:
safe-outputs:
  create-pull-request:
    title-prefix: "docs: "
    labels: [documentation]
    draft: false
  noop:
---

# Skills Update Agent

You maintain the Copilot skill files at `.github/skills/*/SKILL.md`. Each skill documents code patterns, file paths, structs, and function signatures for a domain of the codebase.

Be concise. Conserve tokens. Only read files relevant to recent changes.

## Task

1. Search for open PRs with label `documentation` or title containing "skills". If one exists, `noop` — do not duplicate.
2. Analyze recent commits (push event `before`/`after` SHAs, or last 10 on `main`).
3. Identify which changed files overlap with content documented in skills. List all skill directories under `.github/skills/` and read only those SKILL.md files whose domain is plausibly affected.
4. If a skill references stale paths, renamed functions, changed signatures, moved packages, or removed APIs — edit it. Otherwise `noop`.

## Rules

- **Never create new skills.** Only update existing ones.
- Make minimal, targeted edits — do not rewrite unaffected sections.
- Preserve each skill's structure, style, and frontmatter.
- Ignore internal-only changes (variable renames inside function bodies, test-only changes) unless they alter documented patterns.
- If skill files were already updated in this push, verify correctness — `noop` if accurate.
- Be concise in your changes, conserve tokens.

## Output

Edit affected SKILL.md files and create a PR titled `Update skills for recent changes` summarizing what changed and which skills were updated. If nothing needs updating, `noop`.
