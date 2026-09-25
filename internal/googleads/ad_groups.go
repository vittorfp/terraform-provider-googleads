package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

type AdGroupInput struct {
	CustomerID   string
	Campaign     string // resource name "customers/{cid}/campaigns/{id}"
	Name         string
	Status       string // ENABLED | PAUSED | REMOVED
	Type         string // SEARCH_STANDARD | DISPLAY_STANDARD | ...
	CpcBidMicros *int64
}

type AdGroupView struct {
	ResourceName string
	Campaign     string
	Name         string
	Status       string
	Type         string
	CpcBidMicros int64
}

func (c *Client) CreateAdGroup(ctx context.Context, in AdGroupInput) (string, error) {
	cl, err := c.adGroupClient(ctx)
	if err != nil {
		return "", err
	}
	ag := &resources.AdGroup{
		Name:     StringPtr(in.Name),
		Campaign: StringPtr(in.Campaign),
	}
	if in.Status != "" {
		ag.Status = enums.AdGroupStatusEnum_AdGroupStatus(enums.AdGroupStatusEnum_AdGroupStatus_value[in.Status])
	}
	if in.Type != "" {
		ag.Type = enums.AdGroupTypeEnum_AdGroupType(enums.AdGroupTypeEnum_AdGroupType_value[in.Type])
	}
	if in.CpcBidMicros != nil {
		ag.CpcBidMicros = in.CpcBidMicros
	}
	resp, err := cl.MutateAdGroups(ctx, &services.MutateAdGroupsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AdGroupOperation{{
			Operation: &services.AdGroupOperation_Create{Create: ag},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateAdGroup(ctx context.Context, resourceName string, in AdGroupInput, paths []string) error {
	cl, err := c.adGroupClient(ctx)
	if err != nil {
		return err
	}
	ag := &resources.AdGroup{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "name":
			ag.Name = StringPtr(in.Name)
		case "status":
			ag.Status = enums.AdGroupStatusEnum_AdGroupStatus(enums.AdGroupStatusEnum_AdGroupStatus_value[in.Status])
		case "cpc_bid_micros":
			ag.CpcBidMicros = in.CpcBidMicros
		}
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroups")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroups(ctx, &services.MutateAdGroupsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.AdGroupOperation_Update{Update: ag},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetAdGroup(ctx context.Context, resourceName string) (*AdGroupView, error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroups")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			ad_group.resource_name,
			ad_group.campaign,
			ad_group.name,
			ad_group.status,
			ad_group.type,
			ad_group.cpc_bid_micros
		FROM ad_group
		WHERE ad_group.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	g := row.GetAdGroup()
	return &AdGroupView{
		ResourceName: g.GetResourceName(),
		Campaign:     g.GetCampaign(),
		Name:         g.GetName(),
		Status:       g.GetStatus().String(),
		Type:         g.GetType().String(),
		CpcBidMicros: g.GetCpcBidMicros(),
	}, nil
}

func (c *Client) RemoveAdGroup(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroups")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroups(ctx, &services.MutateAdGroupsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupOperation{{
			Operation: &services.AdGroupOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}
