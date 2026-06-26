# Global install walkthrough

Install the standards skills into `~/.claude/skills/` so they are available across every project
on the machine (not committed to any single repo).

## 1. Build the installer

```sh
cd pulsarules_codex
go build -o build/bin/pulsarules_codex-installer ./cmd/pulsarules_codex-installer
```

## 2. Generate + validate

```sh
./build/bin/pulsarules_codex-installer generate
./build/bin/pulsarules_codex-installer validate
```

## 3. Install globally

```sh
# all skills:
./build/bin/pulsarules_codex-installer install --global

# or just the router (recommended starting point):
./build/bin/pulsarules_codex-installer install --global --router-only
```

This writes `~/.claude/skills/<id>/SKILL.md` for each selected skill.

## 4. Package (optional, for distribution)

```sh
./build/bin/pulsarules_codex-installer package
# -> build/standards-skills.zip (27 SKILL.md files)
```

Distribute the zip to teammates; they can unzip into `~/.claude/skills/` without building Go, or
use `templates/installers/install.sh.tmpl` as a no-Go fallback copier.

## 5. Verify

```sh
ls ~/.claude/skills/
~/.claude/skills/project-router/SKILL.md   # should exist
```

## Project vs global

- **Global** (`--global`): skills available everywhere; good for the router + baseline
  (go-style, errors-logging, code-placement, code-minimalism, commits).
- **Project** (`--project PATH`): skills committed to one repo; good when a project pins a
  specific standards version or needs a project-specific skill overlay.

You can mix both: a global router + baseline, plus a project install for the heavier skills
(database-persistence, eventing-outbox, integration-tests) only in repos that need them.
