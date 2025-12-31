# Security Policy

## Supported Versions

**cclogs** is currently in active development. Security updates are provided for the latest release only.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

We recommend always using the most recent version available.

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in **cclogs**, please report it responsibly:

### Where to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues via:
- **GitHub Security Advisories**: Use the [Security tab](https://github.com/13rac1/cclogs/security/advisories/new) to privately report vulnerabilities

### What to Include

Please include:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Any suggested fixes (if available)

### Response Timeline

- **Initial response**: Within 48 hours
- **Status updates**: Weekly until resolved
- **Fix timeline**: Depends on severity (critical issues within 7 days, others as prioritized)

### What to Expect

- We will acknowledge receipt of your report
- We will work with you to understand and validate the issue
- We will develop and test a fix
- We will coordinate disclosure timing with you
- We will credit you in the security advisory (unless you prefer to remain anonymous)

## Test Data Notice

**IMPORTANT**: The test suite (`internal/redactor/redactor_test.go` and other test files) contains **FAKE** secrets, passwords, API keys, and other sensitive-looking data.

These test values are:
- Randomly generated or copied from public documentation
- **NOT real credentials**
- Safe to commit to version control
- Used only for validating redaction patterns

If you're reviewing the code or running static analysis tools, please note that these are intentional test fixtures, not leaked secrets.

## Security Best Practices

When using **cclogs**:

1. **Storage Security**
   - Only upload to S3 buckets you own and control
   - Enable encryption at rest (SSE-S3 or SSE-KMS)
   - Use restrictive bucket policies (block public access)
   - Enable versioning for recovery from accidental changes

2. **Configuration Security**
   - Use AWS profiles instead of static credentials when possible
   - Set restrictive permissions on config file: `chmod 600 ~/.cclogs/config.yaml`
   - Never commit credentials to version control
   - Rotate credentials regularly

3. **Redaction Verification**
   - Run `--dry-run --debug` before first upload to review redaction behavior
   - Understand that redaction is best-effort and cannot catch all patterns
   - Review the [redaction limitations](README.md#redaction-limitations) in the README

4. **Access Control**
   - Limit who has access to your backup storage
   - Use IAM policies to restrict access to specific users/roles
   - Enable CloudTrail or equivalent logging for audit trails

## Known Limitations

The redaction system has known limitations documented in [README.md](README.md#redaction-limitations):
- Cannot detect custom or proprietary secret formats
- Cannot redact secrets embedded in code logic
- Cannot catch novel API key formats not in the pattern list
- Cannot handle secrets split across multiple lines

**Always treat backup storage as sensitive and apply appropriate access controls.**
