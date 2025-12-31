package redactor

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestPlaceholder(t *testing.T) {
	// Test determinism - same input should produce same output
	p1 := placeholder("EMAIL", "user@example.com")
	p2 := placeholder("EMAIL", "user@example.com")
	if p1 != p2 {
		t.Errorf("placeholder not deterministic: %s != %s", p1, p2)
	}

	// Test format
	if !strings.HasPrefix(p1, "<EMAIL-") || !strings.HasSuffix(p1, ">") {
		t.Errorf("unexpected placeholder format: %s", p1)
	}

	// Test that different inputs produce different outputs
	p3 := placeholder("EMAIL", "other@example.com")
	if p1 == p3 {
		t.Errorf("different inputs produced same placeholder: %s", p1)
	}
}

func TestRedactEmail(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    string
		notContains string
	}{
		{
			name:        "simple email",
			input:       "Contact me at user@example.com please",
			contains:    "<EMAIL-",
			notContains: "user@example.com",
		},
		{
			name:        "email with plus",
			input:       "user+tag@example.com",
			contains:    "<EMAIL-",
			notContains: "user+tag@example.com",
		},
		{
			name:        "multiple emails",
			input:       "From: a@b.com To: c@d.org",
			contains:    "<EMAIL-",
			notContains: "a@b.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Redact(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got: %s", tt.contains, result)
			}
			if strings.Contains(result, tt.notContains) {
				t.Errorf("expected result NOT to contain %q, got: %s", tt.notContains, result)
			}
		})
	}
}

func TestRedactPhone(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"Call 555-123-4567", true},
		{"Phone: (555) 123-4567", true},
		{"Tel: +1-555-123-4567", true},
		{"Fax: 555.123.4567", true},
		{"Not a phone: 12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<PHONE-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactSSN(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"SSN: 123-45-6789", true},
		{"Social: 987-65-4321", true},
		{"Not SSN: 12345-6789", false},
		{"Version: 1.2.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<SSN-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactCreditCard(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"Card: 4111-1111-1111-1111", true},
		{"CC: 4111 1111 1111 1111", true},
		{"Visa: 4111111111111111", false}, // No separators - intentionally doesn't match to reduce false positives
		{"Not a card: 1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<CC-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactIP(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"Server: 192.168.1.1", true},
		{"IP: 10.0.0.1", true},
		{"External: 8.8.8.8", true},
		{"Not IP: 1.2.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<IP-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactAWSKey(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", true},
		{"key: AKIAI44QH8DHBEXAMPLE", true},
		{"Not AWS: AKIA123", false}, // Too short
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<AWS_KEY-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactGitHubToken(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"ghp_1234567890abcdefghijklmnopqrstuvwxyz12", true},
		{"GITHUB_TOKEN=gho_abcdefghijklmnopqrstuvwxyz1234567890", true},
		{"Not GitHub: ghp_short", false}, // Too short
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			hasPlaceholder := strings.Contains(result, "<GITHUB-")
			if hasPlaceholder != tt.shouldMatch {
				t.Errorf("input %q: expected match=%v, got result: %s", tt.input, tt.shouldMatch, result)
			}
		})
	}
}

func TestRedactJWT(t *testing.T) {
	// Real JWT structure (header.payload.signature)
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	result := Redact("Bearer " + jwt)
	if !strings.Contains(result, "<JWT-") {
		t.Errorf("expected JWT to be redacted, got: %s", result)
	}
	if strings.Contains(result, "eyJhbGciOiJIUzI1NiI") {
		t.Errorf("JWT header still present in result: %s", result)
	}
}

