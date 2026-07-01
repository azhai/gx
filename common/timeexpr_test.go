package common

import (
	"testing"
	"time"
)

func TestParseTimeExpr_RelativeTime(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		expr     string
		duration time.Duration
	}{
		{"<=1h", time.Hour},
		{">7d", 7 * 24 * time.Hour},
		{"30m", 30 * time.Minute},
		{">=1w", 7 * 24 * time.Hour},
		{"<60s", 60 * time.Second},
	}
	for _, tt := range tests {
		f, err := ParseTimeExpr(tt.expr, now)
		if err != nil {
			t.Errorf("ParseTimeExpr(%q): %v", tt.expr, err)
			continue
		}
		expected := now.Add(-tt.duration)
		if !f.value.Equal(expected) {
			t.Errorf("ParseTimeExpr(%q).value = %v, want %v", tt.expr, f.value, expected)
		}
	}
}

func TestParseTimeExpr_AbsoluteTime(t *testing.T) {
	now := time.Now()

	f, err := ParseTimeExpr(">=2025-01-01", now)
	if err != nil {
		t.Fatalf("ParseTimeExpr(>=2025-01-01): %v", err)
	}
	if f.op != OpGE {
		t.Errorf("op = %v, want OpGE", f.op)
	}
	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !f.value.Equal(expected) {
		t.Errorf("value = %v, want %v", f.value, expected)
	}

	f2, err := ParseTimeExpr("2025-06-15T14:30:00", now)
	if err != nil {
		t.Fatalf("ParseTimeExpr(2025-06-15T14:30:00): %v", err)
	}
	if f2.op != OpEQ {
		t.Errorf("op = %v, want OpEQ", f2.op)
	}
	expected2 := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	if !f2.value.Equal(expected2) {
		t.Errorf("value = %v, want %v", f2.value, expected2)
	}
}

func TestParseTimeExpr_Errors(t *testing.T) {
	now := time.Now()
	tests := []string{"", "<=1x", "2025-13-01", "not-a-time"}
	for _, expr := range tests {
		_, err := ParseTimeExpr(expr, now)
		if err == nil {
			t.Errorf("ParseTimeExpr(%q): expected error", expr)
		}
	}
}

func TestTimeFilter_Match(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	f, _ := ParseTimeExpr("<=1h", now)
	fileTime := now.Add(-30 * time.Minute)
	if !f.Match(fileTime) {
		t.Error("<=1h should match file modified 30min ago")
	}
	oldTime := now.Add(-2 * time.Hour)
	if f.Match(oldTime) {
		t.Error("<=1h should not match file modified 2h ago")
	}

	f2, _ := ParseTimeExpr(">7d", now)
	recentFile := now.Add(-3 * 24 * time.Hour)
	if f2.Match(recentFile) {
		t.Error(">7d should not match file modified 3 days ago")
	}
	oldFile := now.Add(-10 * 24 * time.Hour)
	if !f2.Match(oldFile) {
		t.Error(">7d should match file modified 10 days ago")
	}
}
