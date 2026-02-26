package report

import (
	"strings"
	"testing"
)

func TestTruncateLines(t *testing.T) {
	in := strings.Repeat("a\n", 10)
	out := truncateLines(in, 3)
	if !strings.Contains(out, "truncated") {
		t.Fatalf("expected truncated marker, got %s", out)
	}
}
