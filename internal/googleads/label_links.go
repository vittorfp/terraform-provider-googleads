package googleads

import (
	"context"
	"fmt"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// The four label-link resources (campaign_label, ad_group_label,
// ad_group_ad_label, ad_group_criterion_label) share the same shape:
// two immutable foreign-key fields, Create + Remove only on the API
// (no Update). All client methods follow the same pattern.

// --- campaign_label ---

func (c *Client) CreateCampaignLabel(ctx context.Context, customerID, campaign, label string) (string, error) {
	cl, err := c.campaignLabelClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateCampaignLabels(ctx, &services.MutateCampaignLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignLabelOperation{{
			Operation: &services.CampaignLabelOperation_Create{
				Create: &resources.CampaignLabel{
					Campaign: StringPtr(campaign),
					Label:    StringPtr(label),
				},
			},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetCampaignLabel(ctx context.Context, resourceName string) (campaign, label string, err error) {
	customerID, _, err := ParseResourceName(resourceName, "campaignLabels")
	if err != nil {
		return "", "", err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT campaign_label.resource_name, campaign_label.campaign, campaign_label.label
		FROM campaign_label
		WHERE campaign_label.resource_name = '`+resourceName+`'`)
	if err != nil {
		return "", "", err
	}
	x := row.GetCampaignLabel()
	return x.GetCampaign(), x.GetLabel(), nil
}

func (c *Client) RemoveCampaignLabel(ctx context.Context, resourceName string) error {
	cl, err := c.campaignLabelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignLabels")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignLabels(ctx, &services.MutateCampaignLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignLabelOperation{{
			Operation: &services.CampaignLabelOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) campaignLabelClient(ctx context.Context) (*pbclients.CampaignLabelClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.campaignLabels != nil {
		return c.campaignLabels, nil
	}
	cl, err := pbclients.NewCampaignLabelClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: campaign label client: %w", err)
	}
	c.campaignLabels = cl
	return cl, nil
}

// --- ad_group_label ---

func (c *Client) CreateAdGroupLabel(ctx context.Context, customerID, adGroup, label string) (string, error) {
	cl, err := c.adGroupLabelClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateAdGroupLabels(ctx, &services.MutateAdGroupLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupLabelOperation{{
			Operation: &services.AdGroupLabelOperation_Create{
				Create: &resources.AdGroupLabel{
					AdGroup: StringPtr(adGroup),
					Label:   StringPtr(label),
				},
			},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetAdGroupLabel(ctx context.Context, resourceName string) (adGroup, label string, err error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupLabels")
	if err != nil {
		return "", "", err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT ad_group_label.resource_name, ad_group_label.ad_group, ad_group_label.label
		FROM ad_group_label
		WHERE ad_group_label.resource_name = '`+resourceName+`'`)
	if err != nil {
		return "", "", err
	}
	x := row.GetAdGroupLabel()
	return x.GetAdGroup(), x.GetLabel(), nil
}

func (c *Client) RemoveAdGroupLabel(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupLabelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupLabels")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupLabels(ctx, &services.MutateAdGroupLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupLabelOperation{{
			Operation: &services.AdGroupLabelOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) adGroupLabelClient(ctx context.Context) (*pbclients.AdGroupLabelClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adGroupLabels != nil {
		return c.adGroupLabels, nil
	}
	cl, err := pbclients.NewAdGroupLabelClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group label client: %w", err)
	}
	c.adGroupLabels = cl
	return cl, nil
}

// --- ad_group_ad_label ---

func (c *Client) CreateAdGroupAdLabel(ctx context.Context, customerID, adGroupAd, label string) (string, error) {
	cl, err := c.adGroupAdLabelClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateAdGroupAdLabels(ctx, &services.MutateAdGroupAdLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAdLabelOperation{{
			Operation: &services.AdGroupAdLabelOperation_Create{
				Create: &resources.AdGroupAdLabel{
					AdGroupAd: StringPtr(adGroupAd),
					Label:     StringPtr(label),
				},
			},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetAdGroupAdLabel(ctx context.Context, resourceName string) (adGroupAd, label string, err error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupAdLabels")
	if err != nil {
		return "", "", err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT ad_group_ad_label.resource_name, ad_group_ad_label.ad_group_ad, ad_group_ad_label.label
		FROM ad_group_ad_label
		WHERE ad_group_ad_label.resource_name = '`+resourceName+`'`)
	if err != nil {
		return "", "", err
	}
	x := row.GetAdGroupAdLabel()
	return x.GetAdGroupAd(), x.GetLabel(), nil
}

func (c *Client) RemoveAdGroupAdLabel(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupAdLabelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupAdLabels")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupAdLabels(ctx, &services.MutateAdGroupAdLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAdLabelOperation{{
			Operation: &services.AdGroupAdLabelOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) adGroupAdLabelClient(ctx context.Context) (*pbclients.AdGroupAdLabelClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adGroupAdLabels != nil {
		return c.adGroupAdLabels, nil
	}
	cl, err := pbclients.NewAdGroupAdLabelClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group ad label client: %w", err)
	}
	c.adGroupAdLabels = cl
	return cl, nil
}

// --- ad_group_criterion_label ---

func (c *Client) CreateAdGroupCriterionLabel(ctx context.Context, customerID, adGroupCriterion, label string) (string, error) {
	cl, err := c.adGroupCriterionLabelClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateAdGroupCriterionLabels(ctx, &services.MutateAdGroupCriterionLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionLabelOperation{{
			Operation: &services.AdGroupCriterionLabelOperation_Create{
				Create: &resources.AdGroupCriterionLabel{
					AdGroupCriterion: StringPtr(adGroupCriterion),
					Label:            StringPtr(label),
				},
			},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetAdGroupCriterionLabel(ctx context.Context, resourceName string) (adGroupCriterion, label string, err error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriterionLabels")
	if err != nil {
		return "", "", err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT ad_group_criterion_label.resource_name, ad_group_criterion_label.ad_group_criterion, ad_group_criterion_label.label
		FROM ad_group_criterion_label
		WHERE ad_group_criterion_label.resource_name = '`+resourceName+`'`)
	if err != nil {
		return "", "", err
	}
	x := row.GetAdGroupCriterionLabel()
	return x.GetAdGroupCriterion(), x.GetLabel(), nil
}

func (c *Client) RemoveAdGroupCriterionLabel(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupCriterionLabelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriterionLabels")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupCriterionLabels(ctx, &services.MutateAdGroupCriterionLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionLabelOperation{{
			Operation: &services.AdGroupCriterionLabelOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) adGroupCriterionLabelClient(ctx context.Context) (*pbclients.AdGroupCriterionLabelClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adGroupCriterionLabels != nil {
		return c.adGroupCriterionLabels, nil
	}
	cl, err := pbclients.NewAdGroupCriterionLabelClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group criterion label client: %w", err)
	}
	c.adGroupCriterionLabels = cl
	return cl, nil
}
