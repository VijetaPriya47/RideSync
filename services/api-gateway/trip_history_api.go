package main

import (
	"context"
	"net/http"
	"strconv"

	"ride-sharing/services/api-gateway/grpc_clients"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/contracts"
)

// handleTripHistory lists MongoDB trips for the JWT subject as rider or driver.
func handleTripHistory(w http.ResponseWriter, r *http.Request, trip *grpc_clients.TripServiceClient) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sub, _, _, ok := authFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := int32(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = int32(n)
		}
	}
	resp, err := trip.Client.ListMyTrips(context.Background(), &pb.ListMyTripsRequest{
		UserId: sub,
		Limit:  limit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: resp})
}
