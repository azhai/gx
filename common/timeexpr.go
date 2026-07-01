package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TimeFilter struct {
	op    Operator
	value time.Time
}

func (f *TimeFilter) Match(t time.Time) bool {
	switch f.op {
	case OpLT:
		return t.Before(f.value)
	case OpLE:
		return !t.After(f.value)
	case OpGT:
		return t.After(f.value)
	case OpGE:
		return !t.Before(f.value)
	case OpEQ:
		return t.Equal(f.value)
	}
	return false
}

func ParseTimeExpr(expr string, now time.Time) (*TimeFilter, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty time expression")
	}

	op, rest := parseOperator(expr)

	if isRelativeTime(rest) {
		d, err := parseRelativeDuration(rest)
		if err != nil {
			return nil, err
		}
		value := now.Add(-d)
		invertedOp := invertOperator(op)
		return &TimeFilter{op: invertedOp, value: value}, nil
	}

	t, err := parseAbsoluteTime(rest)
	if err != nil {
		return nil, err
	}
	return &TimeFilter{op: op, value: t}, nil
}

func isRelativeTime(s string) bool {
	if len(s) == 0 {
		return false
	}
	last := s[len(s)-1]
	switch last {
	case 's', 'm', 'h', 'd', 'w':
		return true
	}
	return false
}

func parseRelativeDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty relative time")
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in relative time: %q", s)
	}
	if num < 0 {
		return 0, fmt.Errorf("negative relative time not allowed: %q", s)
	}

	switch unit {
	case 's':
		return time.Duration(num * float64(time.Second)), nil
	case 'm':
		return time.Duration(num * float64(time.Minute)), nil
	case 'h':
		return time.Duration(num * float64(time.Hour)), nil
	case 'd':
		return time.Duration(num * 24 * float64(time.Hour)), nil
	case 'w':
		return time.Duration(num * 7 * 24 * float64(time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown time unit %q in: %q", unit, s)
	}
}

func invertOperator(op Operator) Operator {
	switch op {
	case OpLT:
		return OpGT
	case OpLE:
		return OpGE
	case OpGT:
		return OpLT
	case OpGE:
		return OpLE
	case OpEQ:
		return OpEQ
	}
	return op
}

func parseAbsoluteTime(s string) (time.Time, error) {
	if strings.Contains(s, "T") {
		t, err := time.Parse("2006-01-02T15:04:05", s)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid absolute time (expected YYYY-MM-DDTHH:MM:SS): %q", s)
		}
		return t, nil
	}

	if strings.Contains(s, "-") {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid absolute time (expected YYYY-MM-DD): %q", s)
		}
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse time expression: %q", s)
}
