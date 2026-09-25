package googleads

import (
	"github.com/shenzhencenter/google-ads-pb/resources"
)

// mustBuildBudgetRow constructs a *resources.CampaignBudget pre-populated
// with the fields a Get test needs. Pointer setters keep this readable.
func mustBuildBudgetRow(resourceName, name string, amountMicros int64) *resources.CampaignBudget {
	return &resources.CampaignBudget{
		ResourceName: resourceName,
		Name:         StringPtr(name),
		AmountMicros: Int64Ptr(amountMicros),
	}
}
