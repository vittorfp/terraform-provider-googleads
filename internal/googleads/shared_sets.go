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

// SharedSetInput models a shared set — a reusable bundle of negative
// keywords (or other criteria) that can be attached to many campaigns.
// `type` is immutable; only `name` can be updated in place.
type SharedSetInput struct {
	CustomerID string
	Name       string
	Type       string // NEGATIVE_KEYWORDS | NEGATIVE_PLACEMENTS | ...
}

type SharedSetView struct {
	ResourceName string
	Name         string
	Type         string
	Status       string
}

func (c *Client) CreateSharedSet(ctx context.Context, in SharedSetInput) (string, error) {
	cl, err := c.sharedSetClient(ctx)
	if err != nil {
		return "", err
	}
	set := &resources.SharedSet{
		Name: StringPtr(in.Name),
		Type: enums.SharedSetTypeEnum_SharedSetType(enums.SharedSetTypeEnum_SharedSetType_value[in.Type]),
	}
	resp, err := cl.MutateSharedSets(ctx, &services.MutateSharedSetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.SharedSetOperation{{
			Operation: &services.SharedSetOperation_Create{Create: set},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateSharedSet(ctx context.Context, resourceName, name string) error {
	cl, err := c.sharedSetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "sharedSets")
	if err != nil {
		return err
	}
	_, err = cl.MutateSharedSets(ctx, &services.MutateSharedSetsRequest{
		CustomerId: customerID,
		Operations: []*services.SharedSetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
			Operation:  &services.SharedSetOperation_Update{Update: &resources.SharedSet{ResourceName: resourceName, Name: StringPtr(name)}},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetSharedSet(ctx context.Context, resourceName string) (*SharedSetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "sharedSets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			shared_set.resource_name,
			shared_set.name,
			shared_set.type,
			shared_set.status
		FROM shared_set
		WHERE shared_set.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetSharedSet()
	return &SharedSetView{
		ResourceName: x.GetResourceName(),
		Name:         x.GetName(),
		Type:         x.GetType().String(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveSharedSet(ctx context.Context, resourceName string) error {
	cl, err := c.sharedSetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "sharedSets")
	if err != nil {
		return err
	}
	_, err = cl.MutateSharedSets(ctx, &services.MutateSharedSetsRequest{
		CustomerId: customerID,
		Operations: []*services.SharedSetOperation{{
			Operation: &services.SharedSetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) sharedSetClient(ctx context.Context) (*pbclients.SharedSetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sharedSets != nil {
		return c.sharedSets, nil
	}
	cl, err := pbclients.NewSharedSetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: shared set client: %w", err)
	}
	c.sharedSets = cl
	return cl, nil
}
