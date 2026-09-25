package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// CriterionInput is the v1 keyword shape. Other criterion types
// (placements, audience, etc.) are deliberately excluded.
type CriterionInput struct {
	CustomerID   string
	AdGroup      string // resource name
	KeywordText  string
	MatchType    string // EXACT | PHRASE | BROAD
	Status       string // ENABLED | PAUSED | REMOVED
	Negative     *bool
	CpcBidMicros *int64
}

type CriterionView struct {
	ResourceName string
	AdGroup      string
	KeywordText  string
	MatchType    string
	Status       string
	Negative     bool
	CpcBidMicros int64
}

func (c *Client) CreateCriterion(ctx context.Context, in CriterionInput) (string, error) {
	cr := &resources.AdGroupCriterion{
		AdGroup: StringPtr(in.AdGroup),
		Criterion: &resources.AdGroupCriterion_Keyword{
			Keyword: &common.KeywordInfo{
				Text:      StringPtr(in.KeywordText),
				MatchType: enums.KeywordMatchTypeEnum_KeywordMatchType(enums.KeywordMatchTypeEnum_KeywordMatchType_value[in.MatchType]),
			},
		},
	}
	if in.Status != "" {
		cr.Status = enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus(enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus_value[in.Status])
	}
	if in.Negative != nil {
		cr.Negative = in.Negative
	}
	if in.CpcBidMicros != nil {
		cr.CpcBidMicros = in.CpcBidMicros
	}
	return c.mutateAdGroupCriteriaCreate(ctx, in.CustomerID, &services.AdGroupCriterionOperation{
		Operation: &services.AdGroupCriterionOperation_Create{Create: cr},
	})
}

func (c *Client) UpdateCriterion(ctx context.Context, resourceName string, in CriterionInput, paths []string) error {
	cl, err := c.adGroupCriterionClient(ctx)
	if err != nil {
		return err
	}
	cr := &resources.AdGroupCriterion{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "status":
			cr.Status = enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus(enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus_value[in.Status])
		case "cpc_bid_micros":
			cr.CpcBidMicros = in.CpcBidMicros
		}
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupCriteria(ctx, &services.MutateAdGroupCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.AdGroupCriterionOperation_Update{Update: cr},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetCriterion(ctx context.Context, resourceName string) (*CriterionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			ad_group_criterion.resource_name,
			ad_group_criterion.ad_group,
			ad_group_criterion.status,
			ad_group_criterion.negative,
			ad_group_criterion.cpc_bid_micros,
			ad_group_criterion.keyword.text,
			ad_group_criterion.keyword.match_type
		FROM ad_group_criterion
		WHERE ad_group_criterion.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetAdGroupCriterion()
	return &CriterionView{
		ResourceName: x.GetResourceName(),
		AdGroup:      x.GetAdGroup(),
		Status:       x.GetStatus().String(),
		Negative:     x.GetNegative(),
		CpcBidMicros: x.GetCpcBidMicros(),
		KeywordText:  x.GetKeyword().GetText(),
		MatchType:    x.GetKeyword().GetMatchType().String(),
	}, nil
}

func (c *Client) RemoveCriterion(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupCriterionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupCriteria(ctx, &services.MutateAdGroupCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionOperation{{
			Operation: &services.AdGroupCriterionOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}
