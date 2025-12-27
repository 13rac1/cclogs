# Claude Code Log Shipper

cc-log-shipper `ccls` is be a Go CLI that discovers Claude Code JSONL logs across all local projects by default and uploads them to an S3-compatible bucket using the AWS SDK. Claude Code session transcripts are stored as *.jsonl files under ~/.claude/projects/<project-dir>/
