# Configuration Reference

This document provides comprehensive documentation for all **ccls** configuration options.

## Configuration File Location

Default location: `~/.ccls/config.yaml`

Override with the `--config` flag:

```bash
ccls --config /custom/path/config.yaml list
```

## Configuration File Format

The configuration file uses YAML format with three main sections:

```yaml
local:
  # Local filesystem settings

s3:
  # S3-compatible storage settings

auth:
  # Authentication credentials
```

## Complete Configuration Reference

### Local Section

Controls local filesystem settings for discovering Claude Code projects.

```yaml
local:
  projects_root: "~/.claude/projects"
```

#### `local.projects_root`

- **Type**: String
- **Required**: No
- **Default**: `~/.claude/projects`
- **Description**: Path to the Claude Code projects directory. Each immediate child directory is treated as a project.
- **Tilde expansion**: `~` is expanded to your home directory
- **Example**: `projects_root: "/Users/username/.claude/projects"`

### S3 Section

Configuration for S3-compatible storage.

```yaml
s3:
  bucket: "my-bucket"
  region: "us-west-2"
  prefix: "claude-code/"
  endpoint: "https://s3.example.com"  # Optional
  force_path_style: true               # Optional
```

#### `s3.bucket`

- **Type**: String
- **Required**: Yes
- **Description**: Name of the S3 bucket where logs will be stored
- **Example**: `bucket: "claude-code-backups"`

#### `s3.region`

- **Type**: String
- **Required**: Yes
- **Description**: AWS region or region identifier for your S3-compatible provider
- **Examples**:
  - AWS: `us-west-2`, `eu-central-1`, `ap-southeast-1`
  - Backblaze B2: `us-west-002`, `eu-central-003`
  - MinIO: Any valid region string (often `us-east-1`)

#### `s3.prefix`

- **Type**: String
- **Required**: No
- **Default**: `claude-code/`
- **Description**: Prefix for all uploaded files. Automatically gets a trailing slash if not provided.
- **Example**: `prefix: "backups/claude/"` results in keys like `backups/claude/<project>/<file>.jsonl`
- **Note**: Set to empty string (`prefix: ""`) to upload directly to bucket root (not recommended)

#### `s3.endpoint`

- **Type**: String
- **Required**: No (required for S3-compatible providers other than AWS)
- **Default**: Empty (uses AWS S3 endpoints)
- **Description**: Custom endpoint URL for S3-compatible storage providers
- **Examples**:
  - Backblaze B2: `https://s3.us-west-002.backblazeb2.com`
  - MinIO: `https://minio.example.com`
  - DigitalOcean Spaces: `https://nyc3.digitaloceanspaces.com`
- **Note**: Must include `https://` prefix

#### `s3.force_path_style`

- **Type**: Boolean
- **Required**: No
- **Default**: `false`
- **Description**: Use path-style S3 addressing (`https://endpoint/bucket/key`) instead of virtual-hosted style (`https://bucket.endpoint/key`)
- **When to use**: Required for some S3-compatible providers like Backblaze B2 and MinIO
- **Example**: `force_path_style: true`

### Auth Section

Authentication credentials for accessing S3-compatible storage.

```yaml
auth:
  profile: "default"           # Option 1: Use AWS profile (recommended)
  # OR
  access_key_id: "..."         # Option 2: Static credentials
  secret_access_key: "..."
  session_token: "..."         # Optional for temporary credentials
```

#### `auth.profile`

- **Type**: String
- **Required**: No (but recommended)
- **Default**: Empty
- **Description**: Name of AWS profile from `~/.aws/credentials` or `~/.aws/config`
- **Recommendation**: Preferred method for managing credentials
- **Example**: `profile: "backblaze"`

**Setting up AWS profiles:**

Create or edit `~/.aws/credentials`:

```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY

[backblaze]
aws_access_key_id = YOUR_B2_KEY_ID
aws_secret_access_key = YOUR_B2_APPLICATION_KEY
```

#### `auth.access_key_id`

