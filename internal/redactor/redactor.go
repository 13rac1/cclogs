// Package redactor provides PII and secrets redaction for JSONL log files.
// It replaces sensitive data with deterministic placeholders like <EMAIL-9f86d081>.
//
// SECURITY NOTES:
// 1. This redactor provides defense-in-depth, but is NOT a substitute for
//    proper secret management (use vaults, not logs)
// 2. Patterns may have false negatives (missed secrets) and false positives
// 3. Determined attackers can bypass using encoding, obfuscation, or novel formats
// 4. Regularly audit patterns against services like GitHub Secret Scanning
// 5. Redacted logs are NOT safe to share publicly—metadata leakage is still possible
package redactor

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// pattern represents a redaction pattern with its tag and compiled regex.
type pattern struct {
	tag string
	re  *regexp.Regexp
}

// patterns contains all compiled redaction patterns.
// Order matters: more specific patterns should come before generic ones.
var patterns = []pattern{
	// Private key blocks (multiline, must come first)
	// Prevent ReDoS by using [^-]* instead of .*? to avoid catastrophic backtracking
	{"PRIVKEY", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----[^-]*-----END [A-Z ]*PRIVATE KEY-----`)},
	{"OPENSSH_KEY", regexp.MustCompile(`(?s)-----BEGIN OPENSSH PRIVATE KEY-----[^-]*-----END OPENSSH PRIVATE KEY-----`)},
	{"PUTTY_KEY", regexp.MustCompile(`(?s)PuTTY-User-Key-File-[0-9]:.*?Private-Lines:.*`)},

	// Service tokens (case-insensitive for robustness, specific prefixes before generic patterns)
	{"GITHUB", regexp.MustCompile(`(?i)\bgh[pousr]_[A-Za-z0-9_]{36,}\b`)},
	{"GITLAB", regexp.MustCompile(`(?i)\bglpat-[A-Za-z0-9_-]{20,}\b`)},
	{"ANTHROPIC", regexp.MustCompile(`(?i)\bsk-ant-[A-Za-z0-9_-]{40,}\b`)},
	{"STRIPE", regexp.MustCompile(`(?i)\bsk_(live|test)_[A-Za-z0-9]{24,}\b`)},
	{"OPENAI", regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9]{48,}\b`)},
	{"SLACK", regexp.MustCompile(`(?i)\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"NPM", regexp.MustCompile(`(?i)\bnpm_[A-Za-z0-9]{36}\b`)},
	{"GCP_API", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"SENDGRID", regexp.MustCompile(`\bSG\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{40,}\b`)},
	{"TWILIO_SID", regexp.MustCompile(`(?i)\b(AC|SK)[a-z0-9]{32}\b`)},
	{"DIGITALOCEAN", regexp.MustCompile(`(?i)\bdop_v1_[a-f0-9]{64}\b`)},
	{"DOCKER_PAT", regexp.MustCompile(`(?i)\bdckr_pat_[A-Za-z0-9_-]{32,}\b`)},
	{"CLOUDFLARE", regexp.MustCompile(`(?i)\bv1\.0-[a-f0-9]{8}-[a-f0-9]{113}\b`)},
	{"HEROKU", regexp.MustCompile(`(?i)\b[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\b`)},

	// AWS patterns (case-insensitive)
	{"AWS_KEY", regexp.MustCompile(`(?i)\bAKIA[0-9A-Z]{16}\b`)},
	{"AWS_SECRET", regexp.MustCompile(`(?i)(aws_secret_access_key|secret_access_key)["'\s:=]+[A-Za-z0-9/+=]{40}`)},

	// Azure patterns
	{"AZURE_KEY", regexp.MustCompile(`\b[A-Za-z0-9+/]{88}==\b`)},

	// Database connection strings (before URL_CREDS to catch specific formats)
	{"MONGO_URL", regexp.MustCompile(`(?i)mongodb(\+srv)?://[^:\s]+:[^@\s]+@[^\s]+`)},
	{"REDIS_URL", regexp.MustCompile(`(?i)redis[s]?://[^:\s]+:[^@\s]+@[^\s]+`)},

	// Crypto patterns (labeled keys first, then unlabeled catch-all)
	{"ETH_KEY", regexp.MustCompile(`(?i)(private.?key|eth.?key|wallet.?key)["'\s:=]+(0x)?[a-fA-F0-9]{64}`)},
	{"HEX_KEY", regexp.MustCompile(`\b(0x)?[a-fA-F0-9]{64}\b`)},

	// Auth patterns (case-insensitive, flexible formats)
	{"JWT", regexp.MustCompile(`\bey[A-Za-z0-9_-]{10,}\.ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`)},
	{"BEARER", regexp.MustCompile(`(?i)\bBearer[\s:]+[A-Za-z0-9_.-]{20,}`)},
	{"AUTH_TOKEN", regexp.MustCompile(`(?i)(authorization|token|auth)["'\s:=]+[A-Za-z0-9_.-]{32,}`)},
	{"BASIC_AUTH", regexp.MustCompile(`(?i)\bBasic\s+[A-Za-z0-9+/=]{10,}`)},

	// URL credentials (before email to avoid email matching domain parts)
	{"URL_CREDS", regexp.MustCompile(`([a-z]+://|^)[^/:@\s]+:[^/@\s]+@[^/\s]+`)},
	{"SSH_URL", regexp.MustCompile(`[a-zA-Z0-9_-]+@[a-zA-Z0-9.-]+:[a-zA-Z0-9/_-]+\.git`)},

	// PII patterns
	{"EMAIL", regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`)},
	{"SSN", regexp.MustCompile(`\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b`)},
	{"CC", regexp.MustCompile(`\b\d{4}[-\s]\d{4}[-\s]\d{4}[-\s]\d{4}\b`)},
	{"IP", regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)},
	{"PHONE_US", regexp.MustCompile(`\b(\+1[-.\s]?)?\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}\b`)},
	{"PHONE_INTL", regexp.MustCompile(`\+\d{1,3}[-.\s]?\(?\d{1,4}\)?[-.\s]?\d{1,4}[-.\s]?\d{1,9}`)},

	// Generic secret patterns (last, as catch-all)
	{"ENV_SECRET", regexp.MustCompile(`(?i)\b(password|secret|api_key)\s*[=:]\s*["']?[^\s"']{8,}`)},
	{"HEX_SECRET", regexp.MustCompile(`(?i)\b(key|secret)\s*[=:]\s*["']?[a-f0-9]{32,}`)},
	{"BASE64_SECRET", regexp.MustCompile(`\b[A-Za-z0-9/+=]{40}\b`)},
}

// placeholder generates a deterministic placeholder for a redacted value.
// Format: <TAG-XXXXXXXXXXXXXXXXXXXXXXXX> where X is the first 12 bytes (96 bits) of SHA-256 hash.
// Increased from 4 bytes to 12 bytes to prevent rainbow table attacks.
func placeholder(tag, original string) string {
	hash := sha256.Sum256([]byte(original))
	return fmt.Sprintf("<%s-%x>", tag, hash[:12])
}

// preDecodeAndRedact attempts to detect and decode common encodings,
// then recursively redacts the decoded content to catch encoded secrets.
func preDecodeAndRedact(s string) string {
	// Pattern for potential base64 (40+ chars to reduce false positives)
	base64Pattern := regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`)

	s = base64Pattern.ReplaceAllStringFunc(s, func(m string) string {
		// Attempt base64 decode
		if decoded, err := base64.StdEncoding.DecodeString(m); err == nil {
			decodedStr := string(decoded)
			// Recursively redact the decoded content
			redacted := Redact(decodedStr)
			// If redaction changed the decoded string, a secret was found
			if redacted != decodedStr {
				return placeholder("BASE64_SECRET", m)
			}
		}
		// Also try URL-safe base64
		if decoded, err := base64.URLEncoding.DecodeString(m); err == nil {
			decodedStr := string(decoded)
			redacted := Redact(decodedStr)
			if redacted != decodedStr {
				return placeholder("BASE64_SECRET", m)
			}
		}
		return m
	})

	// Try URL decoding
	if urlDecoded, err := url.QueryUnescape(s); err == nil && urlDecoded != s {
		// Recursively redact the URL-decoded content
		redactedDecoded := Redact(urlDecoded)
		// If redaction found secrets in decoded version, return redacted version
		if redactedDecoded != urlDecoded {
			s = redactedDecoded
		}
	}

	return s
}

// Redact applies all redaction patterns to a string.
// It first normalizes Unicode and attempts to decode common encodings,
// then applies regex patterns to find and redact sensitive data.
func Redact(s string) string {
	// Normalize Unicode to canonical form to prevent homoglyph bypasses
	s = norm.NFC.String(s)

	// Pre-process for encoded secrets (but avoid infinite recursion)
	// We only decode one level deep
	if !strings.Contains(s, "<BASE64_SECRET-") {
		s = preDecodeAndRedact(s)
	}

	for _, p := range patterns {
		s = p.re.ReplaceAllStringFunc(s, func(m string) string {
			return placeholder(p.tag, m)
		})
	}
	return s
}

// RedactJSON recursively redacts all string values in parsed JSON.
// WARNING: This function modifies the input in place. The input map/slice
// will be mutated. Pass a deep copy if you need to preserve the original.
func RedactJSON(v any) any {
	switch val := v.(type) {
	case string:
		return Redact(val)
	case map[string]any:
		for k, v := range val {
			val[k] = RedactJSON(v)
		}
		return val
	case []any:
		for i, v := range val {
			val[i] = RedactJSON(v)
		}
		return val
	default:
		return v
	}
}

// redactLine processes a single JSONL line, parsing as JSON if possible.
func redactLine(line []byte) ([]byte, error) {
	if len(line) == 0 {
		return line, nil
	}

	var data any
	if err := json.Unmarshal(line, &data); err != nil {
		// Not valid JSON - redact as raw string
		return []byte(Redact(string(line))), nil
	}

	redacted := RedactJSON(data)

	// Use encoder with HTML escaping disabled to preserve <TAG-xxx> format
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(redacted); err != nil {
		return nil, err
	}
	// Remove trailing newline added by Encode
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// StreamRedact returns an io.Reader that redacts each JSONL line from r.
// It parses each line as JSON and redacts string values, falling back to
// raw string redaction for non-JSON lines.
func StreamRedact(r io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		err := streamRedact(r, pw)
		pw.CloseWithError(err)
	}()

	return pr
}

// streamRedact performs the actual redaction work, writing redacted lines to w.
func streamRedact(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Increase buffer for large lines (10MB max)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		redacted, err := redactLine(line)
		if err != nil {
			return fmt.Errorf("redacting line: %w", err)
		}

		if _, err := w.Write(redacted); err != nil {
			return fmt.Errorf("writing redacted line: %w", err)
		}

		if _, err := w.Write([]byte("\n")); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	return scanner.Err()
}
