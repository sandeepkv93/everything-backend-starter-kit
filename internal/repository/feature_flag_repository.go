package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/observability"
)

var (
	ErrFeatureFlagNotFound     = errors.New("feature flag not found")
	ErrFeatureFlagRuleNotFound = errors.New("feature flag rule not found")
)

type FeatureFlagRepository interface {
	ListFlags() ([]domain.FeatureFlag, error)
	FindFlagByID(id uint) (*domain.FeatureFlag, error)
	FindFlagByKey(key string) (*domain.FeatureFlag, error)
	CreateFlag(flag *domain.FeatureFlag) error
	UpdateFlag(flag *domain.FeatureFlag) error
	DeleteFlag(id uint) error
	ListRules(flagID uint) ([]domain.FeatureFlagRule, error)
	CreateRule(rule *domain.FeatureFlagRule) error
	UpdateRule(rule *domain.FeatureFlagRule) error
	DeleteRule(flagID, ruleID uint) error
}

type GormFeatureFlagRepository struct{ db *gorm.DB }

func NewFeatureFlagRepository(db *gorm.DB) FeatureFlagRepository {
	return &GormFeatureFlagRepository{db: db}
}

func (r *GormFeatureFlagRepository) ListFlags() ([]domain.FeatureFlag, error) {
	var flags []domain.FeatureFlag
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("priority asc").Order("id asc")
	}).Order("key asc").Find(&flags).Error
	if err != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "list", "error")
		return nil, err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "list", "success")
	return flags, nil
}

func (r *GormFeatureFlagRepository) FindFlagByID(id uint) (*domain.FeatureFlag, error) {
	var flag domain.FeatureFlag
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("priority asc").Order("id asc")
	}).First(&flag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_id", "not_found")
			return nil, ErrFeatureFlagNotFound
		}
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_id", "error")
		return nil, err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_id", "success")
	return &flag, nil
}

func (r *GormFeatureFlagRepository) FindFlagByKey(key string) (*domain.FeatureFlag, error) {
	var flag domain.FeatureFlag
	err := r.db.Preload("Rules", func(db *gorm.DB) *gorm.DB {
		return db.Order("priority asc").Order("id asc")
	}).Where("key = ?", strings.TrimSpace(strings.ToLower(key))).First(&flag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_key", "not_found")
			return nil, ErrFeatureFlagNotFound
		}
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_key", "error")
		return nil, err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "find_by_key", "success")
	return &flag, nil
}

func (r *GormFeatureFlagRepository) CreateFlag(flag *domain.FeatureFlag) error {
	if err := r.db.Create(flag).Error; err != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "create", "error")
		return err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "create", "success")
	return nil
}

func (r *GormFeatureFlagRepository) UpdateFlag(flag *domain.FeatureFlag) error {
	res := r.db.Model(&domain.FeatureFlag{}).Where("id = ?", flag.ID).Updates(map[string]any{
		"key":         strings.TrimSpace(strings.ToLower(flag.Key)),
		"description": strings.TrimSpace(flag.Description),
		"enabled":     flag.Enabled,
	})
	if res.Error != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "update", "error")
		return res.Error
	}
	if res.RowsAffected == 0 {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "update", "not_found")
		return ErrFeatureFlagNotFound
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "update", "success")
	return nil
}

func (r *GormFeatureFlagRepository) DeleteFlag(id uint) error {
	res := r.db.Delete(&domain.FeatureFlag{}, id)
	if res.Error != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "delete", "error")
		return res.Error
	}
	if res.RowsAffected == 0 {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag", "delete", "not_found")
		return ErrFeatureFlagNotFound
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag", "delete", "success")
	return nil
}

func (r *GormFeatureFlagRepository) ListRules(flagID uint) ([]domain.FeatureFlagRule, error) {
	var rules []domain.FeatureFlagRule
	err := r.db.Where("feature_flag_id = ?", flagID).Order("priority asc").Order("id asc").Find(&rules).Error
	if err != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "list", "error")
		return nil, err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "list", "success")
	return rules, nil
}

func (r *GormFeatureFlagRepository) CreateRule(rule *domain.FeatureFlagRule) error {
	if err := r.db.Create(rule).Error; err != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "create", "error")
		return err
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "create", "success")
	return nil
}

func (r *GormFeatureFlagRepository) UpdateRule(rule *domain.FeatureFlagRule) error {
	res := r.db.Model(&domain.FeatureFlagRule{}).
		Where("id = ? AND feature_flag_id = ?", rule.ID, rule.FeatureFlagID).
		Updates(map[string]any{
			"type":        strings.TrimSpace(strings.ToLower(rule.Type)),
			"match_value": strings.TrimSpace(strings.ToLower(rule.MatchValue)),
			"percentage":  rule.Percentage,
			"enabled":     rule.Enabled,
			"priority":    rule.Priority,
		})
	if res.Error != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "update", "error")
		return res.Error
	}
	if res.RowsAffected == 0 {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "update", "not_found")
		return ErrFeatureFlagRuleNotFound
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "update", "success")
	return nil
}

func (r *GormFeatureFlagRepository) DeleteRule(flagID, ruleID uint) error {
	res := r.db.Where("id = ? AND feature_flag_id = ?", ruleID, flagID).Delete(&domain.FeatureFlagRule{})
	if res.Error != nil {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "delete", "error")
		return res.Error
	}
	if res.RowsAffected == 0 {
		observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "delete", "not_found")
		return ErrFeatureFlagRuleNotFound
	}
	observability.RecordRepositoryOperation(context.Background(), "feature_flag_rule", "delete", "success")
	return nil
}
