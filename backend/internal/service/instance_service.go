package service

import "github.com/sharique/mansooba/internal/domain"

// InstanceService serves this instance's own version, source location and
// licence. It holds no config
// reading of its own — the values are resolved once in main.go and injected.
type InstanceService interface {
	Info() domain.InstanceInfo
}

type instanceService struct {
	info domain.InstanceInfo
}

// NewInstanceService builds the service from already-validated values. An
// empty version is normalized to "dev" (a local build with no version baked
// in); License/LicenseURL are always the fixed domain constants regardless
// of what the caller passes.
func NewInstanceService(info domain.InstanceInfo) InstanceService {
	if info.Version == "" {
		info.Version = "dev"
	}
	info.License = domain.InstanceLicense
	info.LicenseURL = domain.InstanceLicenseURL
	return &instanceService{info: info}
}

func (s *instanceService) Info() domain.InstanceInfo {
	return s.info
}
