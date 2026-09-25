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

// CustomerNegativeCriterionInput models a customer-level exclusion.
// Exactly one of the variant fields must be set — the provider layer
// enforces that; the client wires whichever is present into the right
// oneof. v1 covers five most-used variants + content_label_type
// (closes #41). Mobile-app categories, YouTube videos, and placement
// lists land later.
type CustomerNegativeCriterionInput struct {
	CustomerID string

	PlacementURL          string // e.g. http://www.badsite.com
	YoutubeChannelID      string // YouTube channel ID
	MobileApplicationID   string // {platform}-{native_id}, e.g. 1-476943146
	IPAddress             string // IP or CIDR
	NegativeKeywordListID string // resource name of a NEGATIVE_KEYWORDS shared set
	ContentLabelType      string // ContentLabelType enum (e.g. SEXUALLY_SUGGESTIVE, BELOW_THE_FOLD)
}

type CustomerNegativeCriterionView struct {
	ResourceName string
	Type         string
	// Same model on the read side; only one is populated.
	PlacementURL          string
	YoutubeChannelID      string
	MobileApplicationID   string
	IPAddress             string
	NegativeKeywordListID string
	ContentLabelType      string
}

func (c *Client) CreateCustomerNegativeCriterion(ctx context.Context, in CustomerNegativeCriterionInput) (string, error) {
	cl, err := c.customerNegativeCriterionClient(ctx)
	if err != nil {
		return "", err
	}
	cr := &resources.CustomerNegativeCriterion{}
	switch {
	case in.PlacementURL != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_Placement{
			Placement: &common.PlacementInfo{Url: StringPtr(in.PlacementURL)},
		}
	case in.YoutubeChannelID != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_YoutubeChannel{
			YoutubeChannel: &common.YouTubeChannelInfo{ChannelId: StringPtr(in.YoutubeChannelID)},
		}
	case in.MobileApplicationID != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_MobileApplication{
			MobileApplication: &common.MobileApplicationInfo{AppId: StringPtr(in.MobileApplicationID)},
		}
	case in.IPAddress != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_IpBlock{
			IpBlock: &common.IpBlockInfo{IpAddress: StringPtr(in.IPAddress)},
		}
	case in.NegativeKeywordListID != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_NegativeKeywordList{
			NegativeKeywordList: &common.NegativeKeywordListInfo{SharedSet: StringPtr(in.NegativeKeywordListID)},
		}
	case in.ContentLabelType != "":
		cr.Criterion = &resources.CustomerNegativeCriterion_ContentLabel{
			ContentLabel: &common.ContentLabelInfo{
				Type: enums.ContentLabelTypeEnum_ContentLabelType(enums.ContentLabelTypeEnum_ContentLabelType_value[in.ContentLabelType]),
			},
		}
	}
	resp, err := cl.MutateCustomerNegativeCriteria(ctx, &services.MutateCustomerNegativeCriteriaRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CustomerNegativeCriterionOperation{{
			Operation: &services.CustomerNegativeCriterionOperation_Create{Create: cr},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) GetCustomerNegativeCriterion(ctx context.Context, resourceName string) (*CustomerNegativeCriterionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "customerNegativeCriteria")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			customer_negative_criterion.resource_name,
			customer_negative_criterion.type,
			customer_negative_criterion.placement.url,
			customer_negative_criterion.youtube_channel.channel_id,
			customer_negative_criterion.mobile_application.app_id,
			customer_negative_criterion.ip_block.ip_address,
			customer_negative_criterion.negative_keyword_list.shared_set,
			customer_negative_criterion.content_label.type
		FROM customer_negative_criterion
		WHERE customer_negative_criterion.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetCustomerNegativeCriterion()
	return &CustomerNegativeCriterionView{
		ResourceName:          x.GetResourceName(),
		Type:                  x.GetType().String(),
		PlacementURL:          x.GetPlacement().GetUrl(),
		YoutubeChannelID:      x.GetYoutubeChannel().GetChannelId(),
		MobileApplicationID:   x.GetMobileApplication().GetAppId(),
		IPAddress:             x.GetIpBlock().GetIpAddress(),
		NegativeKeywordListID: x.GetNegativeKeywordList().GetSharedSet(),
		ContentLabelType:      contentLabelTypeString(x.GetContentLabel().GetType()),
	}, nil
}

// contentLabelTypeString converts the ContentLabelType enum into the
// API-canonical SCREAMING_SNAKE_CASE string, but returns "" for the
// zero values (UNSPECIFIED/UNKNOWN) so the read side treats them as
// "no content_label set". Otherwise GetType() defaults would make
// every non-ContentLabel CNC look like a ContentLabel one.
func contentLabelTypeString(t enums.ContentLabelTypeEnum_ContentLabelType) string {
	if t == enums.ContentLabelTypeEnum_UNSPECIFIED || t == enums.ContentLabelTypeEnum_UNKNOWN {
		return ""
	}
	return t.String()
}

func (c *Client) RemoveCustomerNegativeCriterion(ctx context.Context, resourceName string) error {
	cl, err := c.customerNegativeCriterionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "customerNegativeCriteria")
	if err != nil {
		return err
	}
	_, err = cl.MutateCustomerNegativeCriteria(ctx, &services.MutateCustomerNegativeCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.CustomerNegativeCriterionOperation{{
			Operation: &services.CustomerNegativeCriterionOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) customerNegativeCriterionClient(ctx context.Context) (*pbclients.CustomerNegativeCriterionClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.customerNegativeCriteria != nil {
		return c.customerNegativeCriteria, nil
	}
	cl, err := pbclients.NewCustomerNegativeCriterionClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: customer negative criterion client: %w", err)
	}
	c.customerNegativeCriteria = cl
	return cl, nil
}
