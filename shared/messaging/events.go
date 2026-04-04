package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue        = "find_available_drivers"
	DriverCmdTripRequestQueue        = "driver_cmd_trip_request"
	DriverTripResponseQueue          = "driver_trip_response"
	NotifyDriverNoDriversFoundQueue  = "notify_driver_no_drivers_found"
	NotifyDriverAssignQueue          = "notify_driver_assign"
	NotifyTripCreatedQueue           = "notify_trip_created"
	PaymentTripResponseQueue         = "payment_trip_response"
	NotifyPaymentSessionCreatedQueue = "notify_payment_session_created"
	NotifyPaymentSuccessQueue        = "payment_success"
	FinancePaymentSuccessQueue       = "finance_payment_success"
	AuditLogsQueue                   = "audit_logs"
	DeadLetterQueue                  = "dead_letter_queue"
	SearchRetryQueue                 = "search_retry_queue"
)

const (
	DriverSearchMessageTTLMs = 120_000
	SearchRetryTTLMs         = 10_000
)

type TripEventData struct {
	Trip           *pb.Trip `json:"trip"`
	TriedDriverIDs []string `json:"triedDriverIds"`
}

type DriverTripResponseData struct {
	Driver         *pbd.Driver `json:"driver"`
	TripID         string      `json:"tripID"`
	RiderID        string      `json:"riderID"`
	TriedDriverIDs []string    `json:"triedDriverIds"`
}

type PaymentEventSessionCreatedData struct {
	TripID    string  `json:"tripID"`
	SessionID string  `json:"sessionID"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

type PaymentTripResponseData struct {
	TripID   string  `json:"tripID"`
	UserID   string  `json:"userID"`
	DriverID string  `json:"driverID"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentStatusUpdateData struct {
	TripID      string `json:"tripID"`
	UserID      string `json:"userID"`
	DriverID    string `json:"driverID"`
	AmountCents int64  `json:"amountCents,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Region      string `json:"region,omitempty"`
}

// AuditLogPayload is published by the API gateway for state-changing requests (JSON body of AmqpMessage.Data).
type AuditLogPayload struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	ActorUserID string `json:"actorUserId,omitempty"`
	Role        string `json:"role,omitempty"`
	IP          string `json:"ip,omitempty"`
	TS          string `json:"ts"`
	StatusCode  int    `json:"statusCode,omitempty"`
}
