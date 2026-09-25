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

// AssetGroupInput models a Performance Max asset group — the container
// that holds links to text, image, and video assets that fuel a PMax
// campaign's automated creative assembly.
type AssetGroupInput struct {
	CustomerID      string
	Campaign        string // campaign resource name, immutable
	Name            string
	Status          string // ENABLED | PAUSED | REMOVED
	FinalURLs       []string
	FinalMobileURLs []string
	Path1           string
	Path2           string
}

type AssetGroupView struct {
	ResourceName    string
	Campaign        string
	Name            string
	Status          string
	FinalURLs       []string
	FinalMobileURLs []string
	Path1           string
	Path2           string
}

func (c *Client) CreateAssetGroup(ctx context.Context, in AssetGroupInput) (string, error) {
	cl, err := c.assetGroupClient(ctx)
	if err != nil {
		return "", err
	}
	ag := &resources.AssetGroup{
		Campaign:        in.Campaign,
		Name:            in.Name,
		FinalUrls:       in.FinalURLs,
		FinalMobileUrls: in.FinalMobileURLs,
		Path1:           in.Path1,
		Path2:           in.Path2,
	}
	if in.Status != "" {
		ag.Status = enums.AssetGroupStatusEnum_AssetGroupStatus(enums.AssetGroupStatusEnum_AssetGroupStatus_value[in.Status])
	}
	resp, err := cl.MutateAssetGroups(ctx, &services.MutateAssetGroupsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AssetGroupOperation{{
			Operation: &services.AssetGroupOperation_Create{Create: ag},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

// UpdateAssetGroup applies the given paths via FieldMask. `campaign` is
// immutable and must not appear in paths.
func (c *Client) UpdateAssetGroup(ctx context.Context, resourceName string, in AssetGroupInput, paths []string) error {
	cl, err := c.assetGroupClient(ctx)
	if err != nil {
		return err
	}
	ag := &resources.AssetGroup{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "name":
			ag.Name = in.Name
		case "status":
			ag.Status = enums.AssetGroupStatusEnum_AssetGroupStatus(enums.AssetGroupStatusEnum_AssetGroupStatus_value[in.Status])
		case "final_urls":
			ag.FinalUrls = in.FinalURLs
		case "final_mobile_urls":
			ag.FinalMobileUrls = in.FinalMobileURLs
		case "path1":
			ag.Path1 = in.Path1
		case "path2":
			ag.Path2 = in.Path2
		}
	}
	customerID, _, err := ParseResourceName(resourceName, "assetGroups")
	if err != nil {
		return err
	}
	_, err = cl.MutateAssetGroups(ctx, &services.MutateAssetGroupsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetGroupOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.AssetGroupOperation_Update{Update: ag},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetAssetGroup(ctx context.Context, resourceName string) (*AssetGroupView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assetGroups")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			asset_group.resource_name,
			asset_group.campaign,
			asset_group.name,
			asset_group.status,
			asset_group.final_urls,
			asset_group.final_mobile_urls,
			asset_group.path1,
			asset_group.path2
		FROM asset_group
		WHERE asset_group.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetAssetGroup()
	return &AssetGroupView{
		ResourceName:    x.GetResourceName(),
		Campaign:        x.GetCampaign(),
		Name:            x.GetName(),
		Status:          x.GetStatus().String(),
		FinalURLs:       x.GetFinalUrls(),
		FinalMobileURLs: x.GetFinalMobileUrls(),
		Path1:           x.GetPath1(),
		Path2:           x.GetPath2(),
	}, nil
}

func (c *Client) RemoveAssetGroup(ctx context.Context, resourceName string) error {
	cl, err := c.assetGroupClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "assetGroups")
	if err != nil {
		return err
	}
	_, err = cl.MutateAssetGroups(ctx, &services.MutateAssetGroupsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetGroupOperation{{
			Operation: &services.AssetGroupOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) assetGroupClient(ctx context.Context) (*pbclients.AssetGroupClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.assetGroups != nil {
		return c.assetGroups, nil
	}
	cl, err := pbclients.NewAssetGroupClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: asset group client: %w", err)
	}
	c.assetGroups = cl
	return cl, nil
}
