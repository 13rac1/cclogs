# cclogs - Claude Code Log Shipper

**cclogs** is a CLI tool that backs up your Claude Code session logs to S3-compatible storage.

Claude Code stores session transcripts as `.jsonl` files under `~/.claude/projects/`. These logs are valuable for debugging, auditing, and analysis, but they can be lost when machines are rebuilt or cleaned up. **cclogs** automatically discovers all your local Claude Code projects and safely uploads their logs to S3-compatible storage, making it easy to maintain backups across multiple machines.

## SECURITY WARNING

> **Handle With Care**
>
> Claude Code session logs contain your **complete coding conversations**, including:
> - Source code snippets and file contents
> - Terminal commands and their outputs
> - Error messages and stack traces
> - File paths revealing project structure
>
> **Where do logs go?** Logs are uploaded to **your own S3-compatible storage** that you configure. This tool does NOT send data to any third-party service - you control the destination.
>
> **Redaction is best-effort.** While cclogs redacts common PII and secrets (emails, API keys, tokens), it cannot catch everything. Novel secret formats, proprietary patterns, or sensitive business logic in code will NOT be redacted.
>
> **Your responsibilities:**
> - Only upload to storage **you own and control**
> - Enable bucket encryption at rest
> - Use restrictive bucket policies (no public access)
> - Review redaction output with `--dry-run --debug` before first upload
> - Consider who has access to your backup storage

## Features

- **Automatic discovery**: Finds all Claude Code projects and `.jsonl` logs by default
- **Automatic redaction**: Redacts PII and secrets before upload (enabled by default)
- **Multi-machine safe**: Uploads are idempotent - safely run from multiple machines against the same bucket
- **S3-compatible**: Works with AWS S3, Backblaze B2, MinIO, and other S3-compatible providers
- **Smart uploads**: Skips files that already exist remotely with the same size (saves bandwidth)
- **Project tracking**: Lists local and remote projects with JSONL counts to verify coverage
- **Configuration validation**: Built-in `doctor` command checks your setup before first use

## Installation

### Homebrew (macOS/Linux)
```bash
brew install 13rac1/tap/cclogs
```

### Go Install
```bash
go install github.com/13rac1/cclogs/cmd/cclogs@latest
```

Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

