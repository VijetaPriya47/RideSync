package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/authjwt"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/tracing"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

var tracer = tracing.GetTracer("api-gateway")

func handleTripStart(w http.ResponseWriter, r *http.Request, tripGRPC *grpc_clients.TripServiceClient) {
	ctx, span := tracer.Start(r.Context(), "handleTripStart")
	defer span.End()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var reqBody startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	defer r.Body.Close()

	sub, _, _, ok := authFromRequest(r)
	if !applyCanonicalUserID(&reqBody.UserID, sub, ok) {
		writeJSONError(w, http.StatusForbidden, "user mismatch or missing identity")
		return
	}

	trip, err := tripGRPC.Client.CreateTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("DEBUG: gRPC CreateTrip failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start trip: %v", err))
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSON(w, http.StatusCreated, response)
}

func handleTripPreview(w http.ResponseWriter, r *http.Request, tripGRPC *grpc_clients.TripServiceClient) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview")
	defer span.End()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	defer r.Body.Close()

	sub, _, _, ok := authFromRequest(r)
	if !applyCanonicalUserID(&reqBody.UserID, sub, ok) {
		writeJSONError(w, http.StatusForbidden, "user mismatch or missing identity")
		return
	}

	tripPreview, err := tripGRPC.Client.PreviewTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("DEBUG: gRPC PreviewTrip failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to preview trip: %v", err))
		return
	}

	response := contracts.APIResponse{Data: tripPreview}

	writeJSON(w, http.StatusCreated, response)
}

func handleIncreaseTripFare(w http.ResponseWriter, r *http.Request, tripGRPC *grpc_clients.TripServiceClient) {
	ctx, span := tracer.Start(r.Context(), "handleIncreaseTripFare")
	defer span.End()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var reqBody increaseTripFareRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}
	defer r.Body.Close()

	sub, _, _, ok := authFromRequest(r)
	if !applyCanonicalUserID(&reqBody.UserID, sub, ok) {
		writeJSONError(w, http.StatusForbidden, "user mismatch or missing identity")
		return
	}

	if reqBody.TripID == "" || reqBody.TotalPriceInCents <= 0 {
		writeJSONError(w, http.StatusBadRequest, "tripID and totalPriceInCents are required")
		return
	}

	resp, err := tripGRPC.Client.IncreaseTripFare(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("IncreaseTripFare: %v", err)
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Failed to increase fare: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: resp})
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx, span := tracer.Start(r.Context(), "handleStripeWebhook")
	defer span.End()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")
	if webhookKey == "" {
		log.Printf("stripe webhook: STRIPE_WEBHOOK_KEY is not set; cannot verify or publish payment success")
		http.Error(w, "Stripe webhook not configured", http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		webhookKey,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("stripe webhook: received event type=%s id=%s", event.Type, event.ID)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		tripID := session.Metadata["trip_id"]
		userID := session.Metadata["user_id"]
		if tripID == "" || userID == "" {
			log.Printf("stripe webhook: checkout.session.completed session_id=%s missing trip_id or user_id in metadata (ledger and trip payed flow will not run). Ensure Checkout sessions are created by payment-service with trip_id, user_id, driver_id metadata — mock session IDs (cs_test_mock_*) never produce a real completed session.", session.ID)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}

		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid &&
			session.PaymentStatus != stripe.CheckoutSessionPaymentStatusNoPaymentRequired {
			log.Printf("stripe webhook: checkout.session.completed session_id=%s payment_status=%s (amount may be unset until paid)", session.ID, session.PaymentStatus)
		}

		region := session.Metadata["region"]
		if region == "" {
			region = "unspecified"
		}
		cur := string(session.Currency)
		if cur == "" {
			cur = "usd"
		}
		payload := messaging.PaymentStatusUpdateData{
			TripID:      tripID,
			UserID:      userID,
			DriverID:    session.Metadata["driver_id"],
			AmountCents: session.AmountTotal,
			Currency:    cur,
			Region:      region,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: userID,
			Data:    payloadBytes,
		}

		if err := rb.PublishMessage(
			ctx,
			contracts.PaymentEventSuccess,
			message,
		); err != nil {
			log.Printf("Error publishing payment event: %v", err)
			http.Error(w, "Failed to publish payment event", http.StatusInternalServerError)
			return
		}
		log.Printf("stripe webhook: published payment.event.success for trip_id=%s user_id=%s amount_cents=%d", tripID, userID, session.AmountTotal)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleUpdateTripSeats(w http.ResponseWriter, r *http.Request, tripGRPC *grpc_clients.TripServiceClient) {
	ctx, span := tracer.Start(r.Context(), "handleUpdateTripSeats")
	defer span.End()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var reqBody struct {
		FareID string `json:"fareID"`
		Seats  int32  `json:"seats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}
	defer r.Body.Close()

	if reqBody.FareID == "" {
		writeJSONError(w, http.StatusBadRequest, "fareID is required")
		return
	}

	_, err := tripGRPC.Client.UpdateFareSeats(ctx, &pb.UpdateFareSeatsRequest{
		FareId: reqBody.FareID,
		Seats:  reqBody.Seats,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				writeJSONError(w, http.StatusBadRequest, st.Message())
				return
			}
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	response := contracts.APIResponse{
		Error: &contracts.APIError{
			Message: message,
			Code:    fmt.Sprintf("%d", code),
		},
	}
	writeJSON(w, code, response)
}

func handleGetTripStatus(w http.ResponseWriter, r *http.Request, tripGRPC *grpc_clients.TripServiceClient) {
	ctx, span := tracer.Start(r.Context(), "handleGetTripStatus")
	defer span.End()

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		writeJSONError(w, http.StatusBadRequest, "invalid url")
		return
	}
	tripID := parts[2]

	protoResp, err := tripGRPC.Client.GetTrip(ctx, &pb.GetTripRequest{TripId: tripID})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				writeJSONError(w, http.StatusNotFound, st.Message())
				return
			case codes.InvalidArgument:
				writeJSONError(w, http.StatusBadRequest, st.Message())
				return
			}
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t := protoResp.GetTrip()
	if t == nil {
		writeJSONError(w, http.StatusNotFound, "trip not found")
		return
	}

	sub, role, _, ok := authFromRequest(r)
	if ok && role == authjwt.RoleCustomer && t.GetUserID() != "" && t.GetUserID() != sub {
		writeJSONError(w, http.StatusForbidden, "cannot access another user's trip")
		return
	}

	mo := protojson.MarshalOptions{}
	raw, err := mo.Marshal(t)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode trip")
		return
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to decode response")
		return
	}

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: data})
}

