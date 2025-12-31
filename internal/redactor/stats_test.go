package redactor

import (
	"io"
	"strings"
	"testing"
)

func TestNewStats(t *testing.T) {
	s := NewStats()

	if s == nil {
		t.Fatal("NewStats returned nil")
	}

	if s.ByPattern == nil {
		t.Error("ByPattern map is nil")
	}

	if s.OriginalBytes != 0 {
		t.Errorf("OriginalBytes = %d, want 0", s.OriginalBytes)
	}
}

func TestStats_PercentReduction(t *testing.T) {
	tests := []struct {
		name     string
		original int64
		redacted int64
		want     float64
	}{
		{"zero original", 0, 0, 0},
		{"no reduction", 100, 100, 0},
		{"50% reduction", 100, 50, 50},
		{"full reduction", 100, 0, 100},
		{"small reduction", 1000, 990, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Stats{
				OriginalBytes: tt.original,
				RedactedBytes: tt.redacted,
			}
			got := s.PercentReduction()
			if got != tt.want {
				t.Errorf("PercentReduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_Add(t *testing.T) {
	s1 := &Stats{
		OriginalBytes:  100,
		RedactedBytes:  90,
		LinesProcessed: 10,
		TotalMatches:   5,
		ByPattern:      map[string]int64{"EMAIL": 3, "IP": 2},
	}

	s2 := &Stats{
		OriginalBytes:  200,
		RedactedBytes:  180,
		LinesProcessed: 20,
		TotalMatches:   7,
		ByPattern:      map[string]int64{"EMAIL": 2, "JWT": 5},
	}

	s1.Add(s2)

	if s1.OriginalBytes != 300 {
		t.Errorf("OriginalBytes = %d, want 300", s1.OriginalBytes)
	}
	if s1.RedactedBytes != 270 {
		t.Errorf("RedactedBytes = %d, want 270", s1.RedactedBytes)
	}
	if s1.LinesProcessed != 30 {
		t.Errorf("LinesProcessed = %d, want 30", s1.LinesProcessed)
	}
	if s1.TotalMatches != 12 {
		t.Errorf("TotalMatches = %d, want 12", s1.TotalMatches)
	}
	if s1.ByPattern["EMAIL"] != 5 {
		t.Errorf("ByPattern[EMAIL] = %d, want 5", s1.ByPattern["EMAIL"])
	}
	if s1.ByPattern["IP"] != 2 {
		t.Errorf("ByPattern[IP] = %d, want 2", s1.ByPattern["IP"])
	}
	if s1.ByPattern["JWT"] != 5 {
		t.Errorf("ByPattern[JWT] = %d, want 5", s1.ByPattern["JWT"])
	}
}

func TestStats_Add_Nil(t *testing.T) {
	s := NewStats()
	s.TotalMatches = 5

	// Should not panic
	s.Add(nil)

	if s.TotalMatches != 5 {
		t.Errorf("TotalMatches = %d, want 5", s.TotalMatches)
	}
}

func TestStats_String(t *testing.T) {
	tests := []struct {
		name   string
		stats  *Stats
		expect string
	}{
		{
			name:   "no matches",
			stats:  NewStats(),
			expect: "no redactions",
		},
		{
			name: "single pattern",
			stats: &Stats{
				TotalMatches: 3,
				ByPattern:    map[string]int64{"EMAIL": 3},
			},
			expect: "3 matches (EMAIL: 3)",
		},
		{
			name: "multiple patterns",
			stats: &Stats{
				TotalMatches: 5,
				ByPattern:    map[string]int64{"EMAIL": 3, "IP": 2},
			},
			expect: "5 matches (EMAIL: 3, IP: 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.String()
			if got != tt.expect {
				t.Errorf("String() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestStats_PatternSummary(t *testing.T) {
	s := &Stats{
		ByPattern: map[string]int64{
			"EMAIL":   3,
			"IP":      5,
			"AWS_KEY": 1,
		},
	}

	summary := s.PatternSummary()

	// Should be sorted by count descending
	if len(summary) != 3 {
		t.Fatalf("len(summary) = %d, want 3", len(summary))
	}

	// First should be IP (highest count)
	if summary[0].Pattern != "IP" || summary[0].Count != 5 {
		t.Errorf("summary[0] = %v, want IP:5", summary[0])
	}

	// Second should be EMAIL
	if summary[1].Pattern != "EMAIL" || summary[1].Count != 3 {
		t.Errorf("summary[1] = %v, want EMAIL:3", summary[1])
	}

	// Third should be AWS_KEY
	if summary[2].Pattern != "AWS_KEY" || summary[2].Count != 1 {
		t.Errorf("summary[2] = %v, want AWS_KEY:1", summary[2])
	}
}

func TestStreamRedactWithStats(t *testing.T) {
	input := `{"email": "test@example.com", "ip": "192.168.1.1"}
{"message": "normal text"}
{"key": "AKIAIOSFODNN7EXAMPLE"}`

	reader, statsCh := StreamRedactWithStats(strings.NewReader(input))

	// Read all output
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Wait for stats
	stats := <-statsCh

	// Verify stats
	if stats.LinesProcessed != 3 {
		t.Errorf("LinesProcessed = %d, want 3", stats.LinesProcessed)
	}

	if stats.TotalMatches == 0 {
		t.Error("TotalMatches = 0, expected some matches")
	}

	// Verify output contains placeholders
	outputStr := string(output)
	if !strings.Contains(outputStr, "<EMAIL-") {
		t.Error("Output should contain EMAIL placeholder")
	}
	if !strings.Contains(outputStr, "<IP-") {
		t.Error("Output should contain IP placeholder")
	}
	if !strings.Contains(outputStr, "<AWS_KEY-") {
		t.Error("Output should contain AWS_KEY placeholder")
	}

	// Verify pattern counts
	if stats.ByPattern["EMAIL"] != 1 {
		t.Errorf("ByPattern[EMAIL] = %d, want 1", stats.ByPattern["EMAIL"])
	}
	if stats.ByPattern["IP"] != 1 {
		t.Errorf("ByPattern[IP] = %d, want 1", stats.ByPattern["IP"])
	}
	if stats.ByPattern["AWS_KEY"] != 1 {
		t.Errorf("ByPattern[AWS_KEY] = %d, want 1", stats.ByPattern["AWS_KEY"])
	}
}

func TestStreamRedactWithStats_NoMatches(t *testing.T) {
	input := `{"message": "hello world"}
{"count": 42}`

	reader, statsCh := StreamRedactWithStats(strings.NewReader(input))

	// Read all output
	_, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Wait for stats
	stats := <-statsCh

	if stats.TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", stats.TotalMatches)
	}

	if stats.LinesProcessed != 2 {
		t.Errorf("LinesProcessed = %d, want 2", stats.LinesProcessed)
	}

	// Should still have byte counts
	if stats.OriginalBytes == 0 {
		t.Error("OriginalBytes should be > 0")
	}
}
