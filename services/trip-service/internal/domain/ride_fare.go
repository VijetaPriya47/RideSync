package domain

import (
	"ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty"`
	UserID            string                 `bson:"userID"`
	PackageSlug       string                 `bson:"packageSlug"` // ex: van, luxury, sedan, carpool
	TotalPriceInCents float64                `bson:"totalPriceInCents"`
	RequestedSeats    int32                  `bson:"requestedSeats"`
	Route             *types.OsrmApiResponse `bson:"route"`
}

func (r *RideFareModel) ToProto() *pb.RideFare {
	if r == nil {
		return nil
	}
	return &pb.RideFare{
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
		RequestedSeats:    r.RequestedSeats,
	}
}

func ToRideFaresProto(fares []*RideFareModel) []*pb.RideFare {
	var protoFares []*pb.RideFare
	for _, f := range fares {
		protoFares = append(protoFares, f.ToProto())
	}
	return protoFares
}
