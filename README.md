# Skills

This repository is the source of truth for locally maintained agent skills. Use
it to create, review, and version reusable skill instructions and their supporting
files before installing them into the local skill directories used by agent
tools and Claude.

## Repository layout

Each skill lives in its own top-level directory and must contain a `SKILL.md`
file. Any supporting files should be kept inside the same directory.

```text
skills/
├── review-pr-comments/
│   └── SKILL.md
├── writing-pr/
│   └── SKILL.md
└── sync-skills.sh
```

Only top-level directories containing `SKILL.md` are copied by the sync script.

## Syncing skills

From anywhere, run:

```bash
/path/to/skills/sync-skills.sh
```

Or, from the repository root:

```bash
./sync-skills.sh
```

The script installs every skill into both locations:

- `~/.agents/skills/<skill-name>`
- `~/.claude/skills/<skill-name>`

It creates the parent directories when needed. If a matching skill directory
already exists at either destination, the script removes that entire directory
and replaces it with the repository copy. This ensures removed or renamed source
files do not linger in an installed skill.

Skills installed at either destination but not present in this repository are
left untouched.

## Adding or updating a skill

1. Create or edit a top-level skill directory.
2. Ensure the directory contains a `SKILL.md` file.
3. Keep any scripts, references, templates, or assets used by the skill inside
   that directory.
4. Run `./sync-skills.sh` to install the current repository versions.
5. Commit the source changes to this repository so they can be reviewed and
   restored later.

The script requires Bash and standard Unix utilities (`cp`, `rm`, and `mkdir`).
