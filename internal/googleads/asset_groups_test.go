package googleads

import (
	"context"
	"testing"
)

func TestCreateAssetGroup_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAssetGroup(context.Background(), AssetGroupInput{
		CustomerID: "123",
		Campaign:   "customers/123/campaigns/9",
		Name:       "tf-ag",
		Status:     "PAUSED",
		FinalURLs:  []string{"https://example.com"},
		Path1:      "deals",
	}); err != nil {
		t.Fatalf("CreateAssetGroup: %v", err)
	}
	op := ts.assetGroupOps[0].GetCreate()
	if op.GetCampaign() != "customers/123/campaigns/9" {
		t.Errorf("campaign = %q", op.GetCampaign())
	}
	if op.GetName() != "tf-ag" {
		t.Errorf("name = %q", op.GetName())
	}
	if op.GetStatus().String() != "PAUSED" {
		t.Errorf("status = %q", op.GetStatus().String())
	}
	if got := op.GetFinalUrls(); len(got) != 1 || got[0] != "https://example.com" {
		t.Errorf("final_urls = %v", got)
	}
	if op.GetPath1() != "deals" {
		t.Errorf("path1 = %q", op.GetPath1())
	}
}

func TestUpdateAssetGroup_MaskScopedByPaths(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/assetGroups/42"
	in := AssetGroupInput{
		Name:      "renamed",
		Status:    "PAUSED",
		FinalURLs: []string{"https://example.com/new"},
	}
	// Only `name` and `final_urls` in the paths — status must NOT
	// surface on the wire.
	if err := c.UpdateAssetGroup(context.Background(), rn, in, []string{"name", "final_urls"}); err != nil {
		t.Fatalf("UpdateAssetGroup: %v", err)
	}
	op := ts.assetGroupOps[0]
	if op.GetUpdate().GetName() != "renamed" {
		t.Errorf("name = %q", op.GetUpdate().GetName())
	}
	if op.GetUpdate().GetStatus().String() != "UNSPECIFIED" {
		t.Errorf("status leaked: %q", op.GetUpdate().GetStatus().String())
	}
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 2 {
		t.Errorf("mask = %+v", mask)
	}
}

func TestCreateAssetGroupAsset_FieldType(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateAssetGroupAsset(context.Background(), AssetGroupAssetInput{
		CustomerID: "123",
		AssetGroup: "customers/123/assetGroups/9",
		Asset:      "customers/123/assets/1",
		FieldType:  "HEADLINE",
	}); err != nil {
		t.Fatalf("CreateAssetGroupAsset: %v", err)
	}
	op := ts.assetGroupAssetOps[0].GetCreate()
	if got := op.GetFieldType().String(); got != "HEADLINE" {
		t.Errorf("field_type = %q", got)
	}
}
