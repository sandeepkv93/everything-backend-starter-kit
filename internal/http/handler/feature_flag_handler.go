package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/http/response"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/observability"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/repository"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/service"
)

var featureFlagKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_\-]{0,127}$`)

type FeatureFlagHandler struct {
	svc service.FeatureFlagService
}

func NewFeatureFlagHandler(svc service.FeatureFlagService) *FeatureFlagHandler {
	return &FeatureFlagHandler{svc: svc}
}

func (h *FeatureFlagHandler) EvaluateAll(w http.ResponseWriter, r *http.Request) {
	userID, claims, err := authUserIDAndClaims(r)
	if err != nil {
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}
	ctx := service.FeatureFlagEvaluationContext{
		UserID:      userID,
		Roles:       claims.Roles,
		Org:         strings.TrimSpace(r.URL.Query().Get("org")),
		Environment: strings.TrimSpace(r.URL.Query().Get("environment")),
	}
	results, err := h.svc.EvaluateAll(r.Context(), ctx)
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to evaluate feature flags", nil)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": results})
}

func (h *FeatureFlagHandler) EvaluateOne(w http.ResponseWriter, r *http.Request) {
	userID, claims, err := authUserIDAndClaims(r)
	if err != nil {
		response.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user", nil)
		return
	}
	key := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "key")))
	if !featureFlagKeyRe.MatchString(key) {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid feature flag key", nil)
		return
	}
	ctx := service.FeatureFlagEvaluationContext{
		UserID:      userID,
		Roles:       claims.Roles,
		Org:         strings.TrimSpace(r.URL.Query().Get("org")),
		Environment: strings.TrimSpace(r.URL.Query().Get("environment")),
	}
	result, err := h.svc.EvaluateByKey(r.Context(), key, ctx)
	if err != nil {
		if errors.Is(err, repository.ErrFeatureFlagNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag not found", nil)
			return
		}
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to evaluate feature flag", nil)
		return
	}
	response.JSON(w, r, http.StatusOK, result)
}

func (h *FeatureFlagHandler) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := h.svc.ListFlags(r.Context())
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to list feature flags", nil)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": flags})
}

func (h *FeatureFlagHandler) GetFlag(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	flag, err := h.svc.GetFlagByID(r.Context(), flagID)
	if err != nil {
		if errors.Is(err, repository.ErrFeatureFlagNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag not found", nil)
			return
		}
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to load feature flag", nil)
		return
	}
	response.JSON(w, r, http.StatusOK, flag)
}

func (h *FeatureFlagHandler) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key         string `json:"key"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid payload", nil)
		return
	}
	key := strings.TrimSpace(strings.ToLower(body.Key))
	if !featureFlagKeyRe.MatchString(key) {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid feature flag key", nil)
		return
	}
	flag := &serviceFeatureFlagDTO{Key: key, Description: strings.TrimSpace(body.Description), Enabled: body.Enabled}
	domainFlag := flag.toDomain()
	if err := h.svc.CreateFlag(r.Context(), domainFlag); err != nil {
		if isConflictError(err) {
			response.Error(w, r, http.StatusConflict, "CONFLICT", "feature flag already exists", nil)
			return
		}
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "failed to create feature flag", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.create",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag",
		TargetID:    strconv.FormatUint(uint64(domainFlag.ID), 10),
		Action:      "create",
		Outcome:     "success",
		Reason:      "feature_flag_created",
	}, "key", domainFlag.Key)
	response.JSON(w, r, http.StatusCreated, domainFlag)
}

