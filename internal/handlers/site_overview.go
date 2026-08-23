package handlers

import (
	"errors"

	"github.com/homeadmin/internal/services"
)

// siteOverviewService is the narrow dependency handlers need for site-admin
// global views. *services.SiteAdminService satisfies it; handler tests use a
// fake.
type siteOverviewService interface {
	SiteAdminOverview() ([]services.HouseholdBlock, error)
}

// fetchSiteOverview returns the cross-household overview, failing closed when
// the optional dependency was never wired.
func fetchSiteOverview(svc siteOverviewService) ([]services.HouseholdBlock, error) {
	if svc == nil {
		return nil, errors.New("site admin service not configured")
	}
	return svc.SiteAdminOverview()
}
