package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/http/middleware"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/http/response"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/observability"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/repository"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/security"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/service"
)

type UserHandler struct {
	userSvc    service.UserServiceInterface
	sessionSvc service.SessionServiceInterface
	storageSvc service.StorageService
}

func NewUserHandler(userSvc service.UserServiceInterface, sessionSvc service.SessionServiceInterface, storageSvc service.StorageService) *UserHandler {
	return &UserHandler{
		userSvc:    userSvc,
		sessionSvc: sessionSvc,
		storageSvc: storageSvc,
	}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "unauthorized")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}
	u, _, err := h.userSvc.GetByID(userID)
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "not_found")
		response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
		return
	}
	observability.RecordUserProfileEvent(r.Context(), "success")
	response.JSON(w, r, http.StatusOK, u)
}

func (h *UserHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	userID, claims, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "list", "error")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}

	currentSessionID, err := h.sessionSvc.ResolveCurrentSessionID(r, claims, userID)
	if err != nil && !errors.Is(err, repository.ErrSessionNotFound) {
		observability.RecordSessionManagementEvent(r.Context(), "list", "error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to resolve current session", nil)
		return
	}
	sessionViews, err := h.sessionSvc.ListActiveSessions(userID, currentSessionID)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "list", "error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to list sessions", nil)
		return
	}

	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "session.list",
		ActorUserID: observability.ActorUserID(userID),
		TargetType:  "session",
		TargetID:    "self",
		Action:      "list",
		Outcome:     "success",
		Reason:      "sessions_loaded",
	}, "count", len(sessionViews), "current_session_id", currentSessionID)
	observability.RecordSessionManagementEvent(r.Context(), "list", "success")
	response.JSON(w, r, http.StatusOK, sessionViews)
}

func (h *UserHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "revoke_one", "error")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}

	rawSessionID := chi.URLParam(r, "session_id")
	sessionID64, err := strconv.ParseUint(rawSessionID, 10, 64)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "revoke_one", "error")
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid session id", nil)
		return
	}
	sessionID := uint(sessionID64)

	status, err := h.sessionSvc.RevokeSession(userID, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			observability.RecordSessionManagementEvent(r.Context(), "revoke_one", "not_found")
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "session not found", nil)
			return
		}
		observability.RecordSessionManagementEvent(r.Context(), "revoke_one", "error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to revoke session", nil)
		return
	}

	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "session.revoke.single",
		ActorUserID: observability.ActorUserID(userID),
		TargetType:  "session",
		TargetID:    strconv.FormatUint(uint64(sessionID), 10),
		Action:      "revoke",
		Outcome:     "success",
		Reason:      status,
	}, "status", status)
	observability.RecordSessionManagementEvent(r.Context(), "revoke_one", "success")
	revokedCount := int64(0)
	if status == "revoked" {
		revokedCount = 1
	}
	observability.RecordSessionRevokedCount(r.Context(), "revoke_by_user", revokedCount)
	response.JSON(w, r, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"status":     status,
	})
}

func (h *UserHandler) RevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	userID, claims, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "revoke_others", "error")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}

	currentSessionID, err := h.sessionSvc.ResolveCurrentSessionID(r, claims, userID)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "revoke_others", "error")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unable to determine current session", nil)
		return
	}

	revokedCount, err := h.sessionSvc.RevokeOtherSessions(userID, currentSessionID)
	if err != nil {
		observability.RecordSessionManagementEvent(r.Context(), "revoke_others", "error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to revoke other sessions", nil)
		return
	}

	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "session.revoke.others",
		ActorUserID: observability.ActorUserID(userID),
		TargetType:  "session",
		TargetID:    "others",
		Action:      "revoke",
		Outcome:     "success",
		Reason:      "bulk_revoke",
	}, "current_session_id", currentSessionID, "revoked_count", revokedCount)
	observability.RecordSessionManagementEvent(r.Context(), "revoke_others", "success")
	observability.RecordSessionRevokedCount(r.Context(), "revoke_others", revokedCount)
	response.JSON(w, r, http.StatusOK, map[string]any{
		"current_session_id": currentSessionID,
		"revoked_count":      revokedCount,
	})
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_upload_unauthorized")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}

	// Parse multipart form (limit 10MB in memory)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_upload_parse_error")
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "failed to parse multipart form", nil)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("avatar")
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_upload_missing_file")
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "avatar file is required", nil)
		return
	}
	defer func() { _ = file.Close() }()

	// Upload to storage
	objectKey, err := h.storageSvc.UploadAvatar(r.Context(), userID, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_upload_storage_error")
		if errors.Is(err, service.ErrFileTooBig) {
			response.Error(w, r, http.StatusBadRequest, "FILE_TOO_LARGE", "file size exceeds 5MB limit", nil)
			return
		}
		if errors.Is(err, service.ErrInvalidFileType) {
			response.Error(w, r, http.StatusBadRequest, "INVALID_FILE_TYPE", "only JPEG and PNG images are allowed", nil)
			return
		}
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to upload avatar", nil)
		return
	}

	// Generate presigned URL for response
	avatarURL, err := h.storageSvc.GenerateAvatarURL(r.Context(), objectKey)
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_upload_url_error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to generate avatar URL", nil)
		return
	}

	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "avatar.upload",
		ActorUserID: observability.ActorUserID(userID),
		TargetType:  "avatar",
		TargetID:    objectKey,
		Action:      "upload",
		Outcome:     "success",
		Reason:      "avatar_uploaded",
	}, "object_key", objectKey, "file_size", header.Size, "content_type", header.Header.Get("Content-Type"))
	observability.RecordUserProfileEvent(r.Context(), "avatar_upload_success")
	response.JSON(w, r, http.StatusOK, map[string]any{
		"object_key":  objectKey,
		"avatar_url":  avatarURL,
		"file_size":   header.Size,
		"uploaded_at": "now",
	})
}

func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authUserIDAndClaims(r)
	if err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_delete_unauthorized")
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}

	// Get object_key from request body
	var req struct {
		ObjectKey string `json:"object_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_delete_parse_error")
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", nil)
		return
	}

	if req.ObjectKey == "" {
		observability.RecordUserProfileEvent(r.Context(), "avatar_delete_missing_key")
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "object_key is required", nil)
		return
	}

	// Delete from storage
	if err := h.storageSvc.DeleteAvatar(r.Context(), req.ObjectKey); err != nil {
		observability.RecordUserProfileEvent(r.Context(), "avatar_delete_storage_error")
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to delete avatar", nil)
		return
	}

	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "avatar.delete",
		ActorUserID: observability.ActorUserID(userID),
		TargetType:  "avatar",
		TargetID:    req.ObjectKey,
		Action:      "delete",
		Outcome:     "success",
		Reason:      "avatar_deleted",
	}, "object_key", req.ObjectKey)
	observability.RecordUserProfileEvent(r.Context(), "avatar_delete_success")
	response.JSON(w, r, http.StatusOK, map[string]any{
		"object_key": req.ObjectKey,
		"deleted":    true,
	})
}

func authUserIDAndClaims(r *http.Request) (uint, *security.Claims, error) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		return 0, nil, errors.New("missing auth context")
	}
	id64, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, nil, err
	}
	return uint(id64), claims, nil
}
