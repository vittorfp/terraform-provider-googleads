package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// AdGroupAdInput models a Responsive Search Ad living under an ad group. v1
// scope is RSAs only — every other ad type lives outside the surface area.
type AdGroupAdInput struct {
	CustomerID   string
	AdGroup      string // resource name
	Status       string // ENABLED | PAUSED | REMOVED
	FinalURLs    []string
	Headlines    []string
	Descriptions []string
	Path1        string
	Path2        string
}

type AdGroupAdView struct {
	ResourceName string
	AdGroup      string
	Status       string
	FinalURLs    []string
	Headlines    []string
	Descriptions []string
	Path1        string
	Path2        string
}

func (c *Client) CreateAdGroupAd(ctx context.Context, in AdGroupAdInput) (string, error) {
	cl, err := c.adGroupAdClient(ctx)
	if err != nil {
		return "", err
	}
	aga := &resources.AdGroupAd{
		AdGroup: StringPtr(in.AdGroup),
		Ad: &resources.Ad{
			FinalUrls: in.FinalURLs,
			AdData: &resources.Ad_ResponsiveSearchAd{
				ResponsiveSearchAd: buildRSA(in),
			},
		},
	}
	if in.Status != "" {
		aga.Status = enums.AdGroupAdStatusEnum_AdGroupAdStatus(enums.AdGroupAdStatusEnum_AdGroupAdStatus_value[in.Status])
	}
	resp, err := cl.MutateAdGroupAds(ctx, &services.MutateAdGroupAdsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.AdGroupAdOperation{{
			Operation: &services.AdGroupAdOperation_Create{Create: aga},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

// UpdateAdGroupAd supports a narrow set of mutable fields. Per the Ads API,
// the ad itself is immutable once created — only the AdGroupAd wrapper's
// status can be flipped without re-creating the ad. Callers that need to
// change headlines/descriptions/URLs must let Terraform replace the resource.
func (c *Client) UpdateAdGroupAd(ctx context.Context, resourceName string, in AdGroupAdInput, paths []string) error {
	cl, err := c.adGroupAdClient(ctx)
	if err != nil {
		return err
	}
	aga := &resources.AdGroupAd{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "status":
			aga.Status = enums.AdGroupAdStatusEnum_AdGroupAdStatus(enums.AdGroupAdStatusEnum_AdGroupAdStatus_value[in.Status])
		}
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupAds")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupAds(ctx, &services.MutateAdGroupAdsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAdOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.AdGroupAdOperation_Update{Update: aga},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetAdGroupAd(ctx context.Context, resourceName string) (*AdGroupAdView, error) {
	customerID, _, err := ParseResourceName(resourceName, "adGroupAds")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			ad_group_ad.resource_name,
			ad_group_ad.ad_group,
			ad_group_ad.status,
			ad_group_ad.ad.final_urls,
			ad_group_ad.ad.responsive_search_ad.headlines,
			ad_group_ad.ad.responsive_search_ad.descriptions,
			ad_group_ad.ad.responsive_search_ad.path1,
			ad_group_ad.ad.responsive_search_ad.path2
		FROM ad_group_ad
		WHERE ad_group_ad.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	a := row.GetAdGroupAd()
	rsa := a.GetAd().GetResponsiveSearchAd()
	out := &AdGroupAdView{
		ResourceName: a.GetResourceName(),
		AdGroup:      a.GetAdGroup(),
		Status:       a.GetStatus().String(),
		FinalURLs:    a.GetAd().GetFinalUrls(),
		Path1:        rsa.GetPath1(),
		Path2:        rsa.GetPath2(),
	}
	for _, h := range rsa.GetHeadlines() {
		out.Headlines = append(out.Headlines, h.GetText())
	}
	for _, d := range rsa.GetDescriptions() {
		out.Descriptions = append(out.Descriptions, d.GetText())
	}
	return out, nil
}

func (c *Client) RemoveAdGroupAd(ctx context.Context, resourceName string) error {
	cl, err := c.adGroupAdClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "adGroupAds")
	if err != nil {
		return err
	}
	_, err = cl.MutateAdGroupAds(ctx, &services.MutateAdGroupAdsRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupAdOperation{{
			Operation: &services.AdGroupAdOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func buildRSA(in AdGroupAdInput) *common.ResponsiveSearchAdInfo {
	rsa := &common.ResponsiveSearchAdInfo{}
	for _, h := range in.Headlines {
		rsa.Headlines = append(rsa.Headlines, &common.AdTextAsset{Text: StringPtr(h)})
	}
	for _, d := range in.Descriptions {
		rsa.Descriptions = append(rsa.Descriptions, &common.AdTextAsset{Text: StringPtr(d)})
	}
	if in.Path1 != "" {
		rsa.Path1 = StringPtr(in.Path1)
	}
	if in.Path2 != "" {
		rsa.Path2 = StringPtr(in.Path2)
	}
	return rsa
}
