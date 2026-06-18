package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

type Server struct {
	service *app.Service
}

func New(service *app.Service) *Server {
	return &Server{service: service}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	parts := splitPath(r.URL.Path)
	if len(parts) == 0 || parts[0] != "api" {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}

	isRead := r.Method == http.MethodGet || r.Method == http.MethodHead
	switch {
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "tasks"):
		s.handleCreateTask(w, r, parts[2], parts[4], parts[6])
	case isRead && match(parts, "api", "tasks", "*"):
		s.handleGetTask(w, r, parts[2])
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "assets"):
		s.handleListAssets(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "assets", "*"):
		s.handleGetAsset(w, r, parts[2])
	case r.Method == http.MethodPost && match(parts, "api", "assets", "*", "approve"):
		s.handleReviewAsset(w, r, parts[2], "approve")
	case r.Method == http.MethodPost && match(parts, "api", "assets", "*", "reject"):
		s.handleReviewAsset(w, r, parts[2], "reject")
	case isRead && match(parts, "api", "assets", "*", "original"):
		s.handleAssetFile(w, r, parts[2], "original")
	case isRead && match(parts, "api", "assets", "*", "thumbnail"):
		s.handleAssetFile(w, r, parts[2], "thumbnail")
	default:
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	}
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	defer r.Body.Close()
	var req domain.CreateTaskRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.CreateTask(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	}, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_task_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	response, err := s.service.GetTask(r.Context(), taskID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	response, err := s.service.ListAssets(r.Context(), projectID, campaignID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetAsset(w http.ResponseWriter, r *http.Request, assetID string) {
	response, err := s.service.GetAsset(r.Context(), assetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleReviewAsset(w http.ResponseWriter, r *http.Request, assetID, action string) {
	response, err := s.service.ReviewAsset(r.Context(), assetID, action)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAssetFile(w http.ResponseWriter, r *http.Request, assetID, kind string) {
	path, mimeType, err := s.service.GetAssetFile(r.Context(), assetID, kind)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

func (s *Server) setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func match(parts []string, pattern ...string) bool {
	if len(parts) != len(pattern) {
		return false
	}
	for i := range pattern {
		if pattern[i] == "*" {
			continue
		}
		if parts[i] != pattern[i] {
			return false
		}
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	log.Printf("request failed: %v", err)
	writeError(w, http.StatusBadRequest, "request_failed", err.Error())
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error_code":    code,
		"error_message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
