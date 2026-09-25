package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// BudgetInput is the Terraform-facing shape for a campaign budget. Pointer
// fields are optional — nil means "do not set / do not include in update".
type BudgetInput struct {
	CustomerID       string
	Name             string
	AmountMicros     int64
	DeliveryMethod   string // STANDARD | ACCELERATED; empty defaults to STANDARD on create
	ExplicitlyShared *bool
}

// BudgetView is the read-back shape returned by GetBudget.
type BudgetView struct {
	ResourceName     string
	Name             string
	AmountMicros     int64
	DeliveryMethod   string
	ExplicitlyShared bool
	Status           string
}

// CreateBudget creates a campaign budget and returns its resource name
// (customers/{cid}/campaignBudgets/{id}).
func (c *Client) CreateBudget(ctx context.Context, in BudgetInput) (string, error) {
	cl, err := c.budgetClient(ctx)
	if err != nil {
		return "", err
	}
	budget := &resources.CampaignBudget{
		Name:         StringPtr(in.Name),
		AmountMicros: Int64Ptr(in.AmountMicros),
	}
	if in.DeliveryMethod != "" {
		budget.DeliveryMethod = enums.BudgetDeliveryMethodEnum_BudgetDeliveryMethod(
			enums.BudgetDeliveryMethodEnum_BudgetDeliveryMethod_value[in.DeliveryMethod])
	}
	if in.ExplicitlyShared != nil {
		budget.ExplicitlyShared = in.ExplicitlyShared
	}
	resp, err := cl.MutateCampaignBudgets(ctx, &services.MutateCampaignBudgetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CampaignBudgetOperation{{
			Operation: &services.CampaignBudgetOperation_Create{Create: budget},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

// UpdateBudget mutates the named budget. Only the provided fields (the keys
// of paths) are sent in the update mask.
func (c *Client) UpdateBudget(ctx context.Context, resourceName string, in BudgetInput, paths []string) error {
	cl, err := c.budgetClient(ctx)
	if err != nil {
		return err
	}
	budget := &resources.CampaignBudget{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "name":
			budget.Name = StringPtr(in.Name)
		case "amount_micros":
			budget.AmountMicros = Int64Ptr(in.AmountMicros)
		case "delivery_method":
			budget.DeliveryMethod = enums.BudgetDeliveryMethodEnum_BudgetDeliveryMethod(
				enums.BudgetDeliveryMethodEnum_BudgetDeliveryMethod_value[in.DeliveryMethod])
		case "explicitly_shared":
			if in.ExplicitlyShared != nil {
				budget.ExplicitlyShared = in.ExplicitlyShared
			}
		}
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignBudgets")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignBudgets(ctx, &services.MutateCampaignBudgetsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignBudgetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.CampaignBudgetOperation_Update{Update: budget},
		}},
	})
	return WrapAPIError(err)
}

// GetBudget looks up a budget by its resource name and returns its current
// state. Returns ErrNotFound if the budget no longer exists.
func (c *Client) GetBudget(ctx context.Context, resourceName string) (*BudgetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "campaignBudgets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			campaign_budget.resource_name,
			campaign_budget.name,
			campaign_budget.amount_micros,
			campaign_budget.delivery_method,
			campaign_budget.explicitly_shared,
			campaign_budget.status
		FROM campaign_budget
		WHERE campaign_budget.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	b := row.GetCampaignBudget()
	return &BudgetView{
		ResourceName:     b.GetResourceName(),
		Name:             b.GetName(),
		AmountMicros:     b.GetAmountMicros(),
		DeliveryMethod:   b.GetDeliveryMethod().String(),
		ExplicitlyShared: b.GetExplicitlyShared(),
		Status:           b.GetStatus().String(),
	}, nil
}

// RemoveBudget deletes the named budget.
func (c *Client) RemoveBudget(ctx context.Context, resourceName string) error {
	cl, err := c.budgetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignBudgets")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignBudgets(ctx, &services.MutateCampaignBudgetsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignBudgetOperation{{
			Operation: &services.CampaignBudgetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}