- **Type**: String
- **Required**: No (unless profile is not used)
- **Description**: Static access key ID
- **Security**: Not recommended - use `profile` instead
- **When to use**: Only for testing or when profile-based auth is not feasible

#### `auth.secret_access_key`

- **Type**: String
- **Required**: No (unless profile is not used)
- **Description**: Static secret access key
- **Security**: Not recommended - use `profile` instead
- **File permissions**: If using static credentials, set restrictive permissions:
  ```bash
  chmod 600 ~/.ccls/config.yaml
  ```

#### `auth.session_token`

- **Type**: String
- **Required**: No
- **Description**: Session token for temporary AWS credentials
- **When to use**: For STS temporary credentials or federated access

## Configuration Precedence

When multiple authentication methods are configured:

1. Static credentials (`access_key_id`, `secret_access_key`) take precedence
2. Profile-based credentials (`profile`) are used if static credentials are not provided
3. AWS SDK default credential chain (environment variables, instance profiles) is used as fallback

**Recommendation**: Use only one method to avoid confusion.

## S3-Compatible Provider Examples

### AWS S3

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

### Backblaze B2

Backblaze B2 requires custom endpoint and path-style addressing.

**Step 1**: Create application key in Backblaze B2 console

**Step 2**: Add to `~/.aws/credentials`:

```ini
[backblaze]
aws_access_key_id = YOUR_KEY_ID
aws_secret_access_key = YOUR_APPLICATION_KEY
```

**Step 3**: Configure ccls:

```yaml
s3:
  bucket: "my-claude-logs"
  region: "us-west-002"  # Match your B2 region
  endpoint: "https://s3.us-west-002.backblazeb2.com"
  force_path_style: true
  prefix: "claude-code/"

auth:
  profile: "backblaze"
```

**Backblaze B2 Endpoints by Region**:
- `us-west-001`: `https://s3.us-west-001.backblazeb2.com`
- `us-west-002`: `https://s3.us-west-002.backblazeb2.com`
- `us-west-004`: `https://s3.us-west-004.backblazeb2.com`
- `eu-central-003`: `https://s3.eu-central-003.backblazeb2.com`

Find your region in the Backblaze B2 bucket details.

### MinIO

```yaml
s3:
  bucket: "claude-logs"
  region: "us-east-1"  # MinIO default
  endpoint: "https://minio.example.com"
  force_path_style: true
  prefix: "claude-code/"

auth:
  access_key_id: "minioadmin"
  secret_access_key: "minioadmin"
```

For production MinIO:
- Create dedicated access key instead of using default credentials
- Use AWS profile for better security:

```ini
# ~/.aws/credentials
[minio]
aws_access_key_id = YOUR_MINIO_ACCESS_KEY
aws_secret_access_key = YOUR_MINIO_SECRET_KEY
```

```yaml
auth:
  profile: "minio"
```

### DigitalOcean Spaces

```yaml
s3:
  bucket: "my-space-name"
  region: "nyc3"
  endpoint: "https://nyc3.digitaloceanspaces.com"
  force_path_style: false  # Spaces supports virtual-hosted style
  prefix: "claude-code/"

auth:
  profile: "digitalocean"
```

Create profile in `~/.aws/credentials`:

```ini
[digitalocean]
aws_access_key_id = YOUR_SPACES_ACCESS_KEY
aws_secret_access_key = YOUR_SPACES_SECRET_KEY
```

### Wasabi

```yaml
s3:
  bucket: "my-bucket"
  region: "us-east-1"
  endpoint: "https://s3.us-east-1.wasabisys.com"
  prefix: "claude-code/"

auth:
  profile: "wasabi"
```

**Wasabi Endpoints**:
- `us-east-1`: `https://s3.us-east-1.wasabisys.com`
- `us-east-2`: `https://s3.us-east-2.wasabisys.com`
- `us-west-1`: `https://s3.us-west-1.wasabisys.com`
- `eu-central-1`: `https://s3.eu-central-1.wasabisys.com`

## Configuration Validation

Run `ccls doctor` to validate your configuration:

```bash
ccls doctor
```

