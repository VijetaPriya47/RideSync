package domain

import (
	"context"
	"time"

	pb "ride-sharing/shared/proto/finance"
)

// LedgerRepository persists finance ledger rows and analytics queries.
type LedgerRepository interface {
	InsertPaymentDebit(ctx context.Context, userID string, amountCents int64, currency, region, tripID string) error
	ListByUser(ctx context.Context, userID string, limit int32) ([]*pb.Transaction, error)
	GlobalRevenue(ctx context.Context, from, to *time.Time) (total int64, currency string, trend []*pb.RevenuePoint, err error)
	RegionalAnalytics(ctx context.Context, from, to *time.Time) ([]*pb.RegionTotal, string, error)
	CategoryInsights(ctx context.Context) ([]*pb.CategoryInsight, string, error)
}

// FinanceService is the application API used by gRPC and AMQP consumers.
type FinanceService interface {
	InsertPaymentDebit(ctx context.Context, userID string, amountCents int64, currency, region, tripID string) error
	ListByUser(ctx context.Context, userID string, limit int32) ([]*pb.Transaction, error)
	GlobalRevenue(ctx context.Context, from, to *time.Time) (total int64, currency string, trend []*pb.RevenuePoint, err error)
	RegionalAnalytics(ctx context.Context, from, to *time.Time) ([]*pb.RegionTotal, string, error)
	CategoryInsights(ctx context.Context) ([]*pb.CategoryInsight, string, error)
}
