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

// CustomerAssetInput links an asset to the customer (account) so it's
// eligible across every campaign and ad group. The (asset, field_type)
// pair is immutable; status mutates in place.
type CustomerAssetInput struct {
	CustomerID string
	Asset      string
	FieldType  string
	Status     string
}

type CustomerAssetView struct {
	ResourceName string
	Asset        string
	FieldType    string
	Status       string
}

func (c *Client) CreateCustomerAsset(ctx context.Context, in CustomerAssetInput) (string, error) {
	cl, err := c.customerAssetClient(ctx)
	if err != nil {
		return "", err
	}
	link := &resources.CustomerAsset{
		Asset:     in.Asset,
		FieldType: enums.AssetFieldTypeEnum_AssetFieldType(enums.AssetFieldTypeEnum_AssetFieldType_value[in.FieldType]),
	}
	if in.Status != "" {
		link.Status = enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[in.Status])
	}
	resp, err := cl.MutateCustomerAssets(ctx, &services.MutateCustomerAssetsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CustomerAssetOperation{{
			Operation: &services.CustomerAssetOperation_Create{Create: link},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateCustomerAssetStatus(ctx context.Context, resourceName, status string) error {
	cl, err := c.customerAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "customerAssets")
	if err != nil {
		return err
	}
	link := &resources.CustomerAsset{
		ResourceName: resourceName,
		Status:       enums.AssetLinkStatusEnum_AssetLinkStatus(enums.AssetLinkStatusEnum_AssetLinkStatus_value[status]),
	}
	_, err = cl.MutateCustomerAssets(ctx, &services.MutateCustomerAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.CustomerAssetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"status"}},
			Operation:  &services.CustomerAssetOperation_Update{Update: link},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetCustomerAsset(ctx context.Context, resourceName string) (*CustomerAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "customerAssets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			customer_asset.resource_name,
			customer_asset.asset,
			customer_asset.field_type,
			customer_asset.status
		FROM customer_asset
		WHERE customer_asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetCustomerAsset()
	return &CustomerAssetView{
		ResourceName: x.GetResourceName(),
		Asset:        x.GetAsset(),
		FieldType:    x.GetFieldType().String(),
		Status:       x.GetStatus().String(),
	}, nil
}

func (c *Client) RemoveCustomerAsset(ctx context.Context, resourceName string) error {
	cl, err := c.customerAssetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "customerAssets")
	if err != nil {
		return err
	}
	_, err = cl.MutateCustomerAssets(ctx, &services.MutateCustomerAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.CustomerAssetOperation{{
			Operation: &services.CustomerAssetOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) customerAssetClient(ctx context.Context) (*pbclients.CustomerAssetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.customerAssets != nil {
		return c.customerAssets, nil
	}
	cl, err := pbclients.NewCustomerAssetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: customer asset client: %w", err)
	}
	c.customerAssets = cl
	return cl, nil
}
