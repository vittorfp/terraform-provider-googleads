package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/resources"
)

func TestCreateAdGroupAd_BuildsResponsiveSearchAdPayload(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	_, err := c.CreateAdGroupAd(context.Background(), AdGroupAdInput{
		CustomerID:   "123",
		AdGroup:      "customers/123/adGroups/9",
		Status:       "PAUSED",
		FinalURLs:    []string{"https://example.com"},
		Headlines:    []string{"A", "B", "C"},
		Descriptions: []string{"first description", "second description"},
		Path1:        "deals",
		Path2:        "today",
	})
	if err != nil {
		t.Fatalf("CreateAdGroupAd: %v", err)
	}
	create := ts.adGroupAdOps[0].GetCreate()
	if got := create.GetAdGroup(); got != "customers/123/adGroups/9" {
		t.Errorf("ad_group = %q", got)
	}
	if got := create.GetStatus().String(); got != "PAUSED" {
		t.Errorf("status = %q", got)
	}
	ad := create.GetAd()
	if got := ad.GetFinalUrls(); len(got) != 1 || got[0] != "https://example.com" {
		t.Errorf("final_urls = %v", got)
	}
	rsa, ok := ad.GetAdData().(*resources.Ad_ResponsiveSearchAd)
	if !ok {
		t.Fatalf("AdData should be ResponsiveSearchAd, got %T", ad.GetAdData())
	}
	headlines := rsa.ResponsiveSearchAd.GetHeadlines()
	if len(headlines) != 3 {
		t.Fatalf("headlines = %d, want 3", len(headlines))
	}
	for i, want := range []string{"A", "B", "C"} {
		if got := headlines[i].GetText(); got != want {
			t.Errorf("headlines[%d] = %q", i, got)
		}
	}
	descriptions := rsa.ResponsiveSearchAd.GetDescriptions()
	if len(descriptions) != 2 || descriptions[0].GetText() != "first description" || descriptions[1].GetText() != "second description" {
		t.Errorf("descriptions = %v", descriptions)
	}
	if got := rsa.ResponsiveSearchAd.GetPath1(); got != "deals" {
		t.Errorf("path1 = %q", got)
	}
	if got := rsa.ResponsiveSearchAd.GetPath2(); got != "today" {
		t.Errorf("path2 = %q", got)
	}
}
