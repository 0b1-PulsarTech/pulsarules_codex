#!/usr/bin/env bash
# Installed by pulsarules_cli; remove or edit this file to disable.
#
# Thin orchestrator for the skill-router reminder hook.
#
# It forwards the hook mode ($1) and the stdin payload to the installed
# pulsarules_cli binary, which emits the hookSpecificOutput JSON from
# the embedded reminder templates - so the contract text has a single source of
# truth (the templates), not a copy baked into this script.
#
# Non-blocking by design: if the binary is not present (or not executable) it
# exits 0 so a missing tool never blocks an edit. See README.md for the rationale.
bin="$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"
[ -x "$bin" ] || exit 0

# Generic Go reads only the host-neutral PULSARULES_PROJECT_DIR and
# PULSARULES_SKILLS_DIR; Claude Code only sets CLAUDE_PROJECT_DIR, so this
# Claude-specific script is where that translation - and the Claude skills
# layout - happens.
export PULSARULES_PROJECT_DIR="$CLAUDE_PROJECT_DIR"
export PULSARULES_SKILLS_DIR="$CLAUDE_PROJECT_DIR/.claude/skills"
exec "$bin" hook "$1"
