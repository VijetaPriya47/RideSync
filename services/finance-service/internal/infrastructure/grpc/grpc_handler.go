package grpc

import (
	"context"
	"time"

	"ride-sharing/services/finance-service/internal/domain"
	pb "ride-sharing/shared/proto/finance"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedFinanceServiceServer

	service domain.FinanceService
}

// NewGRPCHandler registers FinanceService on the gRPC server.
func NewGRPCHandler(server *grpc.Server, svc domain.FinanceService) *gRPCHandler {
	h := &gRPCHandler{service: svc}
	pb.RegisterFinanceServiceServer(server, h)
	return h
}

func parseOptionalRange(fromRFC, toRFC string) (from, to *time.Time, err error) {
	if fromRFC != "" {
		t, e := time.Parse(time.RFC3339, fromRFC)
		if e != nil {
			return nil, nil, e
		}
		from = &t
	}
	if toRFC != "" {
		t, e := time.Parse(time.RFC3339, toRFC)
		if e != nil {
			return nil, nil, e
		}
		to = &t
	}
	return from, to, nil
}

func (h *gRPCHandler) GetMyTransactions(ctx context.Context, req *pb.GetMyTransactionsRequest) (*pb.GetMyTransactionsResponse, error) {
	uid := req.GetUserId()
	if uid == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}
	list, err := h.service.ListByUser(ctx, uid, req.GetLimit())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GetMyTransactionsResponse{Transactions: list}, nil
}

func (h *gRPCHandler) GetGlobalRevenue(ctx context.Context, req *pb.GetGlobalRevenueRequest) (*pb.GetGlobalRevenueResponse, error) {
	from, to, err := parseOptionalRange(req.GetFromRfc3339(), req.GetToRfc3339())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date range")
	}
	total, cur, trend, err := h.service.GlobalRevenue(ctx, from, to)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GetGlobalRevenueResponse{TotalCents: total, Currency: cur, Trend: trend}, nil
}

func (h *gRPCHandler) GetRegionalAnalytics(ctx context.Context, req *pb.GetRegionalAnalyticsRequest) (*pb.GetRegionalAnalyticsResponse, error) {
	from, to, err := parseOptionalRange(req.GetFromRfc3339(), req.GetToRfc3339())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date range")
	}
	regions, cur, err := h.service.RegionalAnalytics(ctx, from, to)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GetRegionalAnalyticsResponse{Regions: regions, Currency: cur}, nil
}

func (h *gRPCHandler) GetCategoryInsights(ctx context.Context, _ *pb.GetCategoryInsightsRequest) (*pb.GetCategoryInsightsResponse, error) {
	cats, cur, err := h.service.CategoryInsights(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GetCategoryInsightsResponse{Categories: cats, Currency: cur}, nil
}