func TestRedactPrivateKey(t *testing.T) {
	pemKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdef
GHIJKLMNOPQRSTUVWXYZ1234567890ab
cdefghijklmnopqrstuvwxyz12345678
-----END RSA PRIVATE KEY-----`

	result := Redact(pemKey)
	if !strings.Contains(result, "<PRIVKEY-") {
		t.Errorf("expected private key to be redacted, got: %s", result)
	}
	if strings.Contains(result, "MIIEpAIBAAKCAQEA") {
		t.Errorf("private key content still present in result: %s", result)
	}
}

func TestRedactURLCreds(t *testing.T) {
	tests := []struct {
		input        string
		shouldRedact string // What should be removed from output
	}{
		{"postgres://user:password@localhost/db", "password"},
		{"mysql://admin:secret123@db.example.com:3306/mydb", "secret123"},
		{"redis://default:mypassword@redis.example.com:6379", "mypassword"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Redact(tt.input)
			if strings.Contains(result, tt.shouldRedact) {
				t.Errorf("input %q: password %q should be redacted, got: %s", tt.input, tt.shouldRedact, result)
			}
			if !strings.Contains(result, "<URL_CREDS-") {
				t.Errorf("input %q: expected URL_CREDS placeholder, got: %s", tt.input, result)
			}
		})
	}

	// Test that URLs without credentials are not modified
	result := Redact("https://example.com/path")
	if strings.Contains(result, "<URL_CREDS-") {
		t.Error("URL without credentials should not be redacted")
	}
}

func TestRedactJSON(t *testing.T) {
	input := map[string]any{
		"email": "user@example.com",
		"nested": map[string]any{
			"phone": "555-123-4567",
		},
		"list": []any{
			"item1",
			"secret@email.com",
		},
		"number": 42,
		"bool":   true,
	}

	result := RedactJSON(input)

	// Check that the structure is preserved
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	// Check email was redacted
	email, _ := m["email"].(string)
	if !strings.Contains(email, "<EMAIL-") {
		t.Errorf("expected email to be redacted, got: %s", email)
	}

	// Check nested phone was redacted
	nested, _ := m["nested"].(map[string]any)
	phone, _ := nested["phone"].(string)
	if !strings.Contains(phone, "<PHONE-") {
		t.Errorf("expected phone to be redacted, got: %s", phone)
	}

	// Check list item was redacted
	list, _ := m["list"].([]any)
	listItem, _ := list[1].(string)
	if !strings.Contains(listItem, "<EMAIL-") {
		t.Errorf("expected list email to be redacted, got: %s", listItem)
	}

	// Check non-string values preserved
	if m["number"] != 42 {
		t.Errorf("expected number to be preserved, got: %v", m["number"])
	}
	if m["bool"] != true {
		t.Errorf("expected bool to be preserved, got: %v", m["bool"])
	}
}

func TestRedactLine(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectJSON   bool
		shouldRedact string
	}{
		{
			name:         "valid JSON with email",
			input:        `{"email":"user@example.com"}`,
			expectJSON:   true,
			shouldRedact: "user@example.com",
		},
		{
			name:         "non-JSON with email",
			input:        "Contact: user@example.com",
			expectJSON:   false,
			shouldRedact: "user@example.com",
		},
		{
			name:         "empty line",
			input:        "",
			expectJSON:   false,
			shouldRedact: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactLine([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultStr := string(result)
			if tt.shouldRedact != "" && strings.Contains(resultStr, tt.shouldRedact) {
				t.Errorf("expected %q to be redacted in result: %s", tt.shouldRedact, resultStr)
			}
		})
	}
}

func TestStreamRedact(t *testing.T) {
	input := `{"email":"user@example.com","name":"John"}
{"ip":"192.168.1.1"}
plain text with secret@email.com
`

	reader := StreamRedact(strings.NewReader(input))
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultStr := string(result)

	// Check that sensitive data is redacted
	if strings.Contains(resultStr, "user@example.com") {
		t.Error("email should be redacted")
	}
	if strings.Contains(resultStr, "192.168.1.1") {
		t.Error("IP should be redacted")
	}
	if strings.Contains(resultStr, "secret@email.com") {
		t.Error("secret email should be redacted")
	}

	// Check that placeholders are present
	if !strings.Contains(resultStr, "<EMAIL-") {
		t.Error("expected EMAIL placeholder")
	}
	if !strings.Contains(resultStr, "<IP-") {
		t.Error("expected IP placeholder")
	}

	// Check that lines are preserved
	lines := strings.Split(strings.TrimSpace(resultStr), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestStreamRedactLargeLine(t *testing.T) {
	// Create a line larger than default buffer (64KB)
	largeLine := strings.Repeat("a", 100000) + " user@example.com"
	input := largeLine + "\n"

	reader := StreamRedact(strings.NewReader(input))
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error with large line: %v", err)
	}

	if strings.Contains(string(result), "user@example.com") {
		t.Error("email should be redacted in large line")
	}
}

func TestStreamRedactEmptyInput(t *testing.T) {
	reader := StreamRedact(strings.NewReader(""))
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty output, got: %s", result)
	}
}

func TestStreamRedactPreservesNewlines(t *testing.T) {
	input := "line1\nline2\nline3\n"
	reader := StreamRedact(strings.NewReader(input))
	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := bytes.Split(result, []byte("\n"))
	// Should have 4 elements: line1, line2, line3, and empty after trailing newline
	if len(lines) != 4 {
		t.Errorf("expected 4 line segments, got %d: %q", len(lines), result)
	}
}
