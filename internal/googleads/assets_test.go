package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/resources"
)

// pngHeader returns the minimum bytes Go's http.DetectContentType
// needs to identify the data as image/png. Shared by tests that don't
// need a fully-decodable PNG.
func pngHeader() []byte {
	return []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06,
	}
}

func TestCreateTextAsset_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateTextAsset(context.Background(), AssetInput{
		CustomerID: "123", Name: "h1", Text: "shop now",
	}); err != nil {
		t.Fatalf("CreateTextAsset: %v", err)
	}
	op := ts.assetOps[0].GetCreate()
	if op.GetName() != "h1" {
		t.Errorf("name = %q", op.GetName())
	}
	ta, ok := op.GetAssetData().(*resources.Asset_TextAsset)
	if !ok {
		t.Fatalf("AssetData = %T", op.GetAssetData())
	}
	if ta.TextAsset.GetText() != "shop now" {
		t.Errorf("text = %q", ta.TextAsset.GetText())
	}
}

func TestCreateImageAsset_DetectsMime(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateImageAsset(context.Background(), AssetInput{
		CustomerID: "123", Name: "logo", ImageData: pngHeader(),
	}); err != nil {
		t.Fatalf("CreateImageAsset: %v", err)
	}
	ia := ts.assetOps[0].GetCreate().GetAssetData().(*resources.Asset_ImageAsset)
	if got := ia.ImageAsset.GetMimeType().String(); got != "IMAGE_PNG" {
		t.Errorf("mime_type = %q", got)
	}
}

func TestCreateImageAsset_RejectsNonImage(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	_, err := c.CreateImageAsset(context.Background(), AssetInput{
		CustomerID: "123", ImageData: []byte("not an image"),
	})
	if err == nil {
		t.Fatal("expected error for non-image bytes")
	}
}

func TestCreateSitelinkAsset_PopulatesAllFields(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateSitelinkAsset(context.Background(), SitelinkAssetInput{
		CustomerID:   "123",
		Name:         "Support",
		LinkText:     "24/7 Support",
		Description1: "Chat anytime",
		Description2: "Reply <5min",
		FinalURLs:    []string{"https://example.com/support"},
	}); err != nil {
		t.Fatalf("CreateSitelinkAsset: %v", err)
	}
	op := ts.assetOps[0].GetCreate()
	if got := op.GetFinalUrls(); len(got) != 1 || got[0] != "https://example.com/support" {
		t.Errorf("final_urls = %v", got)
	}
	sl, ok := op.GetAssetData().(*resources.Asset_SitelinkAsset)
	if !ok {
		t.Fatalf("AssetData = %T", op.GetAssetData())
	}
	if sl.SitelinkAsset.GetLinkText() != "24/7 Support" {
		t.Errorf("link_text = %q", sl.SitelinkAsset.GetLinkText())
	}
	if sl.SitelinkAsset.GetDescription1() != "Chat anytime" || sl.SitelinkAsset.GetDescription2() != "Reply <5min" {
		t.Errorf("descriptions = %q / %q", sl.SitelinkAsset.GetDescription1(), sl.SitelinkAsset.GetDescription2())
	}
}

func TestCreateCalloutAsset_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateCalloutAsset(context.Background(), CalloutAssetInput{
		CustomerID: "123", CalloutText: "Free shipping",
	}); err != nil {
		t.Fatalf("CreateCalloutAsset: %v", err)
	}
	co, ok := ts.assetOps[0].GetCreate().GetAssetData().(*resources.Asset_CalloutAsset)
	if !ok {
		t.Fatalf("AssetData = %T", ts.assetOps[0].GetCreate().GetAssetData())
	}
	if got := co.CalloutAsset.GetCalloutText(); got != "Free shipping" {
		t.Errorf("callout_text = %q", got)
	}
}

