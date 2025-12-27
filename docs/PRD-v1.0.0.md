## PRD: cc-log-shipper (`ccls`)

### Document control
- Product name: `cc-log-shipper`
- Executable: `ccls`
- Version: v0.1 (MVP)
- Status: Final (as of 2025-12-27)

### Summary
`ccls` is a small Golang CLI that discovers Claude Code session logs on a machine and ships them to an S3-compatible bucket, then can list local and remote “projects” with `.jsonl` counts.

### Background / problem
Claude Code stores session logs locally as newline-delimited JSON (`*.jsonl`) under a per-project directory structure rooted at `~/.claude/projects/`.  
These logs are useful for debugging, auditing, and analysis, but can be lost when machines are rebuilt, replaced, or cleaned up.

### Goals
- Automatically discover **all local projects by default** and identify all `*.jsonl` logs.
- Upload those logs to an S3-compatible bucket using the AWS SDK for Go.
- Provide `ccls list` that reports:
  - Local projects + count of local `.jsonl` files per project.
  - Remote projects + count of remote `.jsonl` objects per project.
- Support running from **multiple machines** against the same bucket/prefix without requiring shared local state.
- Use a config file at `~/.ccls/config.yaml` by default, overridable via `--config`.

### Non-goals
- No log transformation (no parsing/rewriting/format conversion).
- No viewer UI.
- No deletion of local logs or remote objects.
- No environment variable configuration surface (only config file + CLI flags).
- No “real-time” tailing/streaming; this is batch upload/list.

***

## Users & use cases

### Target users
- Developers using Claude Code who want automatic off-machine backups of session logs.
- Teams archiving logs centrally in an S3-compatible system.

### Key user stories
- As a user, running `ccls upload` uploads all `.jsonl` logs across all local Claude Code projects.
- As a user, running `ccls list` shows local and remote projects with JSONL counts so I can verify coverage.
- As a user of an S3-compatible provider (Backblaze B2, MinIO, etc.), I can set a custom endpoint and path-style behavior in config.

***

## Functional requirements

## CLI surface
### Global flags
- `--config <path>`: Path to YAML config file.
  - Default: `~/.ccls/config.yaml`

### Commands
#### `ccls list`
Lists local and remote projects with `.jsonl` counts.

**Local listing requirements**
- Local projects root default: `~/.claude/projects/` (configurable).
- A “project” is each immediate child directory under the projects root.
- For each project directory:
  - Count files ending in `.jsonl` (recommended: recursive within that project directory, but must at least count direct children; define and implement consistently).

**Remote listing requirements**
- Remote “projects” are the immediate child prefixes under `s3://<bucket>/<prefix>/`.
- For each remote project prefix:
  - Count objects ending in `.jsonl` under that prefix.
  - Must support pagination (large buckets).

**Output requirements**
- Default output: human-readable table.
- `--json` optional: machine-readable output, stable schema (see below).

#### `ccls upload`
Uploads local `.jsonl` logs to remote storage.

**Discovery**
- Scan **all projects by default** under `local.projects_root`.
- Include only files ending in `.jsonl`.

**Key mapping**
- For each local file, compute destination object key deterministically:
  - `<s3.prefix>/<project-dir>/<relative-path-under-project>`
- Preserve directory structure to support restores and debugging.

**Multi-machine behavior / idempotency**
Because `ccls` runs from multiple machines, no local state DB is used.
- Default behavior: upload is safe to run repeatedly from any machine.
- Two acceptable MVP modes (choose one; recommended is Mode B):
  - Mode A (simplest): Always PUT (overwrite) the destination key.
  - Mode B (recommended): Skip upload if remote appears identical:
    - Check remote existence + compare size (and optionally etag when meaningful).
    - If identical, skip; if missing/different, upload.

**Upload requirements**
- Use AWS SDK for Go with an uploader that supports multipart/concurrency (for performance on large files).
- Retries should be enabled (use SDK defaults unless explicit overrides are added later).

**Safety**
- `--dry-run`: show planned uploads and skips without performing writes.

#### `ccls doctor`
Validates configuration and connectivity.

Checks:
- Local: projects root exists and is readable.
- Remote: can list the configured bucket/prefix (or perform an equivalent lightweight request).
- Reports actionable errors (missing config keys, auth failure, endpoint unreachable, permission denied, etc.).

