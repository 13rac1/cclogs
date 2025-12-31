// Package redactor provides PII and secrets redaction for JSONL log files.
// It replaces sensitive data with deterministic placeholders like <EMAIL-9f86d081>.
package redactor

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
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
	{"PRIVKEY", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},

	// Service tokens (specific prefixes, before generic patterns)
	{"GITHUB", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{36,}\b`)},
	{"GITLAB", regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}\b`)},
	{"ANTHROPIC", regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{40,}\b`)},
	{"STRIPE", regexp.MustCompile(`\bsk_(live|test)_[A-Za-z0-9]{24,}\b`)},
	{"OPENAI", regexp.MustCompile(`\bsk-[A-Za-z0-9]{48,}\b`)},
	{"SLACK", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"NPM", regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`)},

	// AWS patterns
	{"AWS_KEY", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"AWS_SECRET", regexp.MustCompile(`(?i)(aws_secret_access_key|secret_access_key)["'\s:=]+[A-Za-z0-9/+=]{40}`)},

	// Crypto patterns
	{"ETH_KEY", regexp.MustCompile(`(?i)(private.?key|eth.?key)["'\s:=]+(0x)?[a-fA-F0-9]{64}`)},

	// Auth patterns
	{"JWT", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{20,}\.eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\b`)},
	{"BEARER", regexp.MustCompile(`\bBearer\s+[A-Za-z0-9_.-]{20,}`)},
	{"BASIC_AUTH", regexp.MustCompile(`\bBasic\s+[A-Za-z0-9+/=]{10,}`)},

	// URL credentials (before email to avoid email matching domain parts)
	{"URL_CREDS", regexp.MustCompile(`://[^/:@\s]+:[^/@\s]+@[^/\s]+`)},

	// PII patterns
	{"EMAIL", regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`)},
	{"SSN", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
	{"CC", regexp.MustCompile(`\b\d{4}[-\s]\d{4}[-\s]\d{4}[-\s]\d{4}\b`)},
	{"IP", regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)},
	{"PHONE", regexp.MustCompile(`\b(\+1[-.\s]?)?\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}\b`)},

	// Generic secret patterns (last, as catch-all)
	{"ENV_SECRET", regexp.MustCompile(`(?i)\b(password|secret|api_key)\s*[=:]\s*["']?[^\s"']{8,}`)},
	{"HEX_SECRET", regexp.MustCompile(`(?i)\b(key|secret)\s*[=:]\s*["']?[a-f0-9]{32,}`)},
}

// placeholder generates a deterministic placeholder for a redacted value.
// Format: <TAG-XXXXXXXX> where XXXXXXXX is the first 4 bytes of SHA-256 hash.
func placeholder(tag, original string) string {
	hash := sha256.Sum256([]byte(original))
	return fmt.Sprintf("<%s-%x>", tag, hash[:4])
}

// Redact applies all redaction patterns to a string.
func Redact(s string) string {
	for _, p := range patterns {
		s = p.re.ReplaceAllStringFunc(s, func(m string) string {
			return placeholder(p.tag, m)
		})
	}
	return s
}

// RedactJSON recursively redacts all string values in parsed JSON.
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
		defer func() { _ = pw.Close() }()

		scanner := bufio.NewScanner(r)
		// Increase buffer for large lines (10MB max)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			redacted, err := redactLine(line)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("redacting line: %w", err))
				return
			}

			if _, err := pw.Write(redacted); err != nil {
				pw.CloseWithError(fmt.Errorf("writing redacted line: %w", err))
				return
			}

			if _, err := pw.Write([]byte("\n")); err != nil {
				pw.CloseWithError(fmt.Errorf("writing newline: %w", err))
				return
			}
		}

		if err := scanner.Err(); err != nil {
			pw.CloseWithError(fmt.Errorf("scanning input: %w", err))
		}
	}()

	return pr
}
