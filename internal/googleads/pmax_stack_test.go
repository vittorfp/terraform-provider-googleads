package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/resources"
)

// TestPMaxStack_EndToEnd drives every resource a Performance Max
// account needs through the client layer and asserts the wire-level
// shape of each Mutate request:
//
//	1. campaign_budget
//	2. conversion_action (Smart bidding needs a goal)
//	3. campaign (PERFORMANCE_MAX channel, TARGET_ROAS bidding)
//	4. text_asset × 5 (headlines, descriptions, long_headline, business_name)
//	5. image_asset (logo)
//	6. asset_group
//	7. asset_group_asset linking each asset to its field_type
//
// This is a unit test, not an acceptance test — it runs in CI on every
// PR without credentials. It catches the structural bugs a real-account
// run would (wrong oneof variants, missing required fields, FieldMask
// typos) before destroying budgets and broken ads in production.
func TestPMaxStack_EndToEnd(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	ctx := context.Background()
	const cid = "1234567890"

	// ── 1. budget ────────────────────────────────────────────────────
	budgetRN, err := c.CreateBudget(ctx, BudgetInput{
		CustomerID:   cid,
		Name:         "tf-pmax-budget",
		AmountMicros: 10_000_000,
	})
	if err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}

	// ── 2. conversion action ────────────────────────────────────────
	primary := true
	_, err = c.CreateConversionAction(ctx, ConversionActionInput{
		CustomerID:     cid,
		Name:           "Website purchase",
		Type:           "WEBPAGE",
		Category:       "PURCHASE",
		CountingType:   "ONE_PER_CLICK",
		PrimaryForGoal: &primary,
	})
	if err != nil {
		t.Fatalf("CreateConversionAction: %v", err)
	}

	// ── 3. PMax campaign with TARGET_ROAS ───────────────────────────
	roas := 3.5
	campaignRN, err := c.CreateCampaign(ctx, CampaignInput{
		CustomerID:             cid,
		Name:                   "tf-pmax",
		AdvertisingChannelType: "PERFORMANCE_MAX",
		Status:                 "PAUSED",
		CampaignBudget:         budgetRN,
		BiddingStrategyType:    "TARGET_ROAS",
		TargetRoas:             &roas,
	})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}

	// ── 4. text assets ──────────────────────────────────────────────
	type textPlan struct {
		name string
		text string
	}
	texts := []textPlan{
		{"headline-1", "Shop the latest deals"},
		{"headline-2", "Discover great products"},
		{"headline-3", "Free shipping over $50"},
		{"description-1", "Curated picks updated daily."},
		{"long-headline", "Save more on the brands you love."},
		{"business-name", "Acme Co."},
	}
	textRNs := make([]string, len(texts))
	for i, tp := range texts {
		rn, err := c.CreateTextAsset(ctx, AssetInput{
			CustomerID: cid,
			Name:       tp.name,
			Text:       tp.text,
		})
		if err != nil {
			t.Fatalf("CreateTextAsset %d: %v", i, err)
		}
		textRNs[i] = rn
	}

	// ── 5. image asset (logo) ───────────────────────────────────────
	// One-pixel PNG, valid header so http.DetectContentType sees it as
	// image/png. Just enough to drive the API call shape.
	logoBytes := pngOnePixel()
	logoRN, err := c.CreateImageAsset(ctx, AssetInput{
		CustomerID: cid,
		Name:       "logo",
		ImageData:  logoBytes,
	})
	if err != nil {
		t.Fatalf("CreateImageAsset: %v", err)
	}

	// ── 6. asset group ──────────────────────────────────────────────
	agRN, err := c.CreateAssetGroup(ctx, AssetGroupInput{
		CustomerID: cid,
		Campaign:   campaignRN,
		Name:       "tf-asset-group",
		Status:     "PAUSED",
		FinalURLs:  []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("CreateAssetGroup: %v", err)
	}

	// ── 7. asset group ↔ asset links ────────────────────────────────
	links := []struct {
		assetRN   string
		fieldType string
	}{
		{textRNs[0], "HEADLINE"},
		{textRNs[1], "HEADLINE"},
		{textRNs[2], "HEADLINE"},
		{textRNs[3], "DESCRIPTION"},
		{textRNs[4], "LONG_HEADLINE"},
		{textRNs[5], "BUSINESS_NAME"},
		{logoRN, "LOGO"},
	}
	for _, l := range links {
		if _, err := c.CreateAssetGroupAsset(ctx, AssetGroupAssetInput{
			CustomerID: cid,
			AssetGroup: agRN,
			Asset:      l.assetRN,
			FieldType:  l.fieldType,
		}); err != nil {
			t.Fatalf("CreateAssetGroupAsset %s: %v", l.fieldType, err)
		}
	}

	// ── assertions: every Mutate request had the right shape ────────

	// Budget op shape.
	if got := len(ts.budgetOps); got != 1 {
		t.Fatalf("budget ops = %d, want 1", got)
	}
	if got := ts.budgetOps[0].GetCreate().GetAmountMicros(); got != 10_000_000 {
		t.Errorf("budget amount_micros = %d", got)
	}

	// Conversion action.
	if got := len(ts.conversionOps); got != 1 {
		t.Fatalf("conversion ops = %d, want 1", got)
	}
	ca := ts.conversionOps[0].GetCreate()
	if got := ca.GetType().String(); got != "WEBPAGE" {
		t.Errorf("conversion type = %q", got)
	}
	if got := ca.GetCategory().String(); got != "PURCHASE" {
		t.Errorf("conversion category = %q", got)
	}
	if !ca.GetPrimaryForGoal() {
		t.Error("PrimaryForGoal should be true")
	}

	// Campaign — PMax + TargetRoas oneof with the right value.
	if got := len(ts.campaignOps); got != 1 {
		t.Fatalf("campaign ops = %d, want 1", got)
	}
	camp := ts.campaignOps[0].GetCreate()
	if got := camp.GetAdvertisingChannelType().String(); got != "PERFORMANCE_MAX" {
		t.Errorf("channel = %q", got)
	}
	if camp.GetCampaignBudget() != budgetRN {
		t.Errorf("campaign_budget = %q, want %q", camp.GetCampaignBudget(), budgetRN)
	}
	tr, ok := camp.GetCampaignBiddingStrategy().(*resources.Campaign_TargetRoas)
	if !ok {
		t.Fatalf("bidding strategy = %T, want *Campaign_TargetRoas", camp.GetCampaignBiddingStrategy())
	}
	if got := tr.TargetRoas.GetTargetRoas(); got != 3.5 {
		t.Errorf("target_roas = %g", got)
	}

	// Six text assets + one image — seven asset ops total.
	if got := len(ts.assetOps); got != 7 {
		t.Fatalf("asset ops = %d, want 7", got)
	}
	for i, op := range ts.assetOps[:6] {
		a := op.GetCreate()
		switch d := a.GetAssetData().(type) {
		case *resources.Asset_TextAsset:
			if got := d.TextAsset.GetText(); got != texts[i].text {
				t.Errorf("text asset[%d] text = %q, want %q", i, got, texts[i].text)
			}
		default:
			t.Errorf("text asset[%d] AssetData = %T, want *Asset_TextAsset", i, d)
		}
	}
	imgOp := ts.assetOps[6].GetCreate()
	imgPayload, ok := imgOp.GetAssetData().(*resources.Asset_ImageAsset)
	if !ok {
		t.Fatalf("image AssetData = %T, want *Asset_ImageAsset", imgOp.GetAssetData())
	}
	if got := imgPayload.ImageAsset.GetMimeType().String(); got != "IMAGE_PNG" {
		t.Errorf("image mime_type = %q", got)
	}
	if got := imgPayload.ImageAsset.GetData(); len(got) == 0 {
		t.Error("image bytes empty")
	}

	// Asset group — PMax campaign referenced; status PAUSED.
	if got := len(ts.assetGroupOps); got != 1 {
		t.Fatalf("asset group ops = %d, want 1", got)
	}
	ag := ts.assetGroupOps[0].GetCreate()
	if ag.GetCampaign() != campaignRN {
		t.Errorf("asset_group.campaign = %q, want %q", ag.GetCampaign(), campaignRN)
	}
	if got := ag.GetStatus().String(); got != "PAUSED" {
		t.Errorf("asset_group.status = %q", got)
	}
	if got := ag.GetFinalUrls(); len(got) != 1 || got[0] != "https://example.com" {
		t.Errorf("final_urls = %v", got)
	}

	// 7 asset-group-asset links, each with the right field_type.
	if got := len(ts.assetGroupAssetOps); got != 7 {
		t.Fatalf("asset_group_asset ops = %d, want 7", got)
	}
	for i, op := range ts.assetGroupAssetOps {
		link := op.GetCreate()
		if got := link.GetFieldType().String(); got != links[i].fieldType {
			t.Errorf("link[%d] field_type = %q, want %q", i, got, links[i].fieldType)
		}
		if link.GetAssetGroup() != agRN {
			t.Errorf("link[%d] asset_group = %q", i, link.GetAssetGroup())
		}
		if link.GetAsset() != links[i].assetRN {
			t.Errorf("link[%d] asset = %q, want %q", i, link.GetAsset(), links[i].assetRN)
		}
	}

	// Compile-time link — common is referenced by the assertions above
	// transitively but the import would be removed if we ever stop
	// covering an oneof.
	_ = common.TextAsset{}

	t.Logf("end-to-end PMax stack: 1 budget + 1 conversion + 1 campaign + 6 text + 1 image + 1 asset group + 7 links → 18 ops, all correct shape")
}

// pngOnePixel returns a minimal valid PNG so http.DetectContentType
// reports image/png and CreateImageAsset doesn't reject the bytes.
func pngOnePixel() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00,
		0x1f, 0x15, 0xc4, 0x89,
		0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
		0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01,
		0x0d, 0x0a, 0x2d, 0xb4,
		0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D',
		0xae, 0x42, 0x60, 0x82,
	}
}
