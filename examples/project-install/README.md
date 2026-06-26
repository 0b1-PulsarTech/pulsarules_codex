# Project install walkthrough

Install the standards skills into a single project's `.claude/skills/` so they activate when
working in that repo.

## 1. Build the installer (once)

```sh
cd pulsarules_codex
go build -o build/bin/pulsarules_codex-installer ./cmd/pulsarules_codex-installer
```

## 2. Generate the skills (after any manifest/standard change)

```sh
./build/bin/pulsarules_codex-installer generate
./build/bin/pulsarules_codex-installer validate
```

## 3. Install into your project

```sh
# all skills:
./build/bin/pulsarules_codex-installer install --project /path/to/my-service --all

# or start with just the router:
./build/bin/pulsarules_codex-installer install --project /path/to/my-service --router-only

# or a selected subset:
./build/bin/pulsarules_codex-installer install --project /path/to/my-service \
  --skills go-style,errors-logging,commits
```

One of `--all`, `--skills`, or `--router-only` is required; on an interactive terminal, omitting all
three prompts with a numbered list instead. This writes
`/path/to/my-service/.claude/skills/<id>/SKILL.md` for each selected skill, plus the mandatory
baseline (`project-router` and every `always_load` skill, which are always installed) and any skill a
selection's `compose_skills` pulls in transitively - the installer prints what it added and why.

## 4. Wire the project's AGENTS.md to the standards

Copy `templates/docs/AGENTS.md.tmpl` into the project as `AGENTS.md`, fill in
`{{.ProjectName}}` and `{{.ProjectDescription}}`, and adjust the "Engineering standards" section to
point at this repo. The stop-signs block is already generalized.

## 5. Verify

```sh
ls /path/to/my-service/.claude/skills/
cat /path/to/my-service/.claude/skills/project-router/SKILL.md | head
```

When you now work in `my-service`, the `project-router` skill is the mandatory first step: it
classifies the task and loads the required skills in composition order.

## Updating

Re-run `generate` + `install --project ... --all` (or your chosen `--skills`) after pulling changes
to this repo. Installation is
idempotent (overwrites). Commit the installed `.claude/skills/` into the project if you want the
team to share the exact skill versions.
