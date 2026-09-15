---
name: pr-description-writing
description: Write or edit a copy-ready pull request description from a git diff. Use when asked to write, revise, describe, or summarize a PR body. Do not use for title-only requests.
---

## Content contract

- Analyze the current feature-branch diff against its base branch.
- Describe the final behavior and intent, not intermediate work or line-by-line implementation details.
- Keep the body concise. Scale context and risk detail to the size of the change.
- Describe only the final squash-commit outcome.
- Use exactly these sections:
  - `## What does this PR do?`
  - `## Why are we making this change?`
  - `## How to test`
  - `## Breaking changes / Risks`
- Put the Mermaid diagram under `## What does this PR do?`.
- In `## How to test`, provide verification instructions without reporting tests already run.
- For visual changes, include a before/after table with images.
- For benchmarks, include a before/after table using the target branch as the baseline.
- Use `None` when there are no meaningful breaking changes or risks.

## Output contract

These requirements are mandatory unless the user explicitly overrides them:

- Return only the final PR description. Do not add an introduction, explanation, or closing text.
- Return copy-ready Markdown source by wrapping the entire response in one outer code fence labeled `markdown`.
- Use at least four backticks for the outer fence so nested code fences remain valid.
- Include at least one Mermaid diagram in every PR description, even for a small change.
- Place the diagram under `## What does this PR do?`.
- The diagram must illustrate the primary request, data, or state flow described by the PR.
- Write the diagram as a nested triple-backtick `mermaid` block.