### Download Binary
Download the latest release from [GitHub Releases](https://github.com/13rac1/cclogs/releases)

## Quick Start

### 1. Initial Setup

Run `cclogs` for the first time to generate a starter configuration:

```bash
cclogs list
```

This creates `~/.cclogs/config.yaml` with template settings. Edit this file to configure your S3 bucket:

```yaml
s3:
  bucket: "your-bucket-name"        # REQUIRED: your S3 bucket
  region: "us-west-2"                # REQUIRED: AWS region
  prefix: "claude-code/"             # Optional: prefix for all files
  # For S3-compatible providers (Backblaze B2, MinIO):
  # endpoint: "https://s3.us-west-002.backblazeb2.com"
  # force_path_style: true

auth:
  profile: "default"                 # Recommended: use AWS profile
  # Or use static credentials (not recommended):
  # access_key_id: "..."
  # secret_access_key: "..."
```

### 2. Validate Configuration

Check that everything is configured correctly:

```bash
cclogs doctor
```

This validates:
- Config file is readable
- S3 bucket and region are configured
- Local projects directory exists
- S3 connectivity and permissions work

### 3. Upload Logs

Upload all local `.jsonl` logs to your S3 bucket:

```bash
cclogs upload
```

Use `--dry-run` to preview what would be uploaded:

```bash
cclogs upload --dry-run
```

### 4. Verify Uploads

List all local and remote projects with JSONL counts:

```bash
cclogs list
```

Example output:

```
Projects
+----------------------------------+-------+--------+-----------+
|             Project              | Local | Remote |  Status   |
+----------------------------------+-------+--------+-----------+
| claude-code-log-shipper          |    15 |     15 | OK        |
| my-web-app                       |     8 |      8 | OK        |
| experiments                      |     3 |      - | Local-only|
+----------------------------------+-------+--------+-----------+
```

## Commands

### `cclogs doctor`

Validates configuration and connectivity.

```bash
cclogs doctor
```

Checks:
- Configuration file is valid
- S3 bucket and region are set
- Local projects directory exists and is readable
- S3 bucket is accessible with current credentials

### `cclogs list`

Lists local and remote projects with JSONL file counts.

```bash
cclogs list              # Table output
cclogs list --json       # Machine-readable JSON output
```

Helps you verify that all projects are backed up and identify any mismatches.

### `cclogs upload`

Uploads all local `.jsonl` logs to remote storage.

```bash
cclogs upload              # Upload new/changed files (with redaction)
cclogs upload --dry-run    # Preview planned uploads
cclogs upload --no-redact  # Upload without redaction (not recommended)
```

Safe to run repeatedly:
- Automatically redacts PII and secrets before upload
- Skips files that already exist remotely with identical size
- Preserves directory structure for easy restoration
- Works correctly when run from multiple machines

## Configuration

The default config location is `~/.cclogs/config.yaml`. Override with:

```bash
cclogs --config /path/to/config.yaml list
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for detailed configuration reference and examples for different S3 providers.

## Examples

See [docs/EXAMPLES.md](docs/EXAMPLES.md) for:
- First-time setup workflow
- Regular backup workflow
- Multi-machine usage scenarios
- Troubleshooting common issues

## S3-Compatible Providers

**cclogs** works with any S3-compatible storage provider. Configuration examples:

### AWS S3

```yaml
s3:
  bucket: "my-claude-logs"
  region: "us-west-2"
auth:
  profile: "default"
```

### Backblaze B2

```yaml
s3:
  bucket: "my-claude-logs"
  region: "us-west-002"
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
auth:
  profile: "backblaze"  # Configure in ~/.aws/credentials
```

### MinIO

```yaml
s3:
  bucket: "claude-logs"
  region: "us-east-1"
  endpoint: "https://minio.example.com"
  force_path_style: true
auth:
  access_key_id: "minioadmin"
  secret_access_key: "minioadmin"
```

## Security

### Threat Model

**cclogs** assumes you are uploading to storage you own and control. The redaction system provides defense-in-depth but is not a substitute for proper access controls.

### Configuration Security

- Use AWS profiles (recommended) or set restrictive permissions on `config.yaml`:
  ```bash
  chmod 600 ~/.cclogs/config.yaml
  ```
- Never commit credentials to version control
- Enable bucket encryption at rest (SSE-S3 or SSE-KMS)
- Block public access on your bucket
- Use lifecycle policies to auto-expire old backups if desired

### Redaction Limitations

The redaction system catches common patterns but **cannot detect**:
- Custom or proprietary secret formats
- Secrets embedded in code logic
- Sensitive business information
- Novel API key formats not in the pattern list
- Secrets split across multiple lines

**Always review with `--dry-run --debug` before your first upload** to verify the redaction behavior meets your needs.

## Redaction

By default, **cclogs** automatically redacts sensitive data before uploading. Redacted values are replaced with deterministic placeholders like `<EMAIL-9f86d081>` that preserve structure while protecting privacy.

### Redacted Patterns

| Category | Types |
|----------|-------|
| **PII** | Emails, phone numbers, SSNs, credit cards, IP addresses |
| **Cloud** | AWS access keys, AWS secret keys |
| **Service Tokens** | GitHub PATs, GitLab tokens, Anthropic keys, OpenAI keys, Stripe keys, Slack tokens, npm tokens |
| **Crypto** | Ethereum private keys, PEM private key blocks |
| **Auth** | JWTs, Bearer tokens, Basic auth, URL credentials |
| **Secrets** | Environment variable secrets, hex secrets |

### Placeholder Format

Format: `<TYPE-XXXXXXXXXXXX>` where `XXXXXXXXXXXX` is the first 6 bytes (12 hex chars) of the SHA-256 hash.

- Same values always produce the same placeholder (deterministic)
- Different values produce different placeholders (correlatable for debugging)
- 6 bytes provides ~281 trillion possible values, balancing readability with collision resistance

Example:
- `user@example.com` → `<EMAIL-b4c9a289...>`
- `AKIAIOSFODNN7EXAMPLE` → `<AWS_KEY-a1b2c3d4...>`

### Disabling Redaction

Use `--no-redact` to upload without redaction (not recommended):

```bash
cclogs upload --no-redact
```

## How It Works

1. **Discovery**: Scans `~/.claude/projects/` (configurable) for immediate child directories (projects)
2. **File enumeration**: Recursively finds all `.jsonl` files within each project
3. **Key mapping**: Computes S3 keys as `<prefix>/<project-dir>/<relative-path>`
4. **Remote checking**: For each file, checks if it exists remotely with the same size
5. **Upload**: Uploads only new or changed files using AWS SDK multipart uploads

This design ensures:
- No local state database required
- Safe concurrent usage from multiple machines
- Bandwidth-efficient (only uploads what's needed)
- Directory structure preserved for easy restoration

## Contributing

Issues and pull requests welcome at [github.com/13rac1/cclogs](https://github.com/13rac1/cclogs).

## License

MIT License - see LICENSE file for details.
