package sqlite

import "strings"

type scannable interface {
	Scan(...any) error
}

func generateParameters(n int) string {
	if n == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("(?")
	for range n - 1 {
		sb.WriteString(",?")
	}

	sb.WriteString(")")
	return sb.String()
}
