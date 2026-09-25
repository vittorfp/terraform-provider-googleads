package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// AudienceCriterionInput is the audience/demographic side of
// ad_group_criterion. Lives in its own file (alongside criteria.go which
// holds keywords) to keep each variant family readable.
//
// Exactly one of UserListID, AgeRangeType, GenderType must be set.
// Status and BidModifier are mutable; everything else is RequiresReplace.
type AudienceCriterionInput struct {
	CustomerID  string
	AdGroup     string
	Status      string
	BidModifier *float64
	Negative    *bool

	UserListID    string // user_list resource name (immutable)
	AgeRangeType  string // AGE_RANGE_18_24 .. AGE_RANGE_65_UP | AGE_RANGE_UNDETERMINED
	GenderType    string // MALE | FEMALE | UNDETERMINED
}

type AudienceCriterionView struct {
	ResourceName string
	AdGroup      string
	Status       string
	BidModifier  float64
	Negative     bool

	UserListID   string
	AgeRangeType string
	GenderType   string
}

func (c *Client) CreateAudienceCriterion(ctx context.Context, in AudienceCriterionInput) (string, error) {
	cl, err := c.adGroupCriterionClient(ctx)
	if err != nil {
		return "", err
	}
	cr := &resources.AdGroupCriterion{
		AdGroup: StringPtr(in.AdGroup),
	}
	if in.Status != "" {
		cr.Status = enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus(enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus_value[in.Status])
	}
	if in.Negative != nil {
		cr.Negative = in.Negative
	}
	if in.BidModifier != nil {
		cr.BidModifier = in.BidModifier
	}
	switch {
	case in.UserListID != "":
		cr.Criterion = &resources.AdGroupCriterion_UserList{
			UserList: &common.UserListInfo{UserList: StringPtr(in.UserListID)},
		}
	case in.AgeRangeType != "":
		cr.Criterion = &resources.AdGroupCriterion_AgeRange{
			AgeRange: &common.AgeRangeInfo{
				Type: enums.AgeRangeTypeEnum_AgeRangeType(enums.AgeRangeTypeEnum_AgeRangeType_value[in.AgeRangeType]),
			},
		}
	case in.GenderType != "":
		cr.Criterion = &resources.AdGroupCriterion_Gender{
			Gender: &common.GenderInfo{
				Type: enums.GenderTypeEnum_GenderType(enums.GenderTypeEnum_GenderType_value[in.GenderType]),
			},
		}
	}
	resp, err := cl.MutateAdGroupCriteria(ctx, &services.MutateAdGroupCriteriaRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AdGroupCriterionOperation{{
			Operation: &services.AdGroupCriterionOperation_Create{Create: cr},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateAudienceCriterion(ctx context.Context, resourceName string, in AudienceCriterionInput, paths []string) error {
	cl, err := c.adGroupCriterionClient(ctx)
	if err != nil {
		return err
	}
	cr := &resources.AdGroupCriterion{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "status":
			cr.Status = enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus(enums.AdGroupCriterionStatusEnum_AdGroupCriterionStatus_value[in.Status])
		case "bid_modifier":
			cr.BidModifier = in.BidModifier
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

func (c *Client) GetAudienceCriterion(ctx context.Context, resourceName string) (*AudienceCriterionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			ad_group_criterion.resource_name,
			ad_group_criterion.ad_group,
			ad_group_criterion.status,
			ad_group_criterion.bid_modifier,
			ad_group_criterion.negative,
			ad_group_criterion.user_list.user_list,
			ad_group_criterion.age_range.type,
			ad_group_criterion.gender.type
		FROM ad_group_criterion
		WHERE ad_group_criterion.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetAdGroupCriterion()
	return &AudienceCriterionView{
		ResourceName: x.GetResourceName(),
		AdGroup:      x.GetAdGroup(),
		Status:       x.GetStatus().String(),
		BidModifier:  x.GetBidModifier(),
		Negative:     x.GetNegative(),
		UserListID:   x.GetUserList().GetUserList(),
		AgeRangeType: x.GetAgeRange().GetType().String(),
		GenderType:   x.GetGender().GetType().String(),
	}, nil
}

// RemoveAudienceCriterion piggybacks on the existing RemoveCriterion;
// alias kept here for symmetry with the Create/Update/Get pair.
func (c *Client) RemoveAudienceCriterion(ctx context.Context, resourceName string) error {
	return c.RemoveCriterion(ctx, resourceName)
}