---

## Configuration

### Config file
- Default path: `~/.ccls/config.yaml`
- YAML format.
- No environment variable support as part of product UX.

### Proposed schema (v0.1)
```yaml
local:
  projects_root: "/Users/me/.claude/projects"   # default if omitted: ~/.claude/projects

s3:
  bucket: "my-bucket"                          # required
  prefix: "claude-code/"                       # optional; default: "claude-code/"
  region: "us-west-002"                        # required for most SDK setups
  endpoint: "https://s3.us-west-002.backblazeb2.com"  # optional; needed for S3-compatible providers
  force_path_style: true                       # optional; default false

auth:
  profile: "default"                           # optional: use shared AWS config/credentials profile
  access_key_id: ""                            # optional (only if you decide to support static keys)
  secret_access_key: ""                        # optional
  session_token: ""                            # optional
```

### Config validation rules
- Must fail fast with a clear message if `s3.bucket` is missing for remote operations.
- Must fail fast if auth is not configured in any supported way.
- Should normalize `prefix` (ensure trailing `/` for consistent key building).

### Credentials approach (MVP decision)
To keep the tool simple and consistent across machines:
- Prefer `auth.profile` + the standard shared AWS config/credentials files on each machine.
- Optionally allow static keys in config for S3-compatible providers (but document security implications and recommend restrictive file permissions).

***

## Data & naming

### Definitions
- **Local project**: immediate child directory under `~/.claude/projects/` (or configured root).
- **Remote project**: immediate child prefix under `s3://bucket/prefix/`.
- **Log file**: any file ending with `.jsonl` under a local project directory.

### Remote layout
- Base: `s3://<bucket>/<prefix>/`
- Keys:
  - `s3://<bucket>/<prefix>/<project-dir>/.../*.jsonl`

***

## Output specs

### `ccls list` (table output)
Columns (suggested):
- Project
- Local JSONL count
- Remote JSONL count
- Status (e.g., `OK`, `Local-only`, `Remote-only`, `Mismatch`)

### `ccls list --json` schema
```json
{
  "generatedAt": "RFC3339 timestamp",
  "config": {
    "bucket": "…",
    "prefix": "…",
    "endpoint": "…"
  },
  "localProjects": [
    { "name": "…", "path": "…", "jsonlCount": 123 }
  ],
  "remoteProjects": [
    { "name": "…", "prefix": "…", "jsonlCount": 120 }
  ]
}
```

***

## Non-functional requirements

### Security & privacy
- Logs may contain sensitive content.
- Never print log contents.
- Never print secrets (redact keys if present in config).
- Recommend users set restrictive permissions on `~/.ccls/config.yaml`.

### Reliability
- `list` and `upload` must handle:
  - Missing local root.
  - Permission errors.
  - Remote pagination for large listings.
  - Transient network failures (retries).

### Performance
- Upload should use concurrent/multipart uploading for acceptable throughput.
- Counting remote objects should be streaming/paginated (avoid loading all keys into memory).

### Compatibility
- Target OS for MVP: macOS and Linux.
- S3-compatible support must include custom endpoint and optional path-style addressing.

***

## Acceptance criteria (v0.1)
- With logs present locally, `ccls list` shows all local projects and correct `.jsonl` counts.
- With valid remote config, `ccls list` also shows remote projects and correct remote `.jsonl` counts.
- `ccls upload` uploads all local `.jsonl` logs to the configured bucket/prefix.
- Re-running `ccls upload` is safe from multiple machines:
  - Either overwrites deterministically (Mode A), or
  - Skips identical objects based on remote checks (Mode B).
- `--config <path>` loads an alternate config and all commands honor it.
- `ccls doctor` clearly diagnoses missing config, local discovery issues, and remote connectivity/auth problems.

***

## Open questions (optional for MVP finalization)
- Should `upload` default to Mode A (always overwrite) for simplicity, or Mode B (skip if identical) to reduce bandwidth/cost?
  - Mode B
- Should project naming remain the raw directory name under `~/.claude/projects/`, or should `ccls` optionally maintain a human-friendly mapping file for readability?
  - Leave existing names

