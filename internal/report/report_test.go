package report

import (
	"strings"
	"testing"

	"feishu-codex-runner/internal/codex"
	"feishu-codex-runner/internal/model"
)

func TestTruncateLines(t *testing.T) {
	in := strings.Repeat("a\n", 10)
	out := truncateLines(in, 3)
	if !strings.Contains(out, "truncated") {
		t.Fatalf("expected truncated marker, got %s", out)
	}
}

func TestFinalTimeoutStatus(t *testing.T) {
	msg := Final(model.Task{ID: "task1"}, codex.Result{TimedOut: true}, "", "")
	if !strings.Contains(msg, "❌ 超时") {
		t.Fatalf("expected timeout status, got %s", msg)
	}
}
