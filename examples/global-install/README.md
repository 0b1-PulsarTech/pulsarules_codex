# Global install walkthrough

Install the standards skills into `~/.claude/skills/` so they are available across every project
on the machine (not committed to any single repo).

## 1. Build the installer

```sh
cd pulsarules_codex
go build -o build/bin/pulsarules_cli ./cmd/pulsarules_cli
```

## 2. Generate + validate

```sh
./build/bin/pulsarules_cli generate
./build/bin/pulsarules_cli validate
```

## 3. Install globally

```sh
# all skills:
./build/bin/pulsarules_cli install --global --all

# or just the router (recommended starting point):
./build/bin/pulsarules_cli install --global --router-only
```

`install` requires exactly one of `--all`, `--skills a,b,c`, or `--router-only`; with none of them and
no interactive terminal (e.g. CI), it fails rather than guessing. `--target` (repeatable, default
`claude`) picks the install layout - `opencode`, `agents`, and `cursor` also exist; see
[INSTALL.md](../../INSTALL.md#install) for the per-target details.

This writes `~/.claude/skills/<id>/SKILL.md` for each selected skill.

## 4. Package (optional, for distribution)

```sh
./build/bin/pulsarules_cli package
# -> build/standards-skills.zip (one SKILL.md per skill)
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
