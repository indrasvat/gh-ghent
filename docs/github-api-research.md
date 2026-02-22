# GitHub API Research for PR Review Operations

Research conducted 2026-02-21. Covers GraphQL mutations/queries for PR review threads,
REST API endpoints for checks/workflow runs, and the go-gh library for building CLI extensions.

---

## Table of Contents

1. [GraphQL: Fetching Unresolved PR Review Threads](#1-graphql-fetching-unresolved-pr-review-threads)
2. [GraphQL: Resolving a Review Thread](#2-graphql-resolving-a-review-thread)
3. [GraphQL: Unresolving a Review Thread](#3-graphql-unresolving-a-review-thread)
4. [GraphQL: Minimizing (Hiding) Comments](#4-graphql-minimizing-hiding-comments)
5. [Key GraphQL Types](#5-key-graphql-types)
6. [REST API: Check Runs and Annotations](#6-rest-api-check-runs-and-annotations)
7. [REST API: Workflow Runs and Jobs](#7-rest-api-workflow-runs-and-jobs)
8. [REST API: Pull Request Reviews and Comments](#8-rest-api-pull-request-reviews-and-comments)
9. [gh CLI: Viewing Run Logs](#9-gh-cli-viewing-run-logs)
10. [go-gh Library](#10-go-gh-library)
11. [Rate Limiting](#11-rate-limiting)

---

## 1. GraphQL: Fetching Unresolved PR Review Threads

### Full Query with Comments, File Paths, and Line Numbers

```graphql
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      id
      reviewThreads(first: 100) {
        totalCount
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          resolvedBy {
            login
          }
          viewerCanResolve
          viewerCanUnresolve
          comments(first: 50) {
            totalCount
            nodes {
              id
              body
              author {
                login
              }
              path
              position
              originalPosition
              diffHunk
              createdAt
              updatedAt
              url
              pullRequestReview {
                state
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
```

### Usage with gh CLI

```bash
gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes {
            id
            isResolved
            path
            line
            startLine
            comments(first: 50) {
              nodes {
                id
                body
                author { login }
                path
                diffHunk
                createdAt
              }
            }
          }
        }
      }
    }
  }
' -f owner='OWNER' -f repo='REPO' -F pr=123
```

### Filtering for Unresolved Only (Client-Side)

The GraphQL API does not support server-side filtering of `reviewThreads` by `isResolved`.
You must fetch all threads and filter client-side:

```javascript
const unresolvedThreads = reviewThreads.nodes.filter(t => !t.isResolved);
```

### Paginated Query (for PRs with many threads)

```graphql
query($owner: String!, $repo: String!, $pr: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100, after: $cursor) {
        nodes {
          id
          isResolved
          path
          line
          startLine
          comments(first: 10) {
            nodes {
              id
              body
              author { login }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
```

---

## 2. GraphQL: Resolving a Review Thread

### Mutation

```graphql
mutation ResolveThread($threadId: ID!) {
  resolveReviewThread(input: { threadId: $threadId }) {
    thread {
      id
      isResolved
      resolvedBy {
        login
      }
      viewerCanUnresolve
    }
  }
}
```

### Input Type: ResolveReviewThreadInput

| Field             | Type   | Required | Description                                      |
|-------------------|--------|----------|--------------------------------------------------|
| `threadId`        | ID!    | Yes      | The Node ID of the review thread to resolve       |
| `clientMutationId`| String | No       | A unique identifier for the client performing it  |

### Return Fields

| Field              | Type                        | Description                                    |
|--------------------|-----------------------------|------------------------------------------------|
| `clientMutationId` | String                      | A unique identifier for the client             |
| `thread`           | PullRequestReviewThread     | The review thread that was marked as resolved  |

### Usage with gh CLI

```bash
# First, get the thread ID from the query above, then:
gh api graphql -f query='
  mutation($threadId: ID!) {
    resolveReviewThread(input: {threadId: $threadId}) {
      thread {
        id
        isResolved
      }
    }
  }
' -f threadId='PRRT_kwDOxxxxxx'
```

### Bulk Resolve All Unresolved Threads

```bash
#!/bin/bash
OWNER="myorg"
REPO="myrepo"
PR=123

# Fetch all unresolved thread IDs
THREAD_IDS=$(gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes { id isResolved }
        }
      }
    }
  }
' -f owner="$OWNER" -f repo="$REPO" -F pr=$PR \
  --jq '.data.repository.pullRequest.reviewThreads.nodes[] | select(.isResolved == false) | .id')

# Resolve each thread
for THREAD_ID in $THREAD_IDS; do
  gh api graphql -f query='
    mutation($threadId: ID!) {
      resolveReviewThread(input: {threadId: $threadId}) {
        thread { id isResolved }
      }
    }
  ' -f threadId="$THREAD_ID"
  echo "Resolved: $THREAD_ID"
done
```

### Permissions Required

- Repository Contents: Read and Write access
- Personal Access Token: `repo` scope (classic) or `Pull requests` permission (fine-grained)

---

## 3. GraphQL: Unresolving a Review Thread

### Mutation

```graphql
mutation UnresolveThread($threadId: ID!) {
  unresolveReviewThread(input: { threadId: $threadId }) {
    thread {
      id
      isResolved
    }
  }
}
```

### Input Type: UnresolveReviewThreadInput

| Field             | Type   | Required | Description                                        |
|-------------------|--------|----------|----------------------------------------------------|
| `threadId`        | ID!    | Yes      | The Node ID of the review thread to unresolve       |
| `clientMutationId`| String | No       | A unique identifier for the client performing it    |

### Return Fields

| Field              | Type                        | Description                                      |
|--------------------|-----------------------------|--------------------------------------------------|
| `clientMutationId` | String                      | A unique identifier for the client               |
| `thread`           | PullRequestReviewThread     | The review thread that was marked as unresolved  |

### Usage with gh CLI

```bash
gh api graphql -f query='
  mutation($threadId: ID!) {
    unresolveReviewThread(input: {threadId: $threadId}) {
      thread {
        id
        isResolved
      }
    }
  }
' -f threadId='PRRT_kwDOxxxxxx'
```

---

## 4. GraphQL: Minimizing (Hiding) Comments

The `minimizeComment` mutation can hide individual comments (not entire threads).
This applies to comments on Issues, Commits, Pull Requests, and Gists.

### Mutation

```graphql
mutation MinimizeComment($id: ID!, $classifier: ReportedContentClassifiers!) {
  minimizeComment(input: { subjectId: $id, classifier: $classifier }) {
    minimizedComment {
      isMinimized
      minimizedReason
      viewerCanMinimize
    }
  }
}
```

### Input Type: MinimizeCommentInput

| Field             | Type                          | Required | Description                                    |
|-------------------|-------------------------------|----------|------------------------------------------------|
| `subjectId`       | ID!                           | Yes      | The Node ID of the comment to minimize          |
| `classifier`      | ReportedContentClassifiers!   | Yes      | The classification for the comment              |
| `clientMutationId`| String                        | No       | A unique identifier for the client              |

### ReportedContentClassifiers Enum Values

| Value       | Description                          |
|-------------|--------------------------------------|
| `SPAM`      | A spammy comment                     |
| `ABUSE`     | An abusive or harassing comment      |
| `OFF_TOPIC` | An off-topic comment                 |
| `OUTDATED`  | An outdated comment                  |
| `RESOLVED`  | A resolved comment                   |
| `DUPLICATE` | A duplicate comment                  |

### Usage with gh CLI

```bash
gh api graphql -f query='
  mutation($id: ID!, $classifier: ReportedContentClassifiers!) {
    minimizeComment(input: {subjectId: $id, classifier: $classifier}) {
      minimizedComment {
        isMinimized
        minimizedReason
      }
    }
  }
' -f id='IC_kwDOxxxxxx' -f classifier='OUTDATED'
```

### Important Notes

- `minimizeComment` hides an individual comment, NOT an entire review thread.
- To "minimize" a thread, you would need to minimize each comment in it individually.
- The `OUTDATED` classifier is most appropriate for review comments that have been addressed.
- There is a known behavioral difference between the GraphQL mutation and the GitHub UI button:
  the UI shows "This comment has been minimized" generically, while the API always requires
  a specific classifier.
- Requires write permissions on issues/PRs.

---

## 5. Key GraphQL Types

### PullRequestReviewThread

| Field               | Type                                    | Description                                               |
|---------------------|-----------------------------------------|-----------------------------------------------------------|
| `id`                | ID!                                     | Node ID of the thread                                     |
| `databaseId`        | Int                                     | Primary key from the database                             |
| `isResolved`        | Boolean!                                | Whether the thread has been marked as resolved            |
| `isOutdated`        | Boolean!                                | Whether the thread is outdated (code has changed)         |
| `resolvedBy`        | User                                    | The user who resolved this thread                         |
| `path`              | String!                                 | The file path this thread concerns                        |
| `line`              | Int                                     | The line number for single-line threads                   |
| `startLine`         | Int                                     | The starting line for multi-line threads                  |
| `comments`          | PullRequestReviewCommentConnection!     | The comments in this thread                               |
| `pullRequest`       | PullRequest!                            | The pull request containing this thread                   |
| `repository`        | Repository!                             | The associated repository                                 |
| `viewerCanReply`    | Boolean!                                | Whether current viewer can reply                          |
| `viewerCanResolve`  | Boolean!                                | Whether current viewer can resolve                        |
| `viewerCanUnresolve`| Boolean!                                | Whether current viewer can unresolve                      |

### PullRequestReviewComment

| Field                | Type               | Description                                                  |
|----------------------|--------------------|--------------------------------------------------------------|
| `id`                 | ID!                | Global identifier                                            |
| `body`               | String!            | The comment body text                                        |
| `bodyHTML`           | HTML!              | The comment body rendered to HTML                            |
| `bodyText`           | String!            | The comment body rendered as plain text                      |
| `path`               | String!            | The file path the comment applies to                         |
| `position`           | Int                | The line index in the diff (null if outdated)                |
| `originalPosition`   | Int!               | The original line index in the diff                          |
| `diffHunk`           | String!            | The diff hunk context for the comment                        |
| `commit`             | Commit!            | The commit associated with the comment                       |
| `originalCommit`     | Commit!            | The original commit associated with the comment              |
| `author`             | Actor              | The author of the comment                                    |
| `editor`             | Actor              | The editor of the comment (if edited)                        |
| `pullRequest`        | PullRequest!       | The associated pull request                                  |
| `pullRequestReview`  | PullRequestReview! | The associated pull request review                           |
| `url`                | URI!               | The HTTP URL permalink for this review comment               |
| `createdAt`          | DateTime!          | When the comment was created                                 |
| `updatedAt`          | DateTime!          | When the comment was last updated                            |
| `lastEditedAt`       | DateTime           | When the editor made the last edit                           |
| `createdViaEmail`    | Boolean!           | Whether the comment was created via email                    |
| `viewerCanDelete`    | Boolean!           | Whether the viewer can delete this comment                   |
| `viewerCanEdit`      | Boolean!           | Whether the viewer can edit this comment                     |
| `viewerCanReact`     | Boolean!           | Whether the viewer can react to this comment                 |
| `viewerDidAuthor`    | Boolean!           | Whether the viewer authored this comment                     |
| `reactionGroups`     | [ReactionGroup!]   | Reactions grouped by emoji content                           |
| `reactions`          | ReactionConnection!| Reactions list with pagination                               |

### PullRequestReview

| Field          | Type                                    | Description                                          |
|----------------|-----------------------------------------|------------------------------------------------------|
| `id`           | ID!                                     | Node ID                                              |
| `state`        | PullRequestReviewState!                 | PENDING, COMMENTED, APPROVED, CHANGES_REQUESTED, DISMISSED |
| `body`         | String!                                 | The review body text                                 |
| `author`       | Actor                                   | The author of the review                             |
| `comments`     | PullRequestReviewCommentConnection!     | Comments in this review                              |
| `commit`       | Commit                                  | The commit the review was made on                    |
| `createdAt`    | DateTime!                               | When the review was created                          |
| `submittedAt`  | DateTime                                | When the review was submitted                        |
| `updatedAt`    | DateTime!                               | When the review was last updated                     |
| `url`          | URI!                                    | Permalink URL                                        |

---

## 6. REST API: Check Runs and Annotations

### List Check Runs for a Git Reference

```
GET /repos/{owner}/{repo}/commits/{ref}/check-runs
```

Parameters:
- `ref` (path, required): SHA, branch name, or tag name
- `check_name` (query): Filter by check name
- `status` (query): `queued`, `in_progress`, `completed`
- `filter` (query): `latest` (default) or `all`
- `per_page` (query): Results per page (max 100, default 30)
- `page` (query): Page number

Response includes `total_count` and `check_runs[]` array with:
- `id`, `name`, `status`, `conclusion`
- `started_at`, `completed_at`
- `output.title`, `output.summary`, `output.text`
- `output.annotations_count`
- `html_url`, `details_url`

### Get a Check Run

```
GET /repos/{owner}/{repo}/check-runs/{check_run_id}
```

### List Check Run Annotations

```
GET /repos/{owner}/{repo}/check-runs/{check_run_id}/annotations
```

This is the key endpoint for getting lint errors. Returns an array of annotation objects:

```json
[
  {
    "path": "src/main.go",
    "start_line": 42,
    "end_line": 42,
    "start_column": 5,
    "end_column": 20,
    "annotation_level": "failure",
    "title": "golangci-lint",
    "message": "unused variable 'x'",
    "raw_details": "..."
  }
]
```

Annotation fields:
| Field              | Type    | Description                              |
|--------------------|---------|------------------------------------------|
| `path`             | string  | File path relative to repo root          |
| `start_line`       | integer | Starting line number                     |
| `end_line`         | integer | Ending line number                       |
| `start_column`     | integer | Starting column (optional)               |
| `end_column`       | integer | Ending column (optional)                 |
| `annotation_level` | string  | `notice`, `warning`, or `failure`        |
| `title`            | string  | Title of the annotation                  |
| `message`          | string  | Detailed message (e.g., lint error text) |
| `raw_details`      | string  | Raw details (optional)                   |

### Annotation Limits

- 10 warning annotations and 10 error annotations per step
- 50 annotations per job (sum of all steps)
- 50 annotations per run

### List Check Runs in a Check Suite

```
GET /repos/{owner}/{repo}/check-suites/{check_suite_id}/check-runs
```

### Usage with gh CLI

```bash
# List check runs for a commit/branch
gh api repos/{owner}/{repo}/commits/{ref}/check-runs --jq '.check_runs[] | {name, status, conclusion}'

# Get annotations (lint errors) for a specific check run
gh api repos/{owner}/{repo}/check-runs/{check_run_id}/annotations

# Get failing check runs only
gh api repos/{owner}/{repo}/commits/{ref}/check-runs \
  --jq '.check_runs[] | select(.conclusion == "failure") | {id, name, conclusion}'

# Then get annotations for each failing run
gh api repos/{owner}/{repo}/check-runs/{check_run_id}/annotations \
  --jq '.[] | {path, start_line, annotation_level, message}'
```

### Complete Workflow: Get Lint Errors from PR Checks

```bash
#!/bin/bash
OWNER="myorg"
REPO="myrepo"
PR=123

# Get the head SHA of the PR
HEAD_SHA=$(gh api repos/$OWNER/$REPO/pulls/$PR --jq '.head.sha')

# Get all failing check runs
FAILING_CHECKS=$(gh api repos/$OWNER/$REPO/commits/$HEAD_SHA/check-runs \
  --jq '.check_runs[] | select(.conclusion == "failure") | .id')

# Get annotations from each failing check
for CHECK_ID in $FAILING_CHECKS; do
  echo "--- Check Run: $CHECK_ID ---"
  gh api repos/$OWNER/$REPO/check-runs/$CHECK_ID/annotations \
    --jq '.[] | "\(.path):\(.start_line): [\(.annotation_level)] \(.message)"'
done
```

### Check Run Status and Conclusion Values

Status: `queued`, `in_progress`, `completed`, `waiting`, `requested`, `pending`

Conclusion (when status=completed): `action_required`, `cancelled`, `failure`, `neutral`,
`success`, `skipped`, `stale`, `timed_out`, `startup_failure`

---

## 7. REST API: Workflow Runs and Jobs

### List Workflow Runs

```
GET /repos/{owner}/{repo}/actions/runs
```

Query parameters: `actor`, `branch`, `event`, `status`, `created`, `per_page`, `page`,
`exclude_pull_requests`, `check_suite_id`, `head_sha`

### Get a Workflow Run

```
GET /repos/{owner}/{repo}/actions/runs/{run_id}
```

### List Jobs for a Workflow Run

```
GET /repos/{owner}/{repo}/actions/runs/{run_id}/jobs
```

Query parameters: `filter` (`latest` or `all`), `per_page`, `page`

Response includes job objects with a `steps[]` array:

```json
{
  "total_count": 1,
  "jobs": [
    {
      "id": 123456,
      "run_id": 789012,
      "name": "lint",
      "status": "completed",
      "conclusion": "failure",
      "started_at": "2025-01-01T00:00:00Z",
      "completed_at": "2025-01-01T00:05:00Z",
      "steps": [
        {
          "name": "Run golangci-lint",
          "status": "completed",
          "conclusion": "failure",
          "number": 3,
          "started_at": "2025-01-01T00:02:00Z",
          "completed_at": "2025-01-01T00:04:00Z"
        }
      ]
    }
  ]
}
```

### Get a Specific Job

```
GET /repos/{owner}/{repo}/actions/jobs/{job_id}
```

Returns the same structure as above for a single job, including the `steps[]` array
with name, status, conclusion, and number for each step.

### Download Job Logs

```
GET /repos/{owner}/{repo}/actions/jobs/{job_id}/logs
```

Returns HTTP 302 redirect with `Location` header containing a temporary download URL
(expires in 1 minute). The response is a plain text file of the job's logs.

```bash
# Download job logs
gh api repos/{owner}/{repo}/actions/jobs/{job_id}/logs > job.log

# Or use curl to follow the redirect
curl -L -H "Authorization: Bearer $TOKEN" \
  "https://api.github.com/repos/{owner}/{repo}/actions/jobs/{job_id}/logs" \
  -o job.log
```

### Download Workflow Run Logs (All Jobs)

```
GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs
```

Returns HTTP 302 redirect to a ZIP archive containing logs for all jobs in the run.

```bash
# Download all logs as a zip
gh api repos/{owner}/{repo}/actions/runs/{run_id}/logs > logs.zip
```

### Download Logs for a Specific Run Attempt

```
GET /repos/{owner}/{repo}/actions/runs/{run_id}/attempts/{attempt_number}/logs
```

### List Jobs for a Specific Run Attempt

```
GET /repos/{owner}/{repo}/actions/runs/{run_id}/attempts/{attempt_number}/jobs
```

### Re-run Failed Jobs

```
POST /repos/{owner}/{repo}/actions/runs/{run_id}/rerun-failed-jobs
```

Body: `{ "enable_debug_logging": false }`

### Complete Workflow: Get Failing Step Logs

```bash
#!/bin/bash
OWNER="myorg"
REPO="myrepo"
RUN_ID=123456

# List jobs and find the failing one
FAILING_JOB_ID=$(gh api repos/$OWNER/$REPO/actions/runs/$RUN_ID/jobs \
  --jq '.jobs[] | select(.conclusion == "failure") | .id')

# Get failing step name
FAILING_STEP=$(gh api repos/$OWNER/$REPO/actions/jobs/$FAILING_JOB_ID \
  --jq '.steps[] | select(.conclusion == "failure") | .name')

echo "Failing step: $FAILING_STEP"

# Download the job log and grep for errors
gh api repos/$OWNER/$REPO/actions/jobs/$FAILING_JOB_ID/logs 2>/dev/null | \
  grep -i "error\|fail\|fatal" | head -50
```

---

## 8. REST API: Pull Request Reviews and Comments

### Reviews

```
GET    /repos/{owner}/{repo}/pulls/{pull_number}/reviews           # List reviews
POST   /repos/{owner}/{repo}/pulls/{pull_number}/reviews           # Create review
GET    /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}      # Get review
PUT    /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}      # Update review
DELETE /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}      # Delete pending review
GET    /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}/comments  # List review comments
PUT    /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}/dismissals  # Dismiss review
POST   /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{id}/events     # Submit review
```

Review states: `APPROVE`, `REQUEST_CHANGES`, `COMMENT`, `PENDING`, `DISMISSED`

### Review Comments

```
GET    /repos/{owner}/{repo}/pulls/comments                       # List all review comments in repo
GET    /repos/{owner}/{repo}/pulls/comments/{comment_id}          # Get a review comment
PATCH  /repos/{owner}/{repo}/pulls/comments/{comment_id}          # Update a review comment
DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}          # Delete a review comment
GET    /repos/{owner}/{repo}/pulls/{pull_number}/comments         # List comments on a PR
POST   /repos/{owner}/{repo}/pulls/{pull_number}/comments         # Create a review comment
POST   /repos/{owner}/{repo}/pulls/{pull_number}/comments/{comment_id}/replies  # Reply to a comment
```

### Important Note on REST vs GraphQL

The REST API does NOT expose review threads or their resolved/unresolved state.
To work with review threads (resolve, unresolve, list resolved status), you MUST
use the GraphQL API. The REST API only exposes individual comments and reviews.

---

## 9. gh CLI: Viewing Run Logs

### gh run view

```bash
# View a specific run
gh run view {run_id}

# View logs for a specific run
gh run view {run_id} --log

# View only failed logs
gh run view {run_id} --log-failed

# View logs for a specific job
gh run view {run_id} --job {job_id} --log

# View logs for a specific job (failed only)
gh run view {run_id} --job {job_id} --log-failed
```

### How gh run view --log Works Internally

1. Calls `GET /repos/{owner}/{repo}/actions/runs/{run_id}/logs` to get a redirect URL
2. Downloads the ZIP archive from the redirect URL
3. Extracts and parses the log files from the ZIP
4. Formats and displays them with step/job headers

The log ZIP archive contains one file per step, named like:
`{job_name}/{step_number}_{step_name}.txt`

### Known Issues with gh run view --log

There are several reported issues with the `--log` and `--log-failed` flags:

1. The command sometimes returns empty output (cli/cli#5011)
2. Log file name format in the ZIP has changed over time, causing parsing failures (cli/cli#10551)
3. Occasionally fails with "zip: not a valid zip file" error (cli/cli#8009)
4. API may return HTTP 500 for some runs (cli/cli#6936)

### Alternative: Using gh api Directly

```bash
# More reliable: download job logs directly via the REST API
JOB_ID=$(gh api repos/{owner}/{repo}/actions/runs/{run_id}/jobs \
  --jq '.jobs[] | select(.conclusion == "failure") | .id' | head -1)

gh api repos/{owner}/{repo}/actions/jobs/$JOB_ID/logs
```

### gh pr checks

```bash
# List check status for a PR
gh pr checks {pr_number}

# Watch checks (wait for completion)
gh pr checks {pr_number} --watch

# Get failing checks
gh pr checks {pr_number} --fail-fast
```

---

## 10. go-gh Library

### Package: github.com/cli/go-gh/v2

The official Go library for building GitHub CLI extensions. It inherits authentication,
host configuration, and other settings from the user's `gh` CLI environment.

### Installation

```bash
go get github.com/cli/go-gh/v2@latest
```

### Key Sub-packages

| Package                                      | Description                            |
|----------------------------------------------|----------------------------------------|
| `github.com/cli/go-gh/v2`                   | Top-level: Exec, ExecInteractive       |
| `github.com/cli/go-gh/v2/pkg/api`           | GraphQL, REST, HTTP clients            |
| `github.com/cli/go-gh/v2/pkg/repository`    | Repository context (owner/name/host)   |
| `github.com/cli/go-gh/v2/pkg/tableprinter`  | Formatted table output                 |
| `github.com/cli/go-gh/v2/pkg/term`          | Terminal capabilities                  |
| `github.com/cli/go-gh/v2/pkg/browser`       | Open URLs in browser                   |

### Getting Repository Context

```go
import "github.com/cli/go-gh/v2/pkg/repository"

func main() {
    repo, err := repository.Current()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Host: %s\n", repo.Host)    // e.g., "github.com"
    fmt.Printf("Owner: %s\n", repo.Owner)  // e.g., "myorg"
    fmt.Printf("Name: %s\n", repo.Name)    // e.g., "myrepo"
}
```

The `repository.Current()` function:
- Respects the `GH_REPO` environment variable
- Falls back to reading git remote configuration
- Returns `Repository{Host, Owner, Name}`

Also available:
```go
// Parse from string
repo, err := repository.Parse("owner/repo")
repo, err := repository.ParseWithHost("owner/repo", "github.com")
```

### Making GraphQL Queries

```go
import "github.com/cli/go-gh/v2/pkg/api"

// Using DefaultGraphQLClient (inherits auth from gh CLI)
client, err := api.DefaultGraphQLClient()
if err != nil {
    log.Fatal(err)
}

// Method 1: Raw query string with Do()
var result struct {
    Repository struct {
        PullRequest struct {
            ReviewThreads struct {
                Nodes []struct {
                    ID         string `json:"id"`
                    IsResolved bool   `json:"isResolved"`
                    Path       string `json:"path"`
                    Line       int    `json:"line"`
                    Comments   struct {
                        Nodes []struct {
                            Body   string `json:"body"`
                            Author struct {
                                Login string `json:"login"`
                            } `json:"author"`
                        } `json:"nodes"`
                    } `json:"comments"`
                } `json:"nodes"`
            } `json:"reviewThreads"`
        } `json:"pullRequest"`
    } `json:"repository"`
}

query := `query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
        pullRequest(number: $pr) {
            reviewThreads(first: 100) {
                nodes {
                    id
                    isResolved
                    path
                    line
                    comments(first: 50) {
                        nodes {
                            body
                            author { login }
                        }
                    }
                }
            }
        }
    }
}`

variables := map[string]interface{}{
    "owner": "myorg",
    "repo":  "myrepo",
    "pr":    123,
}

err = client.Do(query, variables, &result)
```

```go
// Method 2: Struct-based query with Query()
// Uses struct tags to derive the GraphQL query
var query struct {
    Repository struct {
        PullRequest struct {
            ReviewThreads struct {
                Nodes []struct {
                    ID         string
                    IsResolved bool
                    Path       string
                }
            } `graphql:"reviewThreads(first: 100)"`
        } `graphql:"pullRequest(number: $pr)"`
    } `graphql:"repository(owner: $owner, name: $repo)"`
}

variables := map[string]interface{}{
    "owner": graphql.String("myorg"),
    "repo":  graphql.String("myrepo"),
    "pr":    graphql.Int(123),
}

err = client.Query("ReviewThreads", &query, variables)
```

### Making GraphQL Mutations

```go
// Method 1: Raw mutation string with Do()
var result struct {
    ResolveReviewThread struct {
        Thread struct {
            ID         string `json:"id"`
            IsResolved bool   `json:"isResolved"`
        } `json:"thread"`
    } `json:"resolveReviewThread"`
}

mutation := `mutation($threadId: ID!) {
    resolveReviewThread(input: {threadId: $threadId}) {
        thread {
            id
            isResolved
        }
    }
}`

variables := map[string]interface{}{
    "threadId": "PRRT_kwDOxxxxxx",
}

err = client.Do(mutation, variables, &result)
```

```go
// Method 2: Struct-based mutation with Mutate()
var mutation struct {
    ResolveReviewThread struct {
        Thread struct {
            ID         string
            IsResolved bool
        }
    } `graphql:"resolveReviewThread(input: $input)"`
}

type ResolveReviewThreadInput struct {
    ThreadID string `json:"threadId"`
}

variables := map[string]interface{}{
    "input": ResolveReviewThreadInput{
        ThreadID: "PRRT_kwDOxxxxxx",
    },
}

err = client.Mutate("ResolveThread", &mutation, variables)
```

### Making REST API Calls

```go
import "github.com/cli/go-gh/v2/pkg/api"

// Using DefaultRESTClient (inherits auth from gh CLI)
client, err := api.DefaultRESTClient()
if err != nil {
    log.Fatal(err)
}

// GET request
var checkRuns struct {
    TotalCount int `json:"total_count"`
    CheckRuns  []struct {
        ID         int    `json:"id"`
        Name       string `json:"name"`
        Status     string `json:"status"`
        Conclusion string `json:"conclusion"`
    } `json:"check_runs"`
}

err = client.Get(
    fmt.Sprintf("repos/%s/%s/commits/%s/check-runs", owner, repo, sha),
    &checkRuns,
)

// Get annotations for a check run
var annotations []struct {
    Path            string `json:"path"`
    StartLine       int    `json:"start_line"`
    EndLine         int    `json:"end_line"`
    AnnotationLevel string `json:"annotation_level"`
    Message         string `json:"message"`
    Title           string `json:"title"`
}

err = client.Get(
    fmt.Sprintf("repos/%s/%s/check-runs/%d/annotations", owner, repo, checkRunID),
    &annotations,
)

// Download job logs (raw response)
resp, err := client.Request(
    "GET",
    fmt.Sprintf("repos/%s/%s/actions/jobs/%d/logs", owner, repo, jobID),
    nil,
)
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
```

### Custom Client Options

```go
// Create client with custom options
client, err := api.NewGraphQLClient(api.ClientOptions{
    Host:      "github.com",         // Override host
    AuthToken: "ghp_xxxx",          // Override auth token (rare)
    Timeout:   30 * time.Second,    // Request timeout
    EnableCache: true,              // Enable response caching
    CacheTTL:   5 * time.Minute,   // Cache TTL
    Headers: map[string]string{    // Custom headers
        "X-Custom": "value",
    },
    Log: os.Stderr,                // Enable request logging
    LogVerboseHTTP: true,          // Log headers and bodies
})
```

### Authentication Inheritance

go-gh automatically inherits authentication from the gh CLI:

1. Checks `GH_TOKEN` environment variable first
2. Falls back to `GITHUB_TOKEN` environment variable
3. Falls back to stored OAuth tokens from `gh auth login`
4. Token is automatically added to requests matching the configured host
5. If `ClientOptions.Host` differs from the request host, token is NOT added (security)

The host is resolved from:
1. `GH_HOST` environment variable
2. `gh` CLI configuration (typically "github.com")

### Executing gh Commands

```go
import gh "github.com/cli/go-gh/v2"

// Non-interactive: captures stdout/stderr
stdout, stderr, err := gh.Exec("pr", "view", "123", "--json", "number,title")

// Interactive: connects to parent process stdin/stdout/stderr
err := gh.ExecInteractive(ctx, "pr", "review", "123")
```

### Error Handling

```go
import "github.com/cli/go-gh/v2/pkg/api"

// REST errors
err := client.Get("repos/owner/repo/pulls/999", &result)
if err != nil {
    var httpErr *api.HTTPError
    if errors.As(err, &httpErr) {
        fmt.Printf("Status: %d\n", httpErr.StatusCode)
        fmt.Printf("Message: %s\n", httpErr.Message)
        for _, e := range httpErr.Errors {
            fmt.Printf("  %s: %s\n", e.Field, e.Message)
        }
    }
}

// GraphQL errors
err := client.Do(query, variables, &result)
if err != nil {
    var gqlErr *api.GraphQLError
    if errors.As(err, &gqlErr) {
        for _, e := range gqlErr.Errors {
            fmt.Printf("Message: %s\n", e.Message)
            fmt.Printf("Type: %s\n", e.Type)
            fmt.Printf("Path: %v\n", e.Path)
        }
        // Check for specific error types
        if gqlErr.Match("NOT_FOUND", "repository.") {
            fmt.Println("Repository not found")
        }
    }
}
```

---

## 11. Rate Limiting

### GraphQL API

| Limit Type    | Value                    | Notes                                        |
|---------------|--------------------------|----------------------------------------------|
| Primary       | 5,000 points/hour        | Per authenticated user/app                   |
| Per-minute    | 2,000 points/minute      | Burst limit                                  |
| Cost model    | Points-based             | Each field has a computational cost           |
| Minimum cost  | 1 point per query        | Even simple queries cost at least 1 point     |

GraphQL rate limiting is points-based, not request-based. A complex query with many
nested connections costs more points than a simple one. You can check your rate limit:

```graphql
query {
  rateLimit {
    limit
    cost
    remaining
    resetAt
    nodeCount
  }
}
```

### REST API

| Limit Type         | Value                         | Notes                                  |
|--------------------|-------------------------------|----------------------------------------|
| Unauthenticated    | 60 requests/hour              | By IP address                          |
| Authenticated      | 5,000 requests/hour           | Per user                               |
| GITHUB_TOKEN       | 1,000 requests/hour/repo      | In GitHub Actions                      |
| Enterprise Cloud   | 15,000 requests/hour/repo     | For Enterprise accounts                |

### Secondary Rate Limits

| Limit Type                | Value                          |
|---------------------------|--------------------------------|
| Content-generating (writes)| 80 requests/minute            |
| Content-generating (writes)| 500 requests/hour             |
| Concurrent requests       | 100 concurrent requests        |
| Search API                | 30 requests/minute             |
| GraphQL search            | 200 requests/minute            |

### Rate Limit Headers (REST)

```
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4999
X-RateLimit-Reset: 1609459200
X-RateLimit-Used: 1
X-RateLimit-Resource: core
```

### Best Practices

1. **Batch GraphQL queries**: Combine multiple queries into one to reduce point consumption.
2. **Use `first: N` judiciously**: Requesting more items per connection costs more points.
3. **Cache responses**: Use go-gh's built-in caching (`EnableCache: true`).
4. **Check remaining limits**: Monitor `X-RateLimit-Remaining` or `rateLimit.remaining`.
5. **Use conditional requests**: For REST, use `If-None-Match` / `If-Modified-Since` headers
   to avoid consuming rate limit for unchanged resources (304 responses are free).
6. **Prefer GraphQL over multiple REST calls**: One GraphQL query can replace several REST
   API calls, reducing overall request count.
7. **Handle 403/429 errors**: Back off and retry after the `Retry-After` header or
   `X-RateLimit-Reset` timestamp.

---

## Appendix A: Complete Example - Fetch and Resolve PR Review Threads in Go

```go
package main

import (
    "fmt"
    "log"

    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/cli/go-gh/v2/pkg/repository"
)

type ReviewThread struct {
    ID         string `json:"id"`
    IsResolved bool   `json:"isResolved"`
    Path       string `json:"path"`
    Line       int    `json:"line"`
    StartLine  int    `json:"startLine"`
    Comments   struct {
        Nodes []struct {
            ID     string `json:"id"`
            Body   string `json:"body"`
            Author struct {
                Login string `json:"login"`
            } `json:"author"`
            DiffHunk string `json:"diffHunk"`
        } `json:"nodes"`
    } `json:"comments"`
}

func fetchUnresolvedThreads(client *api.GraphQLClient, owner, repo string, prNumber int) ([]ReviewThread, error) {
    var result struct {
        Repository struct {
            PullRequest struct {
                ReviewThreads struct {
                    Nodes []ReviewThread `json:"nodes"`
                } `json:"reviewThreads"`
            } `json:"pullRequest"`
        } `json:"repository"`
    }

    query := `query($owner: String!, $repo: String!, $pr: Int!) {
        repository(owner: $owner, name: $repo) {
            pullRequest(number: $pr) {
                reviewThreads(first: 100) {
                    nodes {
                        id
                        isResolved
                        path
                        line
                        startLine
                        comments(first: 50) {
                            nodes {
                                id
                                body
                                author { login }
                                diffHunk
                            }
                        }
                    }
                }
            }
        }
    }`

    variables := map[string]interface{}{
        "owner": owner,
        "repo":  repo,
        "pr":    prNumber,
    }

    if err := client.Do(query, variables, &result); err != nil {
        return nil, err
    }

    var unresolved []ReviewThread
    for _, thread := range result.Repository.PullRequest.ReviewThreads.Nodes {
        if !thread.IsResolved {
            unresolved = append(unresolved, thread)
        }
    }
    return unresolved, nil
}

func resolveThread(client *api.GraphQLClient, threadID string) error {
    var result struct {
        ResolveReviewThread struct {
            Thread struct {
                ID         string `json:"id"`
                IsResolved bool   `json:"isResolved"`
            } `json:"thread"`
        } `json:"resolveReviewThread"`
    }

    mutation := `mutation($threadId: ID!) {
        resolveReviewThread(input: {threadId: $threadId}) {
            thread { id isResolved }
        }
    }`

    variables := map[string]interface{}{
        "threadId": threadID,
    }

    return client.Do(mutation, variables, &result)
}

func main() {
    repo, err := repository.Current()
    if err != nil {
        log.Fatal(err)
    }

    client, err := api.DefaultGraphQLClient()
    if err != nil {
        log.Fatal(err)
    }

    threads, err := fetchUnresolvedThreads(client, repo.Owner, repo.Name, 123)
    if err != nil {
        log.Fatal(err)
    }

    for _, thread := range threads {
        fmt.Printf("Unresolved: %s:%d - %s\n",
            thread.Path, thread.Line,
            thread.Comments.Nodes[0].Body)

        if err := resolveThread(client, thread.ID); err != nil {
            log.Printf("Failed to resolve %s: %v", thread.ID, err)
        }
    }
}
```

---

## Appendix B: Complete Example - Get Failing Check Annotations in Go

```go
package main

import (
    "fmt"
    "log"

    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/cli/go-gh/v2/pkg/repository"
)

type CheckRun struct {
    ID         int    `json:"id"`
    Name       string `json:"name"`
    Status     string `json:"status"`
    Conclusion string `json:"conclusion"`
}

type Annotation struct {
    Path            string `json:"path"`
    StartLine       int    `json:"start_line"`
    EndLine         int    `json:"end_line"`
    AnnotationLevel string `json:"annotation_level"`
    Message         string `json:"message"`
    Title           string `json:"title"`
}

func getFailingAnnotations(restClient *api.RESTClient, owner, repo, sha string) ([]Annotation, error) {
    // Step 1: Get all check runs for the commit
    var checkRunsResp struct {
        TotalCount int        `json:"total_count"`
        CheckRuns  []CheckRun `json:"check_runs"`
    }

    err := restClient.Get(
        fmt.Sprintf("repos/%s/%s/commits/%s/check-runs", owner, repo, sha),
        &checkRunsResp,
    )
    if err != nil {
        return nil, err
    }

    // Step 2: Collect annotations from failing check runs
    var allAnnotations []Annotation
    for _, run := range checkRunsResp.CheckRuns {
        if run.Conclusion != "failure" {
            continue
        }

        var annotations []Annotation
        err := restClient.Get(
            fmt.Sprintf("repos/%s/%s/check-runs/%d/annotations", owner, repo, run.ID),
            &annotations,
        )
        if err != nil {
            log.Printf("Warning: failed to get annotations for %s: %v", run.Name, err)
            continue
        }

        allAnnotations = append(allAnnotations, annotations...)
    }

    return allAnnotations, nil
}

func main() {
    repo, err := repository.Current()
    if err != nil {
        log.Fatal(err)
    }

    restClient, err := api.DefaultRESTClient()
    if err != nil {
        log.Fatal(err)
    }

    sha := "abc123" // Replace with actual HEAD SHA
    annotations, err := getFailingAnnotations(restClient, repo.Owner, repo.Name, sha)
    if err != nil {
        log.Fatal(err)
    }

    for _, a := range annotations {
        fmt.Printf("%s:%d [%s] %s\n", a.Path, a.StartLine, a.AnnotationLevel, a.Message)
    }
}
```

---

## Appendix C: Node IDs and the node() Query

All GraphQL objects in GitHub have a global Node ID (e.g., `PRRT_kwDOxxxxxx` for review threads).
You can fetch any object by its Node ID using the `node` query:

```graphql
query($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewThread {
      id
      isResolved
      path
      line
      comments(first: 10) {
        nodes { body }
      }
    }
  }
}
```

For batch lookups, use `nodes`:

```graphql
query($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on PullRequestReviewThread {
      id
      isResolved
    }
  }
}
```

---

## Sources

### GitHub Official Documentation
- [GraphQL Mutations Reference](https://docs.github.com/en/graphql/reference/mutations)
- [GraphQL Objects - PullRequestReviewThread](https://docs.github.com/en/graphql/reference/objects#pullrequestreviewthread)
- [GraphQL Objects - PullRequestReviewComment](https://docs.github.com/en/graphql/reference/objects#pullrequestreviewcomment)
- [GraphQL Objects - PullRequestReview](https://docs.github.com/en/graphql/reference/objects#pullrequestreview)
- [GraphQL Queries Reference](https://docs.github.com/en/graphql/reference/queries)
- [GraphQL Input Objects Reference](https://docs.github.com/en/graphql/reference/input-objects)
- [REST API - Pull Request Comments](https://docs.github.com/en/rest/pulls/comments)
- [REST API - Pull Request Reviews](https://docs.github.com/en/rest/pulls/reviews)
- [REST API - Check Runs](https://docs.github.com/en/rest/checks/runs)
- [REST API - Workflow Runs](https://docs.github.com/en/rest/actions/workflow-runs)
- [REST API - Workflow Jobs](https://docs.github.com/en/rest/actions/workflow-jobs)
- [REST API - Interacting with Checks](https://docs.github.com/en/rest/guides/using-the-rest-api-to-interact-with-checks)
- [GraphQL Rate Limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api)
- [REST Rate Limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
- [gh run view Manual](https://cli.github.com/manual/gh_run_view)

### go-gh Library
- [go-gh v2 Package](https://pkg.go.dev/github.com/cli/go-gh/v2)
- [go-gh API Package](https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/api)
- [go-gh Repository Package](https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/repository)
- [go-gh Example Tests](https://github.com/cli/go-gh/blob/trunk/example_gh_test.go)
- [go-gh Repository](https://github.com/cli/go-gh)

### Community Resources
- [Bulk Resolve GitHub PR Comments](https://nesin.io/blog/bulk-resolve-github-pr-comments-api)
- [Resolve PR Comments Gist](https://gist.github.com/kieranklaassen/0c91cfaaf99ab600e79ba898918cea8a)
- [GraphQL Resolved Conversations Discussion](https://github.com/orgs/community/discussions/24854)
- [PullRequestReviewComment GraphQL Schema](https://2fd.github.io/graphdoc/github/pullrequestreviewcomment.doc.html)
- [resolveReviewThread Permissions Discussion](https://github.com/orgs/community/discussions/44650)
- [Add Review Threads to REST API Discussion](https://github.com/orgs/community/discussions/41047)
