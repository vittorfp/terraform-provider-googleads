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

// AdGroupAssetInput links an asset (sitelink, callout, etc.) to an ad
// group with a field_type. Identity (ad_group, asset, field_type) is
// immutable; status mutates in place. Mirrors CampaignAssetInput's
// shape at the ad-group scope.
type AdGroupAssetInput struct {
	CustomerID string
	AdGroup    string
	Asset      string
	FieldType  string
	Status     string
}

type AdGroupAssetView struct {
	ResourceName string
	AdGroup      string
	Asset        string
	FieldType    string
	Status       string
}

func (c *Client) CreateAdGroupAssetLink(ctx context.Context, in AdGroupAssetInput) (string, error) {
	cl, err := c.adGroupAssetClient(ctx)
	if err != nil {
		return "", err
	}
	link := &resources.AdGroupAsset{
		AdGroup:   in.AdGroup,
		Asset:     in.Asset,
		FieldType: enums.AssetFieldTypeEnum_AssetFieldType(enums.AssetFieldTypeEnum_AssetFieldType_value[in.FieldType]),
	}
	if in.Status != "" {
		link.Status = enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[in.Status])
	}
	resp, err := cl.MutateAdGroupAssets(ctx, &services.MutateAdGroupAssetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AdGroupAssetOperation{{
			Operation: &services.AdGroupAssetOperation_Create{Create: link},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateAdGroupAssetLinkStatus(ctx context.Context, resourceName, status string) error {
	cl, err := c.adGroupAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupAssets")
	if err != nil {
		return err
	}
	link := &resources.AdGroupAsset{
		ResourceName: resourceName,
		Status:       enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[status]),
	}
	_, err = cl.MutateAdGroupAssets(ctx, &services.MutateAdGroupAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAssetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"status"}},
			Operation:  &services.AdGroupAssetOperation_Update{Update: link},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetAdGroupAssetLink(ctx context.Context, resourceName string) (*AdGroupAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupAssets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			ad_group_asset.resource_name,
			ad_group_asset.ad_group,
			ad_group_asset.asset,
			ad_group_asset.field_type,
			ad_group_asset.status
		FROM ad_group_asset
		WHERE ad_group_asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetAdGroupAsset()
	return &AdGroupAssetView{
		ResourceName: x.GetResourceName(),
		AdGroup:      x.GetAdGroup(),
		Asset:        x.GetAsset(),
		FieldType:    x.GetFieldType().String(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveAdGroupAssetLink(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupAssets")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupAssets(ctx, &services.MutateAdGroupAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAssetOperation{{
			Operation: &services.AdGroupAssetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) adGroupAssetClient(ctx context.Context) (*pbclients.AdGroupAssetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adGroupAssets != nil {
		return c.adGroupAssets, nil
	}
	cl, err := pbclients.NewAdGroupAssetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group asset client: %w", err)
	}
	c.adGroupAssets = cl
	return cl, nil
}
