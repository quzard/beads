---
name: beads
description: Use before any code, config, or doc edit in a repository that contains .beads, including a small change. Run bd prime first. Create and claim an issue when none matches, and close it with the commit id when the change is finished. Also use for ready work, blockers, and handoff.
---

# Beads

Use Beads as the shared project task system. Local plans, scratch files, and personal memories are useful, but they are not the durable source of truth for project work.

## First Step

When the repository contains `.beads`, run this before the first edit. Do this even when the user did not mention bd.

```bash
bd prime
```

If that prints nothing, check whether the repository has an active Beads workspace:

```bash
bd where
```

## Preferred Route

Use the `bd` CLI when shell access is available. It is the most compact and direct Beads interface.

## Core CLI Workflow

1. Find work:

```bash
bd ready
bd list --status=open
bd list --status=in_progress
```

2. Inspect before editing:

```bash
bd show <id>
```

3. Claim work atomically:

```bash
bd update <id> --claim
```

4. Create durable follow-up work when implementation reveals new tasks:

```bash
bd create "Short title" --description="Why this exists and what needs to be done" --type=task --priority=2
```

5. Close completed work:

```bash
bd close <id> --reason="Completed"
```

## What Belongs In Beads

Use Beads for:

- every code, config, or doc change in a repository that contains `.beads`, including a small edit
- shared project tasks
- blockers and dependencies
- discovered follow-up work
- work that must survive thread reset, compaction, or handoff
- status that another person or agent should be able to resume

Use agent-local planning tools only for the current turn's execution checklist. Do not treat them as shared project state.

## Rules

- Do not create markdown TODO files as the source of truth when Beads is available.
- Do not use `bd edit`; it opens an interactive editor. Use `bd update` flags instead.
- Prefer `--json` when parsing `bd` output programmatically.
- Grok does not inject SessionStart output and skips gitignored project rules. When `bd prime` context is absent, run it before editing. Keep `AGENTS.md` and `.grok/rules/` out of `.gitignore`.
- Do not auto-close or mutate tasks unless the work is actually complete.
