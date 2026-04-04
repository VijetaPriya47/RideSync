package sqlmigrate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyFile runs a SQL file split on semicolons (sufficient for our idempotent DDL).
func ApplyFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	stmts := splitSQL(string(b))
	for _, s := range stmts {
		if s == "" {
			continue
		}
		if _, err := pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("exec: %w\n-- %s", err, truncate(s, 200))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func splitSQL(src string) []string {
	var out []string
	var cur strings.Builder
	inDollar := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '$' && i+1 < len(src) && src[i+1] == '$' {
			inDollar = !inDollar
			cur.WriteByte(c)
			continue
		}
		if c == ';' && !inDollar {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if tail := strings.TrimSpace(cur.String()); tail != "" {
		out = append(out, tail)
	}
	return out
}
