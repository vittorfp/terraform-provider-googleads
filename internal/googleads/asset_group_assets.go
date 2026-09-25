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

// AssetGroupAssetInput is the M:N join between an asset and an asset group,
// tagged with the field_type that describes how the asset is used
// (HEADLINE, DESCRIPTION, MARKETING_IMAGE, LOGO, etc.). The triple
// (asset_group, asset, field_type) forms the resource's identity and is
// immutable; only `status` is mutable in place.
type AssetGroupAssetInput struct {
	CustomerID string
	AssetGroup string // resource name
	Asset      string // resource name
	FieldType  string // HEADLINE | DESCRIPTION | MARKETING_IMAGE | LOGO | ...
	Status     string // ENABLED | PAUSED | REMOVED — defaults to ENABLED
}

type AssetGroupAssetView struct {
	ResourceName string
	AssetGroup   string
	Asset        string
	FieldType    string
	Status       string
}

func (c *Client) CreateAssetGroupAsset(ctx context.Context, in AssetGroupAssetInput) (string, error) {
	cl, err := c.assetGroupAssetClient(ctx)
	if err != nil {
		return "", err
	}
	link := &resources.AssetGroupAsset{
		AssetGroup: in.AssetGroup,
		Asset:      in.Asset,
		FieldType:  enums.AssetFieldTypeEnum_AssetFieldType(enums.AssetFieldTypeEnum_AssetFieldType_value[in.FieldType]),
	}
	if in.Status != "" {
		link.Status = enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[in.Status])
	}
	resp, err := cl.MutateAssetGroupAssets(ctx, &services.MutateAssetGroupAssetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AssetGroupAssetOperation{{
			Operation: &services.AssetGroupAssetOperation_Create{Create: link},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateAssetGroupAssetStatus(ctx context.Context, resourceName, status string) error {
	cl, err := c.assetGroupAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "assetGroupAssets")
	if err != nil {
		return err
	}
	link := &resources.AssetGroupAsset{
		ResourceName: resourceName,
		Status:       enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[status]),
	}
	_, err = cl.MutateAssetGroupAssets(ctx, &services.MutateAssetGroupAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetGroupAssetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"status"}},
			Operation:  &services.AssetGroupAssetOperation_Update{Update: link},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetAssetGroupAsset(ctx context.Context, resourceName string) (*AssetGroupAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assetGroupAssets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			asset_group_asset.resource_name,
			asset_group_asset.asset_group,
			asset_group_asset.asset,
			asset_group_asset.field_type,
			asset_group_asset.status
		FROM asset_group_asset
		WHERE asset_group_asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetAssetGroupAsset()
	return &AssetGroupAssetView{
		ResourceName: x.GetResourceName(),
		AssetGroup:   x.GetAssetGroup(),
		Asset:        x.GetAsset(),
		FieldType:    x.GetFieldType().String(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveAssetGroupAsset(ctx context.Context, resourceName string) error {
	cl, err := c.assetGroupAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "assetGroupAssets")
	if err != nil {
		return err
	}
	_, err = cl.MutateAssetGroupAssets(ctx, &services.MutateAssetGroupAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetGroupAssetOperation{{
			Operation: &services.AssetGroupAssetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) assetGroupAssetClient(ctx context.Context) (*pbclients.AssetGroupAssetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.assetGroupAssets != nil {
		return c.assetGroupAssets, nil
	}
	cl, err := pbclients.NewAssetGroupAssetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: asset group asset client: %w", err)
	}
	c.assetGroupAssets = cl
	return cl, nil
}
