package googleads

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// CustomerConversionGoalInput is the per-(category, origin) goal config
// at the customer level. The API has no Create/Remove for these — every
// pair always exists in every account; only `biddable` toggles.
type CustomerConversionGoalInput struct {
	CustomerID string
	Category   string // PURCHASE | SIGNUP | DEFAULT | PAGE_VIEW | ...
	Origin     string // WEBSITE | APP | STORE | ...
	Biddable   bool
}

type CustomerConversionGoalView struct {
	ResourceName string
	Category     string
	Origin       string
	Biddable     bool
}

// SetCustomerConversionGoalBiddable flips the biddable flag on the
// (category, origin) pair under the given customer. Returns the
// resource name even though it's derivable — callers store it as the
// Terraform `id` for state-tracking.
func (c *Client) SetCustomerConversionGoalBiddable(ctx context.Context, in CustomerConversionGoalInput) (string, error) {
	cl, err := c.customerConversionGoalClient(ctx)
	if err != nil {
		return "", err
	}
	rn := fmt.Sprintf("customers/%s/customerConversionGoals/%s~%s", in.CustomerID, in.Category, in.Origin)
	goal := &resources.CustomerConversionGoal{
		ResourceName: rn,
		Category:     enums.ConversionActionCategoryEnum_ConversionActionCategory(enums.ConversionActionCategoryEnum_ConversionActionCategory_value[in.Category]),
		Origin:       enums.ConversionOriginEnum_ConversionOrigin(enums.ConversionOriginEnum_ConversionOrigin_value[in.Origin]),
		Biddable:     in.Biddable,
	}
	_, err = cl.MutateCustomerConversionGoals(ctx, &services.MutateCustomerConversionGoalsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CustomerConversionGoalOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"biddable"}},
			Operation:  &services.CustomerConversionGoalOperation_Update{Update: goal},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return rn, nil
}

func (c *Client) GetCustomerConversionGoal(ctx context.Context, resourceName string) (*CustomerConversionGoalView, error) {
	customerID, _, err := ParseResourceName(resourceName, "customerConversionGoals")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			customer_conversion_goal.resource_name,
			customer_conversion_goal.category,
			customer_conversion_goal.origin,
			customer_conversion_goal.biddable
		FROM customer_conversion_goal
		WHERE customer_conversion_goal.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	g := row.GetCustomerConversionGoal()
	return &CustomerConversionGoalView{
		ResourceName: g.GetResourceName(),
		Category:     g.GetCategory().String(),
		Origin:       g.GetOrigin().String(),
		Biddable:     g.GetBiddable(),
	}, nil
}

func (c *Client) customerConversionGoalClient(ctx context.Context) (*pbclients.CustomerConversionGoalClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.customerConversionGoals != nil {
		return c.customerConversionGoals, nil
	}
	cl, err := pbclients.NewCustomerConversionGoalClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: customer conversion goal client: %w", err)
	}
	c.customerConversionGoals = cl
	return cl, nil
}
