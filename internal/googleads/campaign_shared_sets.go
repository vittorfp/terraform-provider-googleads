package googleads

import (
	"context"
	"fmt"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// CampaignSharedSetInput is the M:N link between a campaign and a shared
// set. Both endpoints are immutable per the API — no Update operation
// exists, so changing either forces resource replacement.
type CampaignSharedSetInput struct {
	CustomerID string
	Campaign   string // resource name
	SharedSet  string // resource name
}

type CampaignSharedSetView struct {
	ResourceName string
	Campaign     string
	SharedSet    string
	Status       string
}

func (c *Client) CreateCampaignSharedSet(ctx context.Context, in CampaignSharedSetInput) (string, error) {
	cl, err := c.campaignSharedSetClient(ctx)
	if err != nil {
		return "", err
	}
	link := &resources.CampaignSharedSet{
		Campaign:  StringPtr(in.Campaign),
		SharedSet: StringPtr(in.SharedSet),
	}
	resp, err := cl.MutateCampaignSharedSets(ctx, &services.MutateCampaignSharedSetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CampaignSharedSetOperation{{
			Operation: &services.CampaignSharedSetOperation_Create{Create: link},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetCampaignSharedSet(ctx context.Context, resourceName string) (*CampaignSharedSetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "campaignSharedSets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			campaign_shared_set.resource_name,
			campaign_shared_set.campaign,
			campaign_shared_set.shared_set,
			campaign_shared_set.status
		FROM campaign_shared_set
		WHERE campaign_shared_set.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetCampaignSharedSet()
	return &CampaignSharedSetView{
		ResourceName: x.GetResourceName(),
		Campaign:     x.GetCampaign(),
		SharedSet:    x.GetSharedSet(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveCampaignSharedSet(ctx context.Context, resourceName string) error {
	cl, err := c.campaignSharedSetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignSharedSets")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignSharedSets(ctx, &services.MutateCampaignSharedSetsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignSharedSetOperation{{
			Operation: &services.CampaignSharedSetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) campaignSharedSetClient(ctx context.Context) (*pbclients.CampaignSharedSetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.campaignSharedSets != nil {
		return c.campaignSharedSets, nil
	}
	cl, err := pbclients.NewCampaignSharedSetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: campaign shared set client: %w", err)
	}
	c.campaignSharedSets = cl
	return cl, nil
}
