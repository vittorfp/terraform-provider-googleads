package googleads

import (
	"context"
	"fmt"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// SharedCriterionInput is v1's narrow surface: keyword shared criteria
// only. The API supports more variants (placements, brands, webpages,
// etc.) and they can be added when a real user needs them.
//
// SharedCriteria have no Update operation — every field is immutable;
// changing anything forces resource replacement.
type SharedCriterionInput struct {
	CustomerID  string
	SharedSet   string // resource name
	KeywordText string
	MatchType   string // EXACT | PHRASE | BROAD
}

type SharedCriterionView struct {
	ResourceName string
	SharedSet    string
	KeywordText  string
	MatchType    string
}

func (c *Client) CreateSharedCriterion(ctx context.Context, in SharedCriterionInput) (string, error) {
	cl, err := c.sharedCriterionClient(ctx)
	if err != nil {
		return "", err
	}
	cr := &resources.SharedCriterion{
		SharedSet: StringPtr(in.SharedSet),
		Criterion: &resources.SharedCriterion_Keyword{
			Keyword: &common.KeywordInfo{
				Text:      StringPtr(in.KeywordText),
				MatchType: enums.KeywordMatchTypeEnum_KeywordMatchType(enums.KeywordMatchTypeEnum_KeywordMatchType_value[in.MatchType]),
			},
		},
	}
	resp, err := cl.MutateSharedCriteria(ctx, &services.MutateSharedCriteriaRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.SharedCriterionOperation{{
			Operation: &services.SharedCriterionOperation_Create{Create: cr},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetSharedCriterion(ctx context.Context, resourceName string) (*SharedCriterionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "sharedCriteria")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			shared_criterion.resource_name,
			shared_criterion.shared_set,
			shared_criterion.keyword.text,
			shared_criterion.keyword.match_type
		FROM shared_criterion
		WHERE shared_criterion.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetSharedCriterion()
	return &SharedCriterionView{
		ResourceName: x.GetResourceName(),
		SharedSet:    x.GetSharedSet(),
		KeywordText:  x.GetKeyword().GetText(),
		MatchType:    x.GetKeyword().GetMatchType().String(),
	}, nil
}

func (c *Client) RemoveSharedCriterion(ctx context.Context, resourceName string) error {
	cl, err := c.sharedCriterionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "sharedCriteria")
	if err != nil {
		return err
	}
	_, err = cl.MutateSharedCriteria(ctx, &services.MutateSharedCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.SharedCriterionOperation{{
			Operation: &services.SharedCriterionOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) sharedCriterionClient(ctx context.Context) (*pbclients.SharedCriterionClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sharedCriteria != nil {
		return c.sharedCriteria, nil
	}
	cl, err := pbclients.NewSharedCriterionClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: shared criterion client: %w", err)
	}
	c.sharedCriteria = cl
	return cl, nil
}
