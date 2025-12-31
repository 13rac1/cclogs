// Package redactor provides PII and secrets redaction for JSONL log files.
package redactor

import (
	"fmt"
	"sort"
	"strings"
)

// Stats tracks redaction statistics for a file or batch of files.
type Stats struct {
	OriginalBytes  int64            // Total bytes before redaction
	RedactedBytes  int64            // Total bytes after redaction
	LinesProcessed int64            // Number of lines processed
	TotalMatches   int64            // Total number of patterns matched
	ByPattern      map[string]int64 // Match count per pattern type
}

// NewStats creates a new Stats instance with initialized map.
func NewStats() *Stats {
	return &Stats{
		ByPattern: make(map[string]int64),
	}
}

// PercentReduction returns the percentage of bytes removed by redaction.
func (s *Stats) PercentReduction() float64 {
	if s.OriginalBytes == 0 {
		return 0
	}
	return float64(s.OriginalBytes-s.RedactedBytes) / float64(s.OriginalBytes) * 100
}

// Add combines another Stats into this one.
func (s *Stats) Add(other *Stats) {
	if other == nil {
		return
	}
	s.OriginalBytes += other.OriginalBytes
	s.RedactedBytes += other.RedactedBytes
	s.LinesProcessed += other.LinesProcessed
	s.TotalMatches += other.TotalMatches
	for pattern, count := range other.ByPattern {
		s.ByPattern[pattern] += count
	}
}

// String returns a human-readable summary of the stats.
func (s *Stats) String() string {
	if s.TotalMatches == 0 {
		return "no redactions"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d matches", s.TotalMatches)

	if len(s.ByPattern) > 0 {
		sb.WriteString(" (")
		// Sort patterns for deterministic output
		patterns := make([]string, 0, len(s.ByPattern))
		for p := range s.ByPattern {
			patterns = append(patterns, p)
		}
		sort.Strings(patterns)

		first := true
		for _, p := range patterns {
			if !first {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%s: %d", p, s.ByPattern[p])
			first = false
		}
		sb.WriteString(")")
	}

	return sb.String()
}

// PatternSummary returns a sorted list of pattern counts for display.
func (s *Stats) PatternSummary() []PatternCount {
	counts := make([]PatternCount, 0, len(s.ByPattern))
	for pattern, count := range s.ByPattern {
		counts = append(counts, PatternCount{Pattern: pattern, Count: count})
	}
	// Sort by count descending, then by pattern name
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Pattern < counts[j].Pattern
	})
	return counts
}

// PatternCount represents a pattern and its match count.
type PatternCount struct {
	Pattern string
	Count   int64
}
