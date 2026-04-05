package repository

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ride-sharing/services/platform-service/internal/domain"
	pb "ride-sharing/shared/proto/finance"
)

// PostgresLedger implements domain.LedgerRepository.
type PostgresLedger struct {
	Pool *pgxpool.Pool
}

var _ domain.LedgerRepository = (*PostgresLedger)(nil)

// NewPostgresLedger creates a PostgreSQL-backed ledger repository.
func NewPostgresLedger(pool *pgxpool.Pool) *PostgresLedger {
	return &PostgresLedger{Pool: pool}
}

// InsertPaymentDebit idempotently records a rider payment (one row per trip_id).
func (r *PostgresLedger) InsertPaymentDebit(ctx context.Context, userID string, amountCents int64, currency, region, tripID string) error {
	if tripID == "" || userID == "" {
		log.Printf("ledger: InsertPaymentDebit skipped (empty tripID or userID); no transaction row written")
		return nil
	}
	if currency == "" {
		currency = "usd"
	}
	if region == "" {
		region = "unspecified"
	}
	id := uuid.New()
	_, err := r.Pool.Exec(ctx, `
INSERT INTO transactions (id, user_id, amount_cents, currency, type, region, status, source_trip_id)
SELECT $1, $2, $3, lower($4), 'debit', $5, 'completed', $6
WHERE NOT EXISTS (SELECT 1 FROM transactions t WHERE t.source_trip_id = $6)
`, id, userID, amountCents, currency, region, tripID)
	return err
}

func (r *PostgresLedger) ListByUser(ctx context.Context, userID string, limit int32) ([]*pb.Transaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.Pool.Query(ctx, `
SELECT id::text, user_id, amount_cents, currency, type, region, status, COALESCE(source_trip_id,''), created_at
FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*pb.Transaction
	for rows.Next() {
		var t pb.Transaction
		var created time.Time
		if err := rows.Scan(&t.Id, &t.UserId, &t.AmountCents, &t.Currency, &t.Type, &t.Region, &t.Status, &t.SourceTripId, &created); err != nil {
			return nil, err
		}
		t.CreatedAtRfc3339 = created.UTC().Format(time.RFC3339)
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *PostgresLedger) GlobalRevenue(ctx context.Context, from, to *time.Time) (total int64, currency string, trend []*pb.RevenuePoint, err error) {
	currency = "usd"
	q := `SELECT COALESCE(SUM(amount_cents),0) FROM transactions WHERE 1=1`
	args := []any{}
	if from != nil {
		args = append(args, *from)
		q += ` AND created_at >= $` + strconv.Itoa(len(args))
	}
	if to != nil {
		args = append(args, *to)
		q += ` AND created_at <= $` + strconv.Itoa(len(args))
	}
	if err = r.Pool.QueryRow(ctx, q, args...).Scan(&total); err != nil {
		return 0, "", nil, err
	}

	tq := `SELECT to_char(date_trunc('day', created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD'), COALESCE(SUM(amount_cents),0)
FROM transactions WHERE 1=1`
	targs := []any{}
	if from != nil {
		targs = append(targs, *from)
		tq += ` AND created_at >= $` + strconv.Itoa(len(targs))
	}
	if to != nil {
		targs = append(targs, *to)
		tq += ` AND created_at <= $` + strconv.Itoa(len(targs))
	}
	tq += ` GROUP BY 1 ORDER BY 1`
	trows, err := r.Pool.Query(ctx, tq, targs...)
	if err != nil {
		return total, currency, nil, err
	}
	defer trows.Close()
	for trows.Next() {
		var p pb.RevenuePoint
		if err := trows.Scan(&p.Period, &p.AmountCents); err != nil {
			return total, currency, nil, err
		}
		trend = append(trend, &p)
	}
	return total, currency, trend, trows.Err()
}

func (r *PostgresLedger) RegionalAnalytics(ctx context.Context, from, to *time.Time) ([]*pb.RegionTotal, string, error) {
	cur := "usd"
	q := `SELECT region, COALESCE(SUM(amount_cents),0), COUNT(*)::int FROM transactions WHERE 1=1`
	args := []any{}
	if from != nil {
		args = append(args, *from)
		q += ` AND created_at >= $` + strconv.Itoa(len(args))
	}
	if to != nil {
		args = append(args, *to)
		q += ` AND created_at <= $` + strconv.Itoa(len(args))
	}
	q += ` GROUP BY region ORDER BY SUM(amount_cents) DESC`
	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, cur, err
	}
	defer rows.Close()
	var out []*pb.RegionTotal
	for rows.Next() {
		var rt pb.RegionTotal
		if err := rows.Scan(&rt.Region, &rt.AmountCents, &rt.TransactionCount); err != nil {
			return nil, cur, err
		}
		out = append(out, &rt)
	}
	return out, cur, rows.Err()
}

func (r *PostgresLedger) CategoryInsights(ctx context.Context) ([]*pb.CategoryInsight, string, error) {
	cur := "usd"
	rows, err := r.Pool.Query(ctx, `
SELECT type, COALESCE(SUM(amount_cents),0), COUNT(*)::int FROM transactions GROUP BY type ORDER BY SUM(amount_cents) DESC`)
	if err != nil {
		return nil, cur, err
	}
	defer rows.Close()
	var out []*pb.CategoryInsight
	for rows.Next() {
		var c pb.CategoryInsight
		if err := rows.Scan(&c.Category, &c.AmountCents, &c.Count); err != nil {
			return nil, cur, err
		}
		out = append(out, &c)
	}
	return out, cur, rows.Err()
}
