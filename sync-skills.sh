#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
destinations=(
  "$HOME/.agents/skills"
  "$HOME/.claude/skills"
)

skill_count=0

for destination in "${destinations[@]}"; do
  mkdir -p "$destination"
done

for skill_dir in "$script_dir"/*/; do
  [[ -f "${skill_dir}SKILL.md" ]] || continue

  skill_name="$(basename "$skill_dir")"
  ((skill_count += 1))

  for destination in "${destinations[@]}"; do
    target="$destination/$skill_name"
    rm -rf "$target"
    cp -R "$skill_dir" "$target"
    printf 'Synced %s to %s\n' "$skill_name" "$target"
  done
done

if ((skill_count == 0)); then
  printf 'No skill directories containing SKILL.md were found in %s\n' "$script_dir" >&2
  exit 1
fi

printf 'Synced %d skill(s) to both destinations.\n' "$skill_count"
