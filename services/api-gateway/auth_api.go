package main

import (
	"context"
	"encoding/json"
	"net/http"

	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	pb "ride-sharing/shared/proto/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func handleAuthLogin(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	defer r.Body.Close()
	resp, err := auth.Client.LoginLocal(context.Background(), &pb.LoginLocalRequest{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: map[string]any{
		"token": resp.Jwt,
		"user":  resp.User,
	}})
}

func handleAuthGoogle(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		IDToken string `json:"idToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	defer r.Body.Close()
	resp, err := auth.Client.GoogleVerify(context.Background(), &pb.GoogleVerifyRequest{IdToken: body.IDToken})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.FailedPrecondition:
				writeJSONError(w, http.StatusServiceUnavailable, st.Message())
				return
			case codes.Unauthenticated:
				writeJSONError(w, http.StatusUnauthorized, st.Message())
				return
			case codes.PermissionDenied:
				writeJSONError(w, http.StatusForbidden, st.Message())
				return
			case codes.Internal:
				writeJSONError(w, http.StatusInternalServerError, "google verify failed")
				return
			default:
				writeJSONError(w, http.StatusBadGateway, st.Message())
				return
			}
		}
		writeJSONError(w, http.StatusBadGateway, "google verify failed")
		return
	}
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: map[string]any{
		"token": resp.Jwt,
		"user":  resp.User,
	}})
}

func handleAuthForgotPassword(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	defer r.Body.Close()
	_, _ = auth.Client.RequestPasswordReset(context.Background(), &pb.RequestPasswordResetRequest{Email: body.Email})
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: map[string]string{"status": "ok"}})
}

func handleAuthResetPassword(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	defer r.Body.Close()
	_, err := auth.Client.ResetPassword(context.Background(), &pb.ResetPasswordRequest{
		Token: body.Token, NewPassword: body.NewPassword,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "reset failed")
		return
	}
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: map[string]string{"status": "ok"}})
}
