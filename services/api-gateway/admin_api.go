package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"ride-sharing/services/api-gateway/grpc_clients"
	pb "ride-sharing/shared/proto/auth"
	"ride-sharing/shared/contracts"
)

func handleAdminSystemLogs(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sub, _, _, ok := authFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	limit := int32(100)
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	resp, err := auth.Client.ListAuditLogs(context.Background(), &pb.ListAuditLogsRequest{
		Limit:            limit,
		BeforeTsRfc3339: q.Get("before"),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = sub
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: resp})
}

func handleAdminRegisterBusiness(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sub, _, _, ok := authFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
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
	resp, err := auth.Client.RegisterBusiness(context.Background(), &pb.RegisterBusinessRequest{
		AdminUserId: sub,
		Email:       body.Email,
		Password:    body.Password,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, contracts.APIResponse{Data: resp.User})
}

func handleAdminRegisterAdmin(w http.ResponseWriter, r *http.Request, auth *grpc_clients.UserAuthServiceClient) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sub, _, _, ok := authFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
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
	resp, err := auth.Client.RegisterAdmin(context.Background(), &pb.RegisterAdminRequest{
		AdminUserId: sub,
		Email:       body.Email,
		Password:    body.Password,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, contracts.APIResponse{Data: resp.User})
}
