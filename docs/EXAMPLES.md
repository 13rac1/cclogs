# Usage Examples

This guide provides practical, step-by-step examples for common **ccls** workflows.

## Table of Contents

- [First-Time Setup](#first-time-setup)
- [Regular Backup Workflow](#regular-backup-workflow)
- [Multi-Machine Setup](#multi-machine-setup)
- [Checking Upload Status](#checking-upload-status)
- [Common Scenarios](#common-scenarios)
- [Troubleshooting](#troubleshooting)

## First-Time Setup

Complete workflow for setting up ccls for the first time.

### Step 1: Install ccls

```bash
go install github.com/13rac1/ccls/cmd/ccls@latest
```

Verify installation:

```bash
ccls --help
```

### Step 2: Generate Starter Configuration

Run any command to create the default configuration file:

```bash
ccls list
```

**Output**:
```
Welcome to ccls!

A starter configuration file has been created at:
  /Users/username/.ccls/config.yaml

Please edit this file and configure:
  1. s3.bucket - Your S3 bucket name
  2. s3.region - Your AWS region
  3. auth.profile - Your AWS profile (or use static credentials)

For S3-compatible providers (Backblaze B2, MinIO, etc.):
  - Set s3.endpoint to your provider's endpoint URL
  - Set s3.force_path_style: true if required

After configuration, run:
  ccls doctor   # Validate configuration
  ccls list     # List local and remote projects
  ccls upload   # Upload local JSONL files
```

### Step 3: Configure S3 Credentials

**Option A: Using AWS Profile (Recommended)**

Create or edit `~/.aws/credentials`:

```bash
mkdir -p ~/.aws
nano ~/.aws/credentials
```

Add your credentials:

```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
```

For Backblaze B2, create a separate profile:

```ini
[backblaze]
aws_access_key_id = YOUR_B2_KEY_ID
aws_secret_access_key = YOUR_B2_APPLICATION_KEY
```

Set permissions:

```bash
chmod 600 ~/.aws/credentials
```

**Option B: Using Static Credentials (Not Recommended)**

Edit `~/.ccls/config.yaml` directly, but be aware this is less secure.

### Step 4: Edit Configuration File

```bash
nano ~/.ccls/config.yaml
```

**Example for AWS S3**:

```yaml
local:
  projects_root: "~/.claude/projects"

s3:
  bucket: "my-claude-logs"
  region: "us-west-2"
  prefix: "claude-code/"

auth:
  profile: "default"
```

**Example for Backblaze B2**:

```yaml
local:
  projects_root: "~/.claude/projects"

s3:
  bucket: "my-claude-logs"
  region: "us-west-002"
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
  prefix: "claude-code/"

auth:
  profile: "backblaze"
```

### Step 5: Validate Configuration

```bash
ccls doctor
```

**Expected output**:

```
ccls doctor - Configuration and connectivity check

Configuration:
  ✓ Config file loaded: /Users/username/.ccls/config.yaml
  ✓ S3 bucket configured: my-claude-logs
  ✓ S3 region configured: us-west-002
  ✓ S3 prefix configured: claude-code/

Local filesystem:
  ✓ Projects root exists: /Users/username/.claude/projects
  ✓ Projects root is readable
  ✓ Found 5 local projects with 23 JSONL files

Remote connectivity:
  ✓ S3 client initialized
  ✓ Connected to bucket: my-claude-logs (us-west-002)

All checks passed! Ready to use ccls.
```

### Step 6: Preview Upload

Use dry-run to see what will be uploaded:

```bash
ccls upload --dry-run
```

**Output**:

```
Planned uploads (dry-run mode):

Project: claude-code-log-shipper
  [UPLOAD] claude-code/claude-code-log-shipper/2024-12-27T10-15-30.jsonl (2.3 MB)
  [UPLOAD] claude-code/claude-code-log-shipper/2024-12-27T14-22-15.jsonl (1.8 MB)

Project: my-web-app
  [UPLOAD] claude-code/my-web-app/2024-12-26T09-30-00.jsonl (4.1 MB)

Summary: 3 to upload (8.2 MB), 0 to skip
```

### Step 7: Perform First Upload

```bash
ccls upload
```

**Output**:

```
[1/3] Uploading /Users/username/.claude/projects/claude-code-log-shipper/2024-12-27T10-15-30.jsonl (2.3 MB)
[2/3] Uploading /Users/username/.claude/projects/claude-code-log-shipper/2024-12-27T14-22-15.jsonl (1.8 MB)
[3/3] Uploading /Users/username/.claude/projects/my-web-app/2024-12-26T09-30-00.jsonl (4.1 MB)

Upload complete: 3 uploaded (8.2 MB), 0 skipped
```

### Step 8: Verify Upload

```bash
ccls list
```

**Output**:

```
Projects
+---------------------------+-------+--------+--------+
|          Project          | Local | Remote | Status |
+---------------------------+-------+--------+--------+
| claude-code-log-shipper   |     2 |      2 | OK     |
| my-web-app                |     1 |      1 | OK     |
+---------------------------+-------+--------+--------+
```

Setup complete! Your logs are now backed up to S3.

## Regular Backup Workflow

Daily or weekly workflow to back up new session logs.

### Quick Check

See what's available locally:

```bash
ccls list
```

This shows you which projects have new logs to upload (look for "Local-only" or "Mismatch" status).

### Upload New Logs

```bash
ccls upload
```

**Output when there are new files**:

```
[1/2] Uploading /Users/username/.claude/projects/my-web-app/2024-12-28T10-00-00.jsonl (3.2 MB)
[2/2] Skipping /Users/username/.claude/projects/claude-code-log-shipper/2024-12-27T10-15-30.jsonl (identical)

Upload complete: 1 uploaded (3.2 MB), 1 skipped
```

**Output when everything is backed up**:

```
Upload complete: 0 uploaded (0 B), 5 skipped
```

### Verify

```bash
ccls list
```

All projects should show "OK" status.

## Multi-Machine Setup

Using ccls across multiple machines with the same S3 bucket.

### Scenario

You have Claude Code on:
- Work laptop (macOS)
- Home desktop (Linux)
- Both backing up to the same S3 bucket

### Setup Each Machine

**Machine 1 (Work Laptop)**:

```bash
# Install
go install github.com/13rac1/ccls/cmd/ccls@latest

# Configure
ccls list  # Generates config
nano ~/.ccls/config.yaml
```

Config for Machine 1:

```yaml
local:
  projects_root: "/Users/alice/.claude/projects"  # macOS path

s3:
  bucket: "my-shared-claude-logs"
  region: "us-west-002"
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
  prefix: "claude-code/"

auth:
  profile: "backblaze"
```

**Machine 2 (Home Desktop)**:

```bash
# Install
go install github.com/13rac1/ccls/cmd/ccls@latest

# Configure
ccls list  # Generates config
nano ~/.ccls/config.yaml
```

Config for Machine 2:

```yaml
local:
  projects_root: "/home/alice/.claude/projects"  # Linux path

s3:
  bucket: "my-shared-claude-logs"  # Same bucket!
  region: "us-west-002"
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
  prefix: "claude-code/"

auth:
  profile: "backblaze"
```

**Key points**:
- Same `bucket`, `region`, `endpoint`, `prefix` on both machines
- Different `projects_root` (OS-specific paths)
- Same auth credentials

### How It Works

When both machines upload:

1. **Project names** are based on directory names (e.g., "my-web-app")
2. **S3 keys** are computed as: `claude-code/<project>/<file>.jsonl`
3. **Duplicate detection**: If both machines have a file with the same S3 key and size, it's skipped

**Example**:

```
Machine 1: /Users/alice/.claude/projects/my-web-app/session.jsonl
    → S3 key: claude-code/my-web-app/session.jsonl

Machine 2: /home/alice/.claude/projects/my-web-app/session.jsonl
    → S3 key: claude-code/my-web-app/session.jsonl
```

If both files are identical (same size), only one is uploaded. The second machine skips it.

### Workflow

**On Work Laptop**:

```bash
ccls upload
# Uploads work logs
```

**On Home Desktop**:

```bash
ccls upload
# Uploads home logs + skips duplicates from work
```

**View All Projects**:

```bash
ccls list
```

This shows the union of all projects from all machines.

### Handling Same Project on Different Machines

If you work on the same project on both machines:

```bash
# Machine 1
ccls list
```

**Output**:

```
Projects
+---------------------------+-------+--------+-----------+
|          Project          | Local | Remote |  Status   |
+---------------------------+-------+--------+-----------+
| my-web-app                |     5 |     10 | Mismatch  |
+---------------------------+-------+--------+-----------+
```

"Mismatch" is normal when:
- Different machines have different numbers of logs
- Remote count is sum of all machines

This is working as designed. Each machine uploads its own logs, and all are preserved remotely.

## Checking Upload Status

### View All Projects

```bash
ccls list
```

Shows table with local count, remote count, and status.

**Status meanings**:
- **OK**: Local count matches remote count
- **Local-only**: Project exists locally but not remotely
- **Remote-only**: Project exists remotely but not on this machine
- **Mismatch**: Counts differ (common with multi-machine setups)

### JSON Output for Scripting

```bash
ccls list --json
```

**Example output**:

```json
{
  "generatedAt": "2024-12-27T15:30:00Z",
  "config": {
    "bucket": "my-claude-logs",
    "prefix": "claude-code/",
    "endpoint": "https://s3.us-west-002.backblazeb2.com"
  },
  "projects": [
    {
      "name": "my-web-app",
      "localPath": "/Users/alice/.claude/projects/my-web-app",
      "localCount": 5,
      "remotePath": "claude-code/my-web-app/",
      "remoteCount": 5
    }
  ]
}
```

Parse with `jq`:

```bash
# Count total local files
ccls list --json | jq '.projects | map(.localCount) | add'

# Find projects that need upload
ccls list --json | jq '.projects[] | select(.localCount > .remoteCount)'
```

### Dry-Run Before Upload

Always available to preview:

```bash
ccls upload --dry-run
```

## Common Scenarios

### Scenario 1: New Project Created

You start a new Claude Code project.

**Before**:

```bash
ccls list
```

```
Projects
+---------------------------+-------+--------+--------+
|          Project          | Local | Remote | Status |
+---------------------------+-------+--------+--------+
| my-web-app                |     5 |      5 | OK     |
+---------------------------+-------+--------+--------+
```

**Create new project** in Claude Code, generating some `.jsonl` logs.

**After**:

```bash
ccls list
```

```
Projects
+---------------------------+-------+--------+-----------+
|          Project          | Local | Remote |  Status   |
+---------------------------+-------+--------+-----------+
| my-web-app                |     5 |      5 | OK        |
| new-project               |     3 |      - | Local-only|
+---------------------------+-------+--------+-----------+
```

**Upload**:

```bash
ccls upload
```

**Verify**:

```bash
ccls list
```

```
Projects
+---------------------------+-------+--------+--------+
|          Project          | Local | Remote | Status |
+---------------------------+-------+--------+--------+
| my-web-app                |     5 |      5 | OK     |
| new-project               |     3 |      3 | OK     |
+---------------------------+-------+--------+--------+
```

### Scenario 2: Deleted Local Project

You delete a project locally but it still exists remotely (backup preserved).

```bash
rm -rf ~/.claude/projects/old-project
ccls list
```

```
Projects
+---------------------------+-------+--------+------------+
|          Project          | Local | Remote |   Status   |
+---------------------------+-------+--------+------------+
| my-web-app                |     5 |      5 | OK         |
| old-project               |     - |     10 | Remote-only|
+---------------------------+-------+--------+------------+
```

This is expected. ccls never deletes remote files. Your backup is preserved.

### Scenario 3: Updating Existing Log File

Claude Code sometimes appends to existing `.jsonl` files.

**Before**:

```
session-2024-12-27.jsonl  (2.5 MB, uploaded)
```

**After working in Claude Code**:

```
session-2024-12-27.jsonl  (3.1 MB, modified)
```

**Upload**:

```bash
ccls upload
```

Output:

```
[1/1] Uploading .../session-2024-12-27.jsonl (3.1 MB)

Upload complete: 1 uploaded (3.1 MB), 0 skipped
```

The file is re-uploaded because the size changed. The remote version is overwritten with the new content.

### Scenario 4: Checking Configuration

Verify your current configuration:

```bash
ccls doctor
```

This is useful when:
- Switching S3 providers
- Troubleshooting upload failures
- Validating credentials after rotation

## Troubleshooting

### Problem: "No files to upload" but I have logs

**Diagnosis**:

```bash
# Check local projects
ls -la ~/.claude/projects/

# Verify config
ccls doctor
```

**Possible causes**:

1. **Wrong projects_root**: Check `local.projects_root` in config
2. **No .jsonl files**: Verify files end with `.jsonl` extension
3. **All files already uploaded**: This is normal - run `ccls list` to verify

**Solution**:

```bash
# List local directory
ls ~/.claude/projects/

# Check if files are .jsonl
ls ~/.claude/projects/*/

# Verify what ccls sees
ccls list
```

### Problem: "Access Denied" when uploading

**Diagnosis**:

```bash
ccls doctor
```

Look for S3 connectivity errors.

**Possible causes**:

1. **Invalid credentials**: Check `~/.aws/credentials`
2. **Wrong bucket permissions**: Verify IAM policy or bucket policy
3. **Wrong region**: Check bucket's actual region matches config

**Solution**:

```bash
# Verify AWS credentials work
aws s3 ls s3://your-bucket-name --profile backblaze

# If using custom endpoint
aws s3 ls s3://your-bucket-name \
  --endpoint-url https://s3.us-west-002.backblazeb2.com \
  --profile backblaze
```

Fix credentials or permissions, then retry:

```bash
ccls doctor
ccls upload
```

### Problem: "Mismatch" status after upload

**Diagnosis**:

```bash
ccls list
```

```
Projects
+---------------------------+-------+--------+----------+
|          Project          | Local | Remote |  Status  |
+---------------------------+-------+--------+----------+
| my-web-app                |     5 |      8 | Mismatch |
+---------------------------+-------+--------+----------+
```

**This is usually normal**. Causes:

1. **Multi-machine setup**: Other machines uploaded different logs
2. **Manual S3 upload**: Files added to bucket outside ccls
3. **Local deletion**: You deleted local files but remote preserved

**Not a problem** unless:
- Remote count is less than local count
- You expect them to match

**Solution** (if local should match remote):

```bash
# Upload local files
ccls upload

# Check again
ccls list
```

If remote count is higher, it's because other machines or manual uploads added files. This is working correctly.

### Problem: Upload is slow

**Diagnosis**:

Large files take time. Check:

```bash
ccls upload --dry-run
```

Look at file sizes.

**Solutions**:

1. **Check network speed**: Large logs on slow connections take time
2. **Multipart upload works automatically**: ccls uses AWS SDK's multipart uploader for files > 5MB
3. **Run in background**: ccls runs synchronously; use `nohup` for long uploads:

```bash
nohup ccls upload > upload.log 2>&1 &
```

### Problem: Changed S3 provider, how to migrate?

**Scenario**: Moving from AWS S3 to Backblaze B2.

**Step 1**: Download from old bucket (optional):

Use AWS CLI or rclone to copy:

```bash
aws s3 sync s3://old-bucket/claude-code/ ./backup/
```

**Step 2**: Update ccls config:

```yaml
s3:
  bucket: "new-bucket"
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
  # ... other settings

auth:
  profile: "backblaze"
```

**Step 3**: Upload to new bucket:

```bash
ccls doctor  # Verify new config
ccls upload  # Upload all local files to new bucket
```

**Step 4**: Verify:

```bash
ccls list
```

All local files should now be in the new bucket.

### Problem: Want to restore logs to a new machine

**Scenario**: Set up a new machine and restore all logs from S3.

**Step 1**: Install ccls and configure:

```bash
go install github.com/13rac1/ccls/cmd/ccls@latest
ccls list  # Generate config
nano ~/.ccls/config.yaml
```

**Step 2**: See what's in S3:

```bash
ccls list
```

This shows all remote projects.

**Step 3**: Download logs manually:

ccls does not have a `download` command (by design). Use AWS CLI:

```bash
# Download all logs
aws s3 sync s3://my-bucket/claude-code/ ~/.claude/projects/ \
  --endpoint-url https://s3.us-west-002.backblazeb2.com \
  --profile backblaze

# Or download specific project
aws s3 sync s3://my-bucket/claude-code/my-web-app/ ~/.claude/projects/my-web-app/ \
  --endpoint-url https://s3.us-west-002.backblazeb2.com \
  --profile backblaze
```

**Step 4**: Verify:

```bash
ccls list
```

Local and remote counts should now match.

## Best Practices

1. **Run regularly**: Set up a cron job or scheduled task
   ```bash
   # crontab -e
   0 */6 * * * /Users/username/go/bin/ccls upload >> /var/log/ccls.log 2>&1
   ```

2. **Use dry-run first**: Preview before uploading
   ```bash
   ccls upload --dry-run
   ```

3. **Verify with doctor**: After config changes
   ```bash
   ccls doctor
   ```

4. **Check status periodically**: Ensure backups are current
   ```bash
   ccls list
   ```

5. **Use AWS profiles**: More secure than static credentials
   ```yaml
   auth:
     profile: "backblaze"
   ```

6. **Set up bucket lifecycle**: Auto-delete old logs or archive to cheaper storage

7. **Monitor S3 costs**: Large log volumes can incur storage costs

8. **Use --json for automation**: Parse output in scripts
   ```bash
   ccls list --json | jq '.projects[] | select(.localCount > .remoteCount)'
   ```
