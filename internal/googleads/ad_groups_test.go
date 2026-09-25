package googleads

import (
	"context"
	"testing"
)

func TestCreateAdGroup_FillsRequiredFields(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	bid := int64(250_000)
	_, err := c.CreateAdGroup(context.Background(), AdGroupInput{
		CustomerID:   "123",
		Campaign:     "customers/123/campaigns/1",
		Name:         "tf-test",
		Status:       "ENABLED",
		Type:         "SEARCH_STANDARD",
		CpcBidMicros: &bid,
	})
	if err != nil {
		t.Fatalf("CreateAdGroup: %v", err)
	}
	create := ts.adGroupOps[0].GetCreate()
	if create.GetName() != "tf-test" {
		t.Errorf("name = %q", create.GetName())
	}
	if create.GetCampaign() != "customers/123/campaigns/1" {
		t.Errorf("campaign = %q", create.GetCampaign())
	}
	if got := create.GetStatus().String(); got != "ENABLED" {
		t.Errorf("status = %q", got)
	}
	if got := create.GetType().String(); got != "SEARCH_STANDARD" {
		t.Errorf("type = %q", got)
	}
	if got := create.GetCpcBidMicros(); got != 250_000 {
		t.Errorf("cpc_bid_micros = %d", got)
	}
}

func TestUpdateAdGroup_MaskScopesWhichFieldsShipOnTheWire(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	bid := int64(400_000)
	in := AdGroupInput{
		Name:         "renamed",
		Status:       "PAUSED",
		CpcBidMicros: &bid,
	}
	// Only the `name` path is in the mask. status + cpc_bid_micros must
	// stay zero on the wire — otherwise the API would silently apply
	// them.
	if err := c.UpdateAdGroup(context.Background(), "customers/123/adGroups/42", in, []string{"name"}); err != nil {
		t.Fatalf("UpdateAdGroup: %v", err)
	}
	update := ts.adGroupOps[0].GetUpdate()
	if update.GetName() != "renamed" {
		t.Errorf("name = %q", update.GetName())
	}
	if got := update.GetStatus().String(); got != "UNSPECIFIED" {
		t.Errorf("status leaked: %q", got)
	}
	if update.GetCpcBidMicros() != 0 {
		t.Errorf("cpc_bid_micros leaked: %d", update.GetCpcBidMicros())
	}
}