This checks:
- Configuration file exists and is valid YAML
- Required fields (`bucket`, `region`) are set
- Local projects directory exists and is readable
- S3 credentials are valid
- S3 bucket is accessible

## Security Best Practices

1. **Use AWS profiles instead of static credentials**
   - Credentials are stored separately from application config
   - Easier to rotate credentials
   - Better audit trail

2. **Set restrictive file permissions**
   ```bash
   chmod 600 ~/.ccls/config.yaml
   chmod 600 ~/.aws/credentials
   ```

3. **Never commit credentials to version control**
   - Add `config.yaml` to `.gitignore` if storing it in a project
   - Use environment-specific configs

4. **Use bucket policies and IAM roles**
   - Grant minimum required permissions (ListBucket, GetObject, PutObject)
   - Use bucket policies to restrict access

5. **Enable bucket encryption**
   - Server-side encryption for data at rest
   - Consider client-side encryption for sensitive logs

6. **Set up lifecycle policies**
   - Automatically archive or delete old logs
   - Transition to cheaper storage classes

## Minimal IAM Policy

For AWS S3, ccls requires these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket",
        "s3:GetObject",
        "s3:PutObject",
        "s3:GetObjectAttributes"
      ],
      "Resource": [
        "arn:aws:s3:::your-bucket-name",
        "arn:aws:s3:::your-bucket-name/*"
      ]
    }
  ]
}
```

For Backblaze B2 and other providers, equivalent permissions are needed.

## Troubleshooting

### "config file not found"

- Default location is `~/.ccls/config.yaml`
- Run any command to auto-generate starter config
- Check tilde expansion: `~` must be at start of path

### "s3.bucket is required"

- Edit config file and set `s3.bucket`
- Remove placeholder `YOUR-BUCKET-NAME`

### "Failed to initialize S3 client"

- Check authentication credentials
- Verify `auth.profile` exists in `~/.aws/credentials`
- Check static credentials if not using profile

### "Failed to connect to S3 bucket"

- Verify bucket name is correct
- Check region matches bucket's actual region
- For S3-compatible providers, verify endpoint URL
- Check network connectivity
- Verify credentials have bucket access permissions

### "Access Denied" or 403 errors

- Check IAM permissions
- Verify bucket policy allows your credentials
- For Backblaze B2, ensure application key has access to the bucket

### Path-style vs virtual-hosted style issues

- Try setting `force_path_style: true` for S3-compatible providers
- AWS S3 works with both styles
- Backblaze B2 requires path-style (`force_path_style: true`)

## Advanced Configuration

### Custom projects directory

If your Claude Code projects are in a non-standard location:

```yaml
local:
  projects_root: "/custom/path/to/projects"
```

### Multiple machines with different local paths

Each machine can have its own config with different `projects_root`:

**Machine 1** (`/Users/alice/.ccls/config.yaml`):
```yaml
local:
  projects_root: "/Users/alice/.claude/projects"
s3:
  bucket: "shared-bucket"
  # ... same S3 config
```

**Machine 2** (`/Users/bob/.ccls/config.yaml`):
```yaml
local:
  projects_root: "/Users/bob/.claude/projects"
s3:
  bucket: "shared-bucket"
  # ... same S3 config
```

Both machines can safely upload to the same bucket. Files with identical S3 keys are deduplicated by size.

### Environment-specific configurations

Use different config files for different environments:

```bash
ccls --config ~/.ccls/config.prod.yaml upload
ccls --config ~/.ccls/config.test.yaml upload
```

## Configuration File Generation

When you run `ccls` for the first time (without an existing config file), it automatically generates a starter configuration at `~/.ccls/config.yaml` with:

- Default values for all settings
- Helpful comments explaining each option
- Examples for common S3-compatible providers

You can also manually create this file by copying the template from the repository or using this minimal example:

```yaml
local:
  projects_root: "~/.claude/projects"

s3:
  bucket: "YOUR-BUCKET-NAME"
  region: "us-west-2"
  prefix: "claude-code/"

auth:
  profile: "default"
```
