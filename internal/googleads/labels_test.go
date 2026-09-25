package googleads

import (
	"context"
	"testing"
)

func TestCreateLabel_WithTextLabel(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateLabel(context.Background(), LabelInput{
		CustomerID:      "123",
		Name:            "High priority",
		BackgroundColor: "#FF0000",
		Description:     "Active campaigns",
	}); err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	op := ts.labelOps[0].GetCreate()
	if op.GetName() != "High priority" {
		t.Errorf("name = %q", op.GetName())
	}
	tl := op.GetTextLabel()
	if tl == nil {
		t.Fatal("text_label should be populated")
	}
	if tl.GetBackgroundColor() != "#FF0000" {
		t.Errorf("background_color = %q", tl.GetBackgroundColor())
	}
	if tl.GetDescription() != "Active campaigns" {
		t.Errorf("description = %q", tl.GetDescription())
	}
}

func TestUpdateLabel_OnlyMaskedFieldsShipped(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/labels/42"
	in := LabelInput{
		Name:        "renamed",
		Description: "ignored because not in paths",
	}
	if err := c.UpdateLabel(context.Background(), rn, in, []string{"name"}); err != nil {
		t.Fatalf("UpdateLabel: %v", err)
	}
	op := ts.labelOps[0]
	if op.GetUpdate().GetName() != "renamed" {
		t.Errorf("name = %q", op.GetUpdate().GetName())
	}
	if op.GetUpdate().GetTextLabel() != nil {
		t.Error("text_label should not be populated (path not in mask)")
	}
}

// The four *_label link resources share Create + Remove only. The
// tests below confirm the joining fields land correctly on the wire.

func TestCreateCampaignLabel_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateCampaignLabel(context.Background(), "123", "customers/123/campaigns/9", "customers/123/labels/1"); err != nil {
		t.Fatalf("CreateCampaignLabel: %v", err)
	}
	op := ts.campaignLabelOps[0].GetCreate()
	if op.GetCampaign() != "customers/123/campaigns/9" || op.GetLabel() != "customers/123/labels/1" {
		t.Errorf("wrong endpoints: campaign=%q label=%q", op.GetCampaign(), op.GetLabel())
	}
}

func TestCreateAdGroupLabel_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAdGroupLabel(context.Background(), "123", "customers/123/adGroups/9", "customers/123/labels/1"); err != nil {
		t.Fatalf("CreateAdGroupLabel: %v", err)
	}
	op := ts.adGroupLabelOps[0].GetCreate()
	if op.GetAdGroup() != "customers/123/adGroups/9" || op.GetLabel() != "customers/123/labels/1" {
		t.Errorf("wrong endpoints: ad_group=%q label=%q", op.GetAdGroup(), op.GetLabel())
	}
}

func TestCreateAdGroupAdLabel_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAdGroupAdLabel(context.Background(), "123", "customers/123/adGroupAds/9~10", "customers/123/labels/1"); err != nil {
		t.Fatalf("CreateAdGroupAdLabel: %v", err)
	}
	op := ts.adGroupAdLabelOps[0].GetCreate()
	if op.GetAdGroupAd() != "customers/123/adGroupAds/9~10" || op.GetLabel() != "customers/123/labels/1" {
		t.Errorf("wrong endpoints: ad_group_ad=%q label=%q", op.GetAdGroupAd(), op.GetLabel())
	}
}

func TestCreateAdGroupCriterionLabel_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAdGroupCriterionLabel(context.Background(), "123", "customers/123/adGroupCriteria/9~10", "customers/123/labels/1"); err != nil {
		t.Fatalf("CreateAdGroupCriterionLabel: %v", err)
	}
	op := ts.adGroupCriterionLabelOps[0].GetCreate()
	if op.GetAdGroupCriterion() != "customers/123/adGroupCriteria/9~10" || op.GetLabel() != "customers/123/labels/1" {
		t.Errorf("wrong endpoints: ad_group_criterion=%q label=%q", op.GetAdGroupCriterion(), op.GetLabel())
	}
}