func TestCreateStructuredSnippetAsset_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateStructuredSnippetAsset(context.Background(), StructuredSnippetAssetInput{
		CustomerID: "123",
		Header:     "Services",
		Values:     []string{"Consulting", "Training", "Support"},
	}); err != nil {
		t.Fatalf("CreateStructuredSnippetAsset: %v", err)
	}
	ss, ok := ts.assetOps[0].GetCreate().GetAssetData().(*resources.Asset_StructuredSnippetAsset)
	if !ok {
		t.Fatalf("AssetData = %T", ts.assetOps[0].GetCreate().GetAssetData())
	}
	if got := ss.StructuredSnippetAsset.GetHeader(); got != "Services" {
		t.Errorf("header = %q", got)
	}
	if got := ss.StructuredSnippetAsset.GetValues(); len(got) != 3 || got[2] != "Support" {
		t.Errorf("values = %v", got)
	}
}

func TestUpdateAsset_OnlyRenames(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/assets/42"
	if err := c.UpdateAsset(context.Background(), rn, "new name"); err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
	op := ts.assetOps[0]
	if mask := op.GetUpdateMask(); mask == nil || mask.GetPaths()[0] != "name" {
		t.Errorf("update mask = %+v, want only [name]", mask)
	}
	if op.GetUpdate().GetName() != "new name" {
		t.Errorf("name = %q", op.GetUpdate().GetName())
	}
}

func TestCreateCampaignAsset_FieldType(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateCampaignAsset(context.Background(), CampaignAssetInput{
		CustomerID: "123",
		Campaign:   "customers/123/campaigns/9",
		Asset:      "customers/123/assets/1",
		FieldType:  "CALLOUT",
	}); err != nil {
		t.Fatalf("CreateCampaignAsset: %v", err)
	}
	op := ts.campaignAssetOps[0].GetCreate()
	if got := op.GetFieldType().String(); got != "CALLOUT" {
		t.Errorf("field_type = %q", got)
	}
}

func TestUpdateCampaignAssetStatus_MaskScoped(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/campaignAssets/9~1~CALLOUT"
	if err := c.UpdateCampaignAssetStatus(context.Background(), rn, "PAUSED"); err != nil {
		t.Fatalf("UpdateCampaignAssetStatus: %v", err)
	}
	op := ts.campaignAssetOps[0]
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "status" {
		t.Errorf("mask = %+v", mask)
	}
	if got := op.GetUpdate().GetStatus().String(); got != "PAUSED" {
		t.Errorf("status = %q", got)
	}
}

func TestCreateAdGroupAsset_FieldType(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAdGroupAssetLink(context.Background(), AdGroupAssetInput{
		CustomerID: "123",
		AdGroup:    "customers/123/adGroups/7",
		Asset:      "customers/123/assets/2",
		FieldType:  "SITELINK",
	}); err != nil {
		t.Fatalf("CreateAdGroupAssetLink: %v", err)
	}
	op := ts.adGroupAssetOps[0].GetCreate()
	if got := op.GetFieldType().String(); got != "SITELINK" {
		t.Errorf("field_type = %q", got)
	}
	if op.GetAdGroup() != "customers/123/adGroups/7" {
		t.Errorf("ad_group = %q", op.GetAdGroup())
	}
}

func TestUpdateAdGroupAssetStatus_MaskScoped(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/adGroupAssets/7~2~SITELINK"
	if err := c.UpdateAdGroupAssetLinkStatus(context.Background(), rn, "PAUSED"); err != nil {
		t.Fatalf("UpdateAdGroupAssetLinkStatus: %v", err)
	}
	op := ts.adGroupAssetOps[0]
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "status" {
		t.Errorf("mask = %+v", mask)
	}
}

func TestCreateCustomerAsset_FieldType(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateCustomerAsset(context.Background(), CustomerAssetInput{
		CustomerID: "123",
		Asset:      "customers/123/assets/3",
		FieldType:  "CALLOUT",
	}); err != nil {
		t.Fatalf("CreateCustomerAsset: %v", err)
	}
	op := ts.customerAssetOps[0].GetCreate()
	if got := op.GetFieldType().String(); got != "CALLOUT" {
		t.Errorf("field_type = %q", got)
	}
}

func TestUpdateCustomerAssetStatus_MaskScoped(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/customerAssets/3~CALLOUT"
	if err := c.UpdateCustomerAssetStatus(context.Background(), rn, "PAUSED"); err != nil {
		t.Fatalf("UpdateCustomerAssetStatus: %v", err)
	}
	op := ts.customerAssetOps[0]
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "status" {
		t.Errorf("mask = %+v", mask)
	}
}
