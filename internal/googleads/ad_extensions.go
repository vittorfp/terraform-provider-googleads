package googleads

import (
	"context"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// SitelinkAssetInput is the v1 sitelink shape — link text plus optional
// two-line descriptions. Scheduling (start_date/end_date/ad_schedule_targets)
// is out of scope until a real user needs it.
type SitelinkAssetInput struct {
	CustomerID   string
	Name         string
	LinkText     string
	Description1 string
	Description2 string
	FinalURLs    []string
}

type SitelinkAssetView struct {
	ResourceName string
	Name         string
	LinkText     string
	Description1 string
	Description2 string
	FinalURLs    []string
}

func (c *Client) CreateSitelinkAsset(ctx context.Context, in SitelinkAssetInput) (string, error) {
	asset := &resources.Asset{
		FinalUrls: in.FinalURLs,
		AssetData: &resources.Asset_SitelinkAsset{
			SitelinkAsset: &common.SitelinkAsset{
				LinkText:     in.LinkText,
				Description1: in.Description1,
				Description2: in.Description2,
			},
		},
	}
	if in.Name != "" {
		asset.Name = StringPtr(in.Name)
	}
	return c.mutateAssetCreate(ctx, in.CustomerID, asset)
}

func (c *Client) GetSitelinkAsset(ctx context.Context, resourceName string) (*SitelinkAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			asset.resource_name,
			asset.name,
			asset.final_urls,
			asset.sitelink_asset.link_text,
			asset.sitelink_asset.description1,
			asset.sitelink_asset.description2
		FROM asset
		WHERE asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	a := row.GetAsset()
	s := a.GetSitelinkAsset()
	return &SitelinkAssetView{
		ResourceName: a.GetResourceName(),
		Name:         a.GetName(),
		LinkText:     s.GetLinkText(),
		Description1: s.GetDescription1(),
		Description2: s.GetDescription2(),
		FinalURLs:    a.GetFinalUrls(),
	}, nil
}

// CalloutAssetInput is the v1 callout shape — just the text. Scheduling
// fields are deferred to a follow-up.
type CalloutAssetInput struct {
	CustomerID  string
	Name        string
	CalloutText string
}

type CalloutAssetView struct {
	ResourceName string
	Name         string
	CalloutText  string
}

func (c *Client) CreateCalloutAsset(ctx context.Context, in CalloutAssetInput) (string, error) {
	asset := &resources.Asset{
		AssetData: &resources.Asset_CalloutAsset{
			CalloutAsset: &common.CalloutAsset{CalloutText: in.CalloutText},
		},
	}
	if in.Name != "" {
		asset.Name = StringPtr(in.Name)
	}
	return c.mutateAssetCreate(ctx, in.CustomerID, asset)
}

func (c *Client) GetCalloutAsset(ctx context.Context, resourceName string) (*CalloutAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT asset.resource_name, asset.name, asset.callout_asset.callout_text
		FROM asset
		WHERE asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	a := row.GetAsset()
	return &CalloutAssetView{
		ResourceName: a.GetResourceName(),
		Name:         a.GetName(),
		CalloutText:  a.GetCalloutAsset().GetCalloutText(),
	}, nil
}

// StructuredSnippetAssetInput is the v1 snippet shape — header (one of the
// predefined values, see the Ads docs) and 3–10 values.
type StructuredSnippetAssetInput struct {
	CustomerID string
	Name       string
	Header     string
	Values     []string
}

type StructuredSnippetAssetView struct {
	ResourceName string
	Name         string
	Header       string
	Values       []string
}

func (c *Client) CreateStructuredSnippetAsset(ctx context.Context, in StructuredSnippetAssetInput) (string, error) {
	asset := &resources.Asset{
		AssetData: &resources.Asset_StructuredSnippetAsset{
			StructuredSnippetAsset: &common.StructuredSnippetAsset{
				Header: in.Header,
				Values: in.Values,
			},
		},
	}
	if in.Name != "" {
		asset.Name = StringPtr(in.Name)
	}
	return c.mutateAssetCreate(ctx, in.CustomerID, asset)
}

func (c *Client) GetStructuredSnippetAsset(ctx context.Context, resourceName string) (*StructuredSnippetAssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			asset.resource_name,
			asset.name,
			asset.structured_snippet_asset.header,
			asset.structured_snippet_asset.values
		FROM asset
		WHERE asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	a := row.GetAsset()
	s := a.GetStructuredSnippetAsset()
	return &StructuredSnippetAssetView{
		ResourceName: a.GetResourceName(),
		Name:         a.GetName(),
		Header:       s.GetHeader(),
		Values:       s.GetValues(),
	}, nil
}

// MutateAssetsRequest's batching is exposed through mutateAssetCreate in
// assets.go and reused above; that helper already takes the customer ID
// and the prepared *resources.Asset. No extra plumbing needed here.

// pin unused import warning suppression: we depend on services for
// MutateAssetsRequest indirectly via mutateAssetCreate, but Go's linter
// won't see that. The next reference keeps the import live.
var _ = services.MutateAssetsRequest{}