func (h *FeatureFlagHandler) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	var body struct {
		Key         string `json:"key"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid payload", nil)
		return
	}
	key := strings.TrimSpace(strings.ToLower(body.Key))
	if !featureFlagKeyRe.MatchString(key) {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid feature flag key", nil)
		return
	}
	flag := &serviceFeatureFlagDTO{ID: flagID, Key: key, Description: strings.TrimSpace(body.Description), Enabled: body.Enabled}
	domainFlag := flag.toDomain()
	if err := h.svc.UpdateFlag(r.Context(), domainFlag); err != nil {
		if errors.Is(err, repository.ErrFeatureFlagNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag not found", nil)
			return
		}
		if isConflictError(err) {
			response.Error(w, r, http.StatusConflict, "CONFLICT", "feature flag already exists", nil)
			return
		}
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "failed to update feature flag", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.update",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag",
		TargetID:    strconv.FormatUint(uint64(flagID), 10),
		Action:      "update",
		Outcome:     "success",
		Reason:      "feature_flag_updated",
	}, "key", domainFlag.Key)
	response.JSON(w, r, http.StatusOK, domainFlag)
}

func (h *FeatureFlagHandler) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	if err := h.svc.DeleteFlag(r.Context(), flagID); err != nil {
		if errors.Is(err, repository.ErrFeatureFlagNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag not found", nil)
			return
		}
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to delete feature flag", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.delete",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag",
		TargetID:    strconv.FormatUint(uint64(flagID), 10),
		Action:      "delete",
		Outcome:     "success",
		Reason:      "feature_flag_deleted",
	})
	response.JSON(w, r, http.StatusOK, map[string]any{"deleted": true})
}

func (h *FeatureFlagHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	rules, err := h.svc.ListRules(r.Context(), flagID)
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to list rules", nil)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]any{"items": rules})
}

func (h *FeatureFlagHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	var body struct {
		Type       string `json:"type"`
		MatchValue string `json:"match_value"`
		Percentage int    `json:"percentage"`
		Enabled    bool   `json:"enabled"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid payload", nil)
		return
	}
	rule := &serviceFeatureFlagRuleDTO{FeatureFlagID: flagID, Type: body.Type, MatchValue: body.MatchValue, Percentage: body.Percentage, Enabled: body.Enabled, Priority: body.Priority}
	domainRule := rule.toDomain()
	if err := h.svc.CreateRule(r.Context(), domainRule); err != nil {
		if errors.Is(err, service.ErrFeatureFlagInvalidRuleType) || errors.Is(err, service.ErrFeatureFlagInvalidRuleValue) {
			response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
		if isConflictError(err) {
			response.Error(w, r, http.StatusConflict, "CONFLICT", "feature flag rule already exists", nil)
			return
		}
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "failed to create rule", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.rule.create",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag_rule",
		TargetID:    strconv.FormatUint(uint64(domainRule.ID), 10),
		Action:      "create",
		Outcome:     "success",
		Reason:      "feature_flag_rule_created",
	}, "feature_flag_id", flagID)
	response.JSON(w, r, http.StatusCreated, domainRule)
}

func (h *FeatureFlagHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	ruleID, err := parsePathID(chi.URLParam(r, "rule_id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id", nil)
		return
	}
	var body struct {
		Type       string `json:"type"`
		MatchValue string `json:"match_value"`
		Percentage int    `json:"percentage"`
		Enabled    bool   `json:"enabled"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid payload", nil)
		return
	}
	rule := &serviceFeatureFlagRuleDTO{ID: ruleID, FeatureFlagID: flagID, Type: body.Type, MatchValue: body.MatchValue, Percentage: body.Percentage, Enabled: body.Enabled, Priority: body.Priority}
	domainRule := rule.toDomain()
	if err := h.svc.UpdateRule(r.Context(), domainRule); err != nil {
		if errors.Is(err, repository.ErrFeatureFlagRuleNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag rule not found", nil)
			return
		}
		if errors.Is(err, service.ErrFeatureFlagInvalidRuleType) || errors.Is(err, service.ErrFeatureFlagInvalidRuleValue) {
			response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
			return
		}
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "failed to update rule", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.rule.update",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag_rule",
		TargetID:    strconv.FormatUint(uint64(ruleID), 10),
		Action:      "update",
		Outcome:     "success",
		Reason:      "feature_flag_rule_updated",
	}, "feature_flag_id", flagID)
	response.JSON(w, r, http.StatusOK, domainRule)
}

func (h *FeatureFlagHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	flagID, err := parsePathID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid flag id", nil)
		return
	}
	ruleID, err := parsePathID(chi.URLParam(r, "rule_id"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id", nil)
		return
	}
	if err := h.svc.DeleteRule(r.Context(), flagID, ruleID); err != nil {
		if errors.Is(err, repository.ErrFeatureFlagRuleNotFound) {
			response.Error(w, r, http.StatusNotFound, "NOT_FOUND", "feature flag rule not found", nil)
			return
		}
		response.Error(w, r, http.StatusInternalServerError, "INTERNAL", "failed to delete rule", nil)
		return
	}
	observability.EmitAudit(r, observability.AuditInput{
		EventName:   "feature_flag.rule.delete",
		ActorUserID: adminActorID(r),
		TargetType:  "feature_flag_rule",
		TargetID:    strconv.FormatUint(uint64(ruleID), 10),
		Action:      "delete",
		Outcome:     "success",
		Reason:      "feature_flag_rule_deleted",
	}, "feature_flag_id", flagID)
	response.JSON(w, r, http.StatusOK, map[string]any{"deleted": true})
}

type serviceFeatureFlagDTO struct {
	ID          uint
	Key         string
	Description string
	Enabled     bool
}

func (d *serviceFeatureFlagDTO) toDomain() *domain.FeatureFlag {
	return &domain.FeatureFlag{ID: d.ID, Key: d.Key, Description: d.Description, Enabled: d.Enabled}
}

type serviceFeatureFlagRuleDTO struct {
	ID            uint
	FeatureFlagID uint
	Type          string
	MatchValue    string
	Percentage    int
	Enabled       bool
	Priority      int
}

func (d *serviceFeatureFlagRuleDTO) toDomain() *domain.FeatureFlagRule {
	return &domain.FeatureFlagRule{ID: d.ID, FeatureFlagID: d.FeatureFlagID, Type: d.Type, MatchValue: d.MatchValue, Percentage: d.Percentage, Enabled: d.Enabled, Priority: d.Priority}
}
