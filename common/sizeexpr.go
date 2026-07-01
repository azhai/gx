package common

import (
	"fmt"
	"strconv"
	"strings"
)

type Operator int

const (
	OpLT Operator = iota
	OpLE
	OpGT
	OpGE
	OpEQ
)

type SizeFilter struct {
	op    Operator
	value int64
}

func (f *SizeFilter) Match(size int64) bool {
	switch f.op {
	case OpLT:
		return size < f.value
	case OpLE:
		return size <= f.value
	case OpGT:
		return size > f.value
	case OpGE:
		return size >= f.value
	case OpEQ:
		return size == f.value
	}
	return false
}

func ParseSizeExpr(expr string) (*SizeFilter, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty size expression")
	}

	op, rest := parseOperator(expr)

	numStr, unit := splitNumberUnit(rest)
	if numStr == "" {
		return nil, fmt.Errorf("invalid size expression: %q", expr)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number in size expression: %q", expr)
	}
	if num < 0 {
		return nil, fmt.Errorf("negative size not allowed: %q", expr)
	}

	multiplier := int64(1)
	switch strings.ToUpper(unit) {
	case "":
		multiplier = 1
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	default:
		return nil, fmt.Errorf("unknown size unit %q in expression: %q", unit, expr)
	}

	value := int64(num * float64(multiplier))
	return &SizeFilter{op: op, value: value}, nil
}

func parseOperator(s string) (Operator, string) {
	if strings.HasPrefix(s, "<=") {
		return OpLE, s[2:]
	}
	if strings.HasPrefix(s, ">=") {
		return OpGE, s[2:]
	}
	if strings.HasPrefix(s, "<") {
		return OpLT, s[1:]
	}
	if strings.HasPrefix(s, ">") {
		return OpGT, s[1:]
	}
	return OpEQ, s
}

func splitNumberUnit(s string) (string, string) {
	i := 0
	if i < len(s) && s[i] == '.' {
		i++
	}
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}
	return s[:i], s[i:]
}
