package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/resources"
)

func TestCreateCampaignCriterion_DispatchesVariant(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	hour9, hour18 := int32(9), int32(18)
	cases := []struct {
		name string
		in   CampaignCriterionInput
		ok   func(any) bool
	}{
		{
			"location",
			CampaignCriterionInput{CustomerID: "123", Campaign: "customers/123/campaigns/9", LocationID: "geoTargetConstants/2840"},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_Location); return ok },
		},
		{
			"language",
			CampaignCriterionInput{CustomerID: "123", Campaign: "customers/123/campaigns/9", LanguageID: "languageConstants/1014"},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_Language); return ok },
		},
		{
			"device",
			CampaignCriterionInput{CustomerID: "123", Campaign: "customers/123/campaigns/9", DeviceType: "MOBILE", BidModifier: 1.25},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_Device); return ok },
		},
		{
			"ip",
			CampaignCriterionInput{CustomerID: "123", Campaign: "customers/123/campaigns/9", IPAddress: "1.2.3.4"},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_IpBlock); return ok },
		},
		{
			"negative_keyword",
			CampaignCriterionInput{
				CustomerID:       "123",
				Campaign:         "customers/123/campaigns/9",
				KeywordText:      "free download",
				KeywordMatchType: "PHRASE",
				Negative:         func() *bool { b := true; return &b }(),
			},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_Keyword); return ok },
		},
		{
			"ad_schedule",
			CampaignCriterionInput{
				CustomerID: "123", Campaign: "customers/123/campaigns/9",
				AdScheduleDayOfWeek:   "MONDAY",
				AdScheduleStartHour:   &hour9,
				AdScheduleEndHour:     &hour18,
				AdScheduleStartMinute: "ZERO",
				AdScheduleEndMinute:   "ZERO",
			},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_AdSchedule); return ok },
		},
		{
			"proximity",
			CampaignCriterionInput{
				CustomerID: "123", Campaign: "customers/123/campaigns/9",
				ProximityLatitudeSet:     true,
				ProximityLatitude:        -19.911,
				ProximityLongitude:       -43.985,
				ProximityRadius:          7.0,
				ProximityRadiusUnits:     "KILOMETERS",
				ProximityAddressCountry:  "BR",
				ProximityAddressCityName: "Belo Horizonte",
			},
			func(v any) bool { _, ok := v.(*resources.CampaignCriterion_Proximity); return ok },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Fresh server per sub-test so the ops slice resets.
			ts := newTestServer(t)
			c := ts.newClient(t)
			if _, err := c.CreateCampaignCriterion(context.Background(), tc.in); err != nil {
				t.Fatalf("Create: %v", err)
			}
			variant := ts.campaignCriterionOps[0].GetCreate().GetCriterion()
			if !tc.ok(variant) {
				t.Errorf("variant = %T", variant)
			}
		})
	}
	_ = ts
	_ = c
}

func TestCreateCampaignCriterion_ProximityProtoFields(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	in := CampaignCriterionInput{
		CustomerID: "123", Campaign: "customers/123/campaigns/9",
		ProximityLatitudeSet:     true,
		ProximityLatitude:        -19.911,
		ProximityLongitude:       -43.985,
		ProximityRadius:          7.5,
		ProximityRadiusUnits:     "KILOMETERS",
		ProximityAddressCountry:  "BR",
		ProximityAddressCityName: "Belo Horizonte",
	}
	if _, err := c.CreateCampaignCriterion(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	variant := ts.campaignCriterionOps[0].GetCreate().GetCriterion()
	p, ok := variant.(*resources.CampaignCriterion_Proximity)
	if !ok {
		t.Fatalf("variant = %T", variant)
	}
	info := p.Proximity
	if got, want := info.GetGeoPoint().GetLatitudeInMicroDegrees(), int32(-19911000); got != want {
		t.Errorf("latitude micro = %d, want %d", got, want)
	}
	if got, want := info.GetGeoPoint().GetLongitudeInMicroDegrees(), int32(-43985000); got != want {
		t.Errorf("longitude micro = %d, want %d", got, want)
	}
	if got, want := info.GetRadius(), 7.5; got != want {
		t.Errorf("radius = %g, want %g", got, want)
	}
	if got, want := info.GetRadiusUnits().String(), "KILOMETERS"; got != want {
		t.Errorf("radius_units = %q, want %q", got, want)
	}
	if got, want := info.GetAddress().GetCountryCode(), "BR"; got != want {
		t.Errorf("country = %q", got)
	}
	if got, want := info.GetAddress().GetCityName(), "Belo Horizonte"; got != want {
		t.Errorf("city = %q", got)
	}
}

func TestUpdateCampaignCriterion_StatusAndBidModifier(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/campaignCriteria/9~10"
	in := CampaignCriterionInput{
		Status:      "PAUSED",
		BidModifier: 1.5,
	}
	if err := c.UpdateCampaignCriterion(context.Background(), rn, in, []string{"status", "bid_modifier"}); err != nil {
		t.Fatalf("UpdateCampaignCriterion: %v", err)
	}
	op := ts.campaignCriterionOps[0]
	if op.GetUpdate().GetStatus().String() != "PAUSED" {
		t.Errorf("status = %q", op.GetUpdate().GetStatus().String())
	}
	if op.GetUpdate().GetBidModifier() != 1.5 {
		t.Errorf("bid_modifier = %g", op.GetUpdate().GetBidModifier())
	}
}
