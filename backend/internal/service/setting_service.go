package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// ErrInvalidSettingValue is returned when a PATCH request supplies an unrecognised value.
var ErrInvalidSettingValue = errors.New("invalid setting value")

// SettingService manages global platform settings.
type SettingService interface {
	GetAll(ctx context.Context) (*dto.SettingsResponse, error)
	Patch(ctx context.Context, userID uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error)
}

var canonicalDefaults = map[string]string{
	domain.SettingKeyOrganizationName: "Mansooba",
	domain.SettingKeyDateFormat:       "YYYY-MM-DD",
	domain.SettingKeyTimeFormat:       "24h",
	domain.SettingKeyLocale:           "en-US",
	domain.SettingKeyWeekStartDay:     "monday",
}

var validDateFormats = map[string]bool{"YYYY-MM-DD": true, "DD/MM/YYYY": true, "MM/DD/YYYY": true, "D-MMM-YYYY": true}
var validTimeFormats = map[string]bool{"12h": true, "24h": true}
var validWeekStartDays = map[string]bool{"monday": true, "sunday": true}

// bcp47Pattern is a simplified BCP-47 check: 2-3 letter language tag, optional region.
var bcp47Pattern = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z]{2,3})?$`)

type settingServiceImpl struct {
	repo domain.GlobalSettingRepository
}

// NewSettingService returns a SettingService backed by the given repository.
func NewSettingService(repo domain.GlobalSettingRepository) SettingService {
	return &settingServiceImpl{repo: repo}
}

func (s *settingServiceImpl) GetAll(ctx context.Context) (*dto.SettingsResponse, error) {
	rows, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]string, len(canonicalDefaults))
	for k, v := range canonicalDefaults {
		merged[k] = v
	}
	for _, row := range rows {
		merged[row.SettingKey] = row.SettingValue
	}
	return toSettingsResponse(merged), nil
}

func (s *settingServiceImpl) Patch(ctx context.Context, userID uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error) {
	updates := map[string]*string{
		domain.SettingKeyOrganizationName: req.OrganizationName,
		domain.SettingKeyDateFormat:       req.DateFormat,
		domain.SettingKeyTimeFormat:       req.TimeFormat,
		domain.SettingKeyLocale:           req.Locale,
		domain.SettingKeyWeekStartDay:     req.WeekStartDay,
	}
	for key, val := range updates {
		if val == nil {
			continue
		}
		if err := validateSettingValue(key, *val); err != nil {
			return nil, err
		}
		if err := s.repo.Upsert(ctx, &domain.GlobalSetting{
			SettingKey:   key,
			SettingValue: *val,
			UpdatedByID:  userID,
			UpdatedAt:    time.Now(),
		}); err != nil {
			return nil, err
		}
	}
	return s.GetAll(ctx)
}

func validateSettingValue(key, value string) error {
	switch key {
	case domain.SettingKeyDateFormat:
		if !validDateFormats[value] {
			return ErrInvalidSettingValue
		}
	case domain.SettingKeyTimeFormat:
		if !validTimeFormats[value] {
			return ErrInvalidSettingValue
		}
	case domain.SettingKeyWeekStartDay:
		if !validWeekStartDays[value] {
			return ErrInvalidSettingValue
		}
	case domain.SettingKeyLocale:
		if !bcp47Pattern.MatchString(value) {
			return ErrInvalidSettingValue
		}
	}
	return nil
}

func toSettingsResponse(m map[string]string) *dto.SettingsResponse {
	return &dto.SettingsResponse{
		OrganizationName: m[domain.SettingKeyOrganizationName],
		DateFormat:       m[domain.SettingKeyDateFormat],
		TimeFormat:       m[domain.SettingKeyTimeFormat],
		Locale:           m[domain.SettingKeyLocale],
		WeekStartDay:     m[domain.SettingKeyWeekStartDay],
	}
}
