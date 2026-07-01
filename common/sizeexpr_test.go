package common

import "testing"

func TestParseSizeExpr_AllOperators(t *testing.T) {
	tests := []struct {
		expr string
		op   Operator
		val  int64
	}{
		{">1M", OpGT, 1048576},
		{"<=100K", OpLE, 102400},
		{"1024", OpEQ, 1024},
		{">=2G", OpGE, 2147483648},
		{"<500", OpLT, 500},
	}
	for _, tt := range tests {
		f, err := ParseSizeExpr(tt.expr)
		if err != nil {
			t.Errorf("ParseSizeExpr(%q): %v", tt.expr, err)
			continue
		}
		if f.op != tt.op {
			t.Errorf("ParseSizeExpr(%q).op = %v, want %v", tt.expr, f.op, tt.op)
		}
		if f.value != tt.val {
			t.Errorf("ParseSizeExpr(%q).value = %d, want %d", tt.expr, f.value, tt.val)
		}
	}
}

func TestParseSizeExpr_Units(t *testing.T) {
	tests := []struct {
		expr string
		val  int64
	}{
		{"1K", 1024},
		{"1M", 1048576},
		{"1G", 1073741824},
		{"512", 512},
		{"1.5K", 1536},
	}
	for _, tt := range tests {
		f, err := ParseSizeExpr(tt.expr)
		if err != nil {
			t.Errorf("ParseSizeExpr(%q): %v", tt.expr, err)
			continue
		}
		if f.value != tt.val {
			t.Errorf("ParseSizeExpr(%q).value = %d, want %d", tt.expr, f.value, tt.val)
		}
	}
}

func TestParseSizeExpr_Errors(t *testing.T) {
	tests := []string{"", ">1X", "abc", "-5K"}
	for _, expr := range tests {
		_, err := ParseSizeExpr(expr)
		if err == nil {
			t.Errorf("ParseSizeExpr(%q): expected error", expr)
		}
	}
}

func TestSizeFilter_Match(t *testing.T) {
	f, _ := ParseSizeExpr(">1K")
	if f.Match(1024) {
		t.Error(">1K should not match 1024")
	}
	if !f.Match(1025) {
		t.Error(">1K should match 1025")
	}

	f2, _ := ParseSizeExpr("<=1M")
	if !f2.Match(1048576) {
		t.Error("<=1M should match 1048576")
	}
	if f2.Match(1048577) {
		t.Error("<=1M should not match 1048577")
	}
}
