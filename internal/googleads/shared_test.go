package googleads

import (
	"context"
	"testing"
)

func TestCreateSharedSet_Wire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateSharedSet(context.Background(), SharedSetInput{
		CustomerID: "123",
		Name:       "Brand negatives",
		Type:       "NEGATIVE_KEYWORDS",
	}); err != nil {
		t.Fatalf("CreateSharedSet: %v", err)
	}
	op := ts.sharedSetOps[0].GetCreate()
	if op.GetName() != "Brand negatives" {
		t.Errorf("name = %q", op.GetName())
	}
	if op.GetType().String() != "NEGATIVE_KEYWORDS" {
		t.Errorf("type = %q", op.GetType().String())
	}
}

func TestUpdateSharedSet_NameOnly(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/sharedSets/42"
	if err := c.UpdateSharedSet(context.Background(), rn, "new name"); err != nil {
		t.Fatalf("UpdateSharedSet: %v", err)
	}
	op := ts.sharedSetOps[0]
	if mask := op.GetUpdateMask(); mask == nil || mask.GetPaths()[0] != "name" {
		t.Errorf("mask = %+v", mask)
	}
}

func TestCreateSharedCriterion_KeywordWire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateSharedCriterion(context.Background(), SharedCriterionInput{
		CustomerID:  "123",
		SharedSet:   "customers/123/sharedSets/9",
		KeywordText: "free",
		MatchType:   "BROAD",
	}); err != nil {
		t.Fatalf("CreateSharedCriterion: %v", err)
	}
	op := ts.sharedCriterionOps[0].GetCreate()
	if op.GetSharedSet() != "customers/123/sharedSets/9" {
		t.Errorf("shared_set = %q", op.GetSharedSet())
	}
	if got := op.GetKeyword().GetText(); got != "free" {
		t.Errorf("keyword text = %q", got)
	}
	if got := op.GetKeyword().GetMatchType().String(); got != "BROAD" {
		t.Errorf("match_type = %q", got)
	}
}

func TestCreateCampaignSharedSet_BothEndpoints(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	if _, err := c.CreateCampaignSharedSet(context.Background(), CampaignSharedSetInput{
		CustomerID: "123",
		Campaign:   "customers/123/campaigns/9",
		SharedSet:  "customers/123/sharedSets/1",
	}); err != nil {
		t.Fatalf("CreateCampaignSharedSet: %v", err)
	}
	op := ts.campaignSharedSetOps[0].GetCreate()
	if op.GetCampaign() != "customers/123/campaigns/9" {
		t.Errorf("campaign = %q", op.GetCampaign())
	}
	if op.GetSharedSet() != "customers/123/sharedSets/1" {
		t.Errorf("shared_set = %q", op.GetSharedSet())
	}
}
