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

// CampaignAssetInput links an asset (sitelink, callout, snippet, etc.)
// to a campaign with a specific field_type that describes how the asset
// is used. Identity (campaign, asset, field_type) is immutable; only
// status mutates in place.
type CampaignAssetInput struct {
	CustomerID string
	Campaign   string
	Asset      string
	FieldType  string // SITELINK | CALLOUT | STRUCTURED_SNIPPET | ...
	Status     string // ENABLED | PAUSED | REMOVED
}

type CampaignAssetView struct {
	ResourceName string
	Campaign     string
	Asset        string
	FieldType    string
	Status       string
}

func (c *Client) CreateCampaignAsset(ctx context.Context, in CampaignAssetInput) (string, error) {
	cl, err := c.campaignAssetClient(ctx)
	if err != nil {
		return "", err
	}
	link := &resources.CampaignAsset{
		Campaign:  StringPtr(in.Campaign),
		Asset:     StringPtr(in.Asset),
		FieldType: enums.AssetFieldTypeEnum_AssetFieldType(enums.AssetFieldTypeEnum_AssetFieldType_value[in.FieldType]),
	}
	if in.Status != "" {
		link.Status = enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[in.Status])
	}
	resp, err := cl.MutateCampaignAssets(ctx, &services.MutateCampaignAssetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CampaignAssetOperation{{
			Operation: &services.CampaignAssetOperation_Create{Create: link},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateCampaignAssetStatus(ctx context.Context, resourceName, status string) error {
	cl, err := c.campaignAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignAssets")
	if err != nil {
		return err
	}
	link := &resources.CampaignAsset{
		ResourceName: resourceName,
		Status:       enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[status]),
	}
	_, err = cl.MutateCampaignAssets(ctx, &services.MutateCampaignAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignAssetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"status"}},
			Operation:  &services.CampaignAssetOperation_Update{Update: link},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetCampaignAsset(ctx context.Context, resourceName string) (*CampaignAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "campaignAssets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			campaign_asset.resource_name,
			campaign_asset.campaign,
			campaign_asset.asset,
			campaign_asset.field_type,
			campaign_asset.status
		FROM campaign_asset
		WHERE campaign_asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetCampaignAsset()
	return &CampaignAssetView{
		ResourceName: x.GetResourceName(),
		Campaign:     x.GetCampaign(),
		Asset:        x.GetAsset(),
		FieldType:    x.GetFieldType().String(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveCampaignAsset(ctx context.Context, resourceName string) error {
	cl, err := c.campaignAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignAssets")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignAssets(ctx, &services.MutateCampaignAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignAssetOperation{{
			Operation: &services.CampaignAssetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) campaignAssetClient(ctx context.Context) (*pbclients.CampaignAssetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.campaignAssets != nil {
		return c.campaignAssets, nil
	}
	cl, err := pbclients.NewCampaignAssetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: campaign asset client: %w", err)
	}
	c.campaignAssets = cl
	return cl, nil
}
