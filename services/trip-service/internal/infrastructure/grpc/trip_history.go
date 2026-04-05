package grpc

import (
	"strings"
	"time"

	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
)

func tripModelToHistoryEntry(t *domain.TripModel, userID string) *pb.RideHistoryEntry {
	if t == nil {
		return nil
	}
	when := t.ID.Timestamp().UTC().Format(time.RFC3339)
	role := ""
	switch {
	case t.UserID == userID:
		role = "rider"
	case t.Driver != nil && t.Driver.ID == userID:
		role = "driver"
	default:
		role = "unknown"
	}
	var fare float64
	pkg := ""
	if t.RideFare != nil {
		fare = t.RideFare.TotalPriceInCents
		pkg = t.RideFare.PackageSlug
	}
	other := ""
	switch role {
	case "rider":
		if t.Driver != nil && strings.TrimSpace(t.Driver.Name) != "" {
			other = "Driver: " + t.Driver.Name
		} else {
			other = "—"
		}
	case "driver":
		other = "You drove this trip"
	default:
		other = "—"
	}
	return &pb.RideHistoryEntry{
		TripId:            t.ID.Hex(),
		Role:              role,
		Status:            t.Status,
		WhenRfc3339:       when,
		FareTotalCents:    fare,
		PackageSlug:       pkg,
		OtherPartyLabel:   other,
	}
}
