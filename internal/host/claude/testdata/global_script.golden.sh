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

# A project install has its own hook; running both would remind twice. The project hook runs
# as `bash script`, so presence is what counts, not the executable bit.
project_hook="$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh"
if [ -f "$project_hook" ] && [ ! "$project_hook" -ef "$0" ]; then
  exit 0
fi
bin='/home/user/.claude/bin/pulsarules_cli'
[ -x "$bin" ] || exit 0

# Generic Go reads only the host-neutral PULSARULES_PROJECT_DIR,
# PULSARULES_SKILLS_DIR and PULSARULES_LOG_PATH; Claude Code only sets
# CLAUDE_PROJECT_DIR, so this Claude-specific script is where that
# translation - and the Claude skills/log layout - happens.
export PULSARULES_PROJECT_DIR="$CLAUDE_PROJECT_DIR"
export PULSARULES_SKILLS_DIR="$CLAUDE_PROJECT_DIR/.claude/skills":'/home/user/.claude/skills'
export PULSARULES_LOG_PATH='/home/user/.claude/hook-execution.log'
exec "$bin" hook "$1"
