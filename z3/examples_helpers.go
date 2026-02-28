package main

import (
	"fmt"
	"strconv"
	"strings"
)

type example struct {
	name string
	run  func() error
}

func parseIntExpr(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty numeral")
	}

	if strings.HasPrefix(s, "(- ") && strings.HasSuffix(s, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(s, "(- "), ")")
		v, err := parseIntExpr(inner)
		if err != nil {
			return 0, err
		}
		return -v, nil
	}

	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid rational numeral %q", raw)
		}
		num, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid rational numerator %q: %w", raw, err)
		}
		den, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid rational denominator %q: %w", raw, err)
		}
		if den == 0 {
			return 0, fmt.Errorf("division by zero in rational numeral %q", raw)
		}
		if num%den != 0 {
			return 0, fmt.Errorf("non-integer rational numeral %q", raw)
		}
		return num / den, nil
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer numeral %q: %w", raw, err)
	}
	return v, nil
}
