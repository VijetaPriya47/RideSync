package service

import (
	"context"
	"time"

	"ride-sharing/services/platform-service/internal/domain"
	pb "ride-sharing/shared/proto/finance"
)

type financeService struct {
	repo domain.LedgerRepository
}

var _ domain.FinanceService = (*financeService)(nil)

// NewFinanceService wires the ledger repository into the application service.
func NewFinanceService(repo domain.LedgerRepository) domain.FinanceService {
	return &financeService{repo: repo}
}

func (s *financeService) InsertPaymentDebit(ctx context.Context, userID string, amountCents int64, currency, region, tripID string) error {
	return s.repo.InsertPaymentDebit(ctx, userID, amountCents, currency, region, tripID)
}

func (s *financeService) ListByUser(ctx context.Context, userID string, limit int32) ([]*pb.Transaction, error) {
	return s.repo.ListByUser(ctx, userID, limit)
}

func (s *financeService) GlobalRevenue(ctx context.Context, from, to *time.Time) (int64, string, []*pb.RevenuePoint, error) {
	return s.repo.GlobalRevenue(ctx, from, to)
}

func (s *financeService) RegionalAnalytics(ctx context.Context, from, to *time.Time) ([]*pb.RegionTotal, string, error) {
	return s.repo.RegionalAnalytics(ctx, from, to)
}

func (s *financeService) CategoryInsights(ctx context.Context) ([]*pb.CategoryInsight, string, error) {
	return s.repo.CategoryInsights(ctx)
}
