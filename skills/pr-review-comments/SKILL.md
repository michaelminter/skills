---
name: pr-review-comments
description: Fetches, evaluates, and triages unresolved review comments on an active pull request, assessing validity and required actions against the current codebase, with optional filtering by commit hash. Use when asked to "review PR comments", "check the feedback on my PR", "what did reviewers say", "triage review comments", or given a PR number/URL or commit hash to check comments against. Read-only: produces a triage and plan, never modifies code. Do not use for writing a PR description or performing a code review of the diff itself.
---

# Review PR Comments

## Objective
Retrieve unresolved review comments from the remote pull request, cross-reference them with the current codebase, and triage them by validity and required effort. This skill is for reviewing and planning only — do not modify any files; present the plan and stop.

## Instructions

1. **Locate the PR**:
   - If a PR number or URL is provided in the prompt, target that PR.
   - Otherwise, use the active branch's PR: `gh pr view --json number,url,headRepositoryOwner,headRepository`.
   - If no PR exists for the branch (or `gh` is not authenticated / not in a repo), report this to the user and stop.

2. **Fetch Review Threads**:
   - Use the GraphQL API so that resolution state is available. Paginate `reviewThreads` (and `comments` within each thread) if `hasNextPage` is true:
     ```sh
     gh api graphql -f query='
       query($owner: String!, $repo: String!, $number: Int!, $cursor: String) {
         repository(owner: $owner, name: $repo) {
           pullRequest(number: $number) {
             reviewThreads(first: 100, after: $cursor) {
               pageInfo { hasNextPage endCursor }
               nodes {
                 isResolved
                 isOutdated
                 path
                 line
                 comments(first: 100) {
                   nodes { author { login } body createdAt commit { oid } url }
                 }
               }
             }
           }
         }
       }' -F owner=OWNER -F repo=REPO -F number=NUMBER
     ```
   - **Skip every thread where `isResolved` is `true`.** Resolved threads are out of scope regardless of whether the code changed. Do not list them in the output.
   - Also fetch top-level conversation comments (`gh api repos/OWNER/REPO/issues/NUMBER/comments`) and include those that contain review feedback (ignore bot status updates, CI noise, and pure acknowledgements like "LGTM").

3. **Commit Hash Filtering** (only if a commit hash is provided in the prompt):
   - Get the commit's timestamp: `git show -s --format=%cI <hash>`.
   - Keep a thread only if its first comment's `createdAt` is at or after that timestamp. Drop everything else.
   - Apply the same rule to top-level conversation comments.

4. **Evaluate Validity** (for each remaining thread):
   - Inspect the referenced `path` and `line` in the local working tree. If the thread is `isOutdated`, locate the corresponding code by content rather than line number.
   - Verify whether the feedback is factually accurate and aligned with project conventions by reading the code. Do not run tests or execute code.
   - Check whether the current working tree already contains a fix for the issue.

5. **Triage Categories**:
   Classify each thread under one of the following statuses:
   - **Action Required**: Valid critique or bug report requiring code changes, tests, or documentation updates.
   - **Needs Clarification**: Ambiguous, subjective, or architectural feedback needing discussion before implementation.
   - **Fixed Locally**: Valid point, and the current working tree already contains the fix, but the thread is not yet marked resolved on GitHub.
   - **Dismiss / Out of Scope**: Inaccurate assumption, false positive, or request outside the scope of this PR.

6. **Output Structure**:
   - **Triage Matrix**: Summary table with columns: `Author`, `File:Line`, `Comment Summary`, `Status`, and `Effort (Low/Med/High)`. Link each row to the thread `url`.
   - **Action Plan**: For all **Action Required** items, provide a concise remediation plan with exact file paths and suggested code modifications. Present this as a plan only — do not apply any changes.
