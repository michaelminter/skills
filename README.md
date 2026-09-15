# Skills

This repository is the source of truth for locally maintained agent skills. Use
it to create, review, and version reusable skill instructions and their supporting
files before installing them into the local skill directories used by agent
tools and Claude.

## Repository layout

Each skill lives in its own directory under `skills/` and must contain a
`SKILL.md` file. Any supporting files should be kept inside the same directory.

```text
repository-root/
├── .gitignore
├── go.mod
├── main.go
├── main_test.go
└── skills/
    ├── review-pr-comments/
    │   └── SKILL.md
    └── writing-pr/
        └── SKILL.md
```

Only immediate child directories of `skills/` containing `SKILL.md` are managed
by the CLI.

## Build the CLI

Install Go 1.22 or newer, then compile the CLI from the repository root:

```bash
mkdir -p bin
go build -o bin/skills .
```

The resulting `bin/skills` executable has no runtime dependency on Go. Keep it
inside the repository, or run it while your current directory is somewhere
inside this repository.

To build executables for the common Linux and macOS platforms:

```bash
mkdir -p dist
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o dist/skills-linux-amd64 .
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o dist/skills-linux-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/skills-macos-amd64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/skills-macos-arm64 .
```

## CLI commands

Run `./bin/skills help` to see all commands:

```text
skills sync
skills reset
skills add <skill>
skills delete <skill>
skills rename <old> <new>
skills help
```

`skills sync` creates a symbolic link for every repository skill in each
existing location:

- `~/.agents/skills/<skill-name>`
- `~/.claude/skills/<skill-name>`

It does not create `~/.agents` or `~/.claude` when those tool directories do not
already exist. For each existing tool directory, it creates the `skills/` child
directory when needed. If a matching skill already exists at a destination, the
command replaces it with a link to the source directory in this repository.
Changes made here are therefore immediately available to installed tools.

Skills installed at either destination but not present in this repository are
left untouched. `skills reset` removes the repository's skills from both
destinations while preserving unrelated skills.

## Adding or updating a skill

1. Run `./bin/skills add <skill>` to create a directory under `skills/` with a
   starter `SKILL.md` file, or edit an existing skill.
2. Keep any scripts, references, templates, or assets used by the skill inside
   that directory.
3. Run `./bin/skills sync` to link the current repository skills.
4. Commit the source changes to this repository so they can be reviewed and
   restored later.

Use `./bin/skills delete <skill>` to delete a skill and all of its files from
this repository. Installed links can subsequently be removed with
`./bin/skills reset`.

Use `./bin/skills rename <old> <new>` to rename a skill. The command updates the
skill's `name:` frontmatter, removes the old links from both installation
directories, and re-syncs all repository skills under their current names.

## Install with npx

To install the skills directly from this GitHub repository with the Skills CLI,
run:

```bash
npx skills add michaelminter/skills
```
