package googleads

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// CampaignInput is the shape the Terraform layer hands the client.
//
// Bidding strategy params are all optional; only the ones that apply to
// the chosen BiddingStrategyType end up on the wire:
//
//	TARGET_CPA                → TargetCpaMicros, CpcBidCeilingMicros, CpcBidFloorMicros
//	TARGET_ROAS               → TargetRoas,      CpcBidCeilingMicros, CpcBidFloorMicros
//	MAXIMIZE_CONVERSIONS      → TargetCpaMicros (optional), CpcBidCeilingMicros (optional)
//	MAXIMIZE_CONVERSION_VALUE → TargetRoas      (optional), CpcBidCeilingMicros (optional)
//	TARGET_SPEND              → CpcBidCeilingMicros (optional) — the per-click cap that
//	                            the UI exposes as "Maximize Clicks → Maximum CPC".
//	MANUAL_*                  → no params
//
// When the wrong combination is sent the API rejects it; we surface that
// error verbatim through APIError rather than duplicating the validation.
type CampaignInput struct {
	CustomerID             string
	Name                   string
	Status                 string // ENABLED | PAUSED | REMOVED
	AdvertisingChannelType string // SEARCH | DISPLAY | ...
	CampaignBudget         string // resource name "customers/{cid}/campaignBudgets/{id}"
	BiddingStrategyType    string // MANUAL_CPC (default) | TARGET_CPA | TARGET_ROAS | MAXIMIZE_CONVERSIONS | ...

	TargetCpaMicros     *int64
	TargetRoas          *float64
	CpcBidCeilingMicros *int64
	CpcBidFloorMicros   *int64

	NetworkSettings      *NetworkSettings
	GeoTargetTypeSetting *GeoTargetTypeSetting

	// ContainsEuPoliticalAdvertising declares whether the campaign runs EU
	// political ads. The Google Ads API now requires this on create. Empty
	// string = not set by user (the client omits it).
	// CONTAINS_EU_POLITICAL_ADVERTISING | DOES_NOT_CONTAIN_EU_POLITICAL_ADVERTISING
	ContainsEuPoliticalAdvertising string
}

// NetworkSettings mirrors campaign.network_settings. Pointer fields preserve
// "user didn't set it" so the client can build a minimal field mask; setting
// any of them to a non-nil value writes that exact boolean to the API.
type NetworkSettings struct {
	TargetGoogleSearch         *bool
	TargetSearchNetwork        *bool
	TargetContentNetwork       *bool
	TargetPartnerSearchNetwork *bool
}

// GeoTargetTypeSetting mirrors campaign.geo_target_type_setting. Empty string
// means "not set by user" — the client omits the field rather than writing
// the API's default.
type GeoTargetTypeSetting struct {
	PositiveGeoTargetType string // PRESENCE_OR_INTEREST | PRESENCE | SEARCH_INTEREST
	NegativeGeoTargetType string // PRESENCE | SEARCH_INTEREST
}

type CampaignView struct {
	ResourceName           string
	Name                   string
	Status                 string
	AdvertisingChannelType string
	CampaignBudget         string
	BiddingStrategyType    string

	TargetCpaMicros     int64
	TargetRoas          float64
	CpcBidCeilingMicros int64
	CpcBidFloorMicros   int64

	NetworkSettings      NetworkSettingsView
	GeoTargetTypeSetting GeoTargetTypeSettingView

	ContainsEuPoliticalAdvertising string
}

type NetworkSettingsView struct {
	TargetGoogleSearch         bool
	TargetSearchNetwork        bool
	TargetContentNetwork       bool
	TargetPartnerSearchNetwork bool
}

type GeoTargetTypeSettingView struct {
	PositiveGeoTargetType string
	NegativeGeoTargetType string
}

func (c *Client) CreateCampaign(ctx context.Context, in CampaignInput) (string, error) {
	cl, err := c.campaignClient(ctx)
	if err != nil {
		return "", err
	}
	camp := &resources.Campaign{
		Name:                   StringPtr(in.Name),
		AdvertisingChannelType: enums.AdvertisingChannelTypeEnum_AdvertisingChannelType(enums.AdvertisingChannelTypeEnum_AdvertisingChannelType_value[in.AdvertisingChannelType]),
		CampaignBudget:         StringPtr(in.CampaignBudget),
	}
	if in.Status != "" {
		camp.Status = enums.CampaignStatusEnum_CampaignStatus(enums.CampaignStatusEnum_CampaignStatus_value[in.Status])
	}
	applyBiddingStrategy(camp, in)
	applyNetworkSettings(camp, in.NetworkSettings)
	applyGeoTargetTypeSetting(camp, in.GeoTargetTypeSetting)
	applyEuPoliticalAdvertising(camp, in.ContainsEuPoliticalAdvertising)
	resp, err := cl.MutateCampaigns(ctx, &services.MutateCampaignsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CampaignOperation{{
			Operation: &services.CampaignOperation_Create{Create: camp},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateCampaign(ctx context.Context, resourceName string, in CampaignInput, paths []string) error {
	cl, err := c.campaignClient(ctx)
	if err != nil {
		return err
	}
	camp := &resources.Campaign{ResourceName: resourceName}
	// If any bidding-param path was passed we need to set the whole oneof,
	// because field masks on oneof child fields require the parent oneof to
	// be populated in the request. The field mask itself still scopes which
	// child fields to update.
	rebuildStrategy := false
	rebuildNetwork := false
	rebuildGeo := false
	for _, p := range paths {
		switch p {
		case "name":
			camp.Name = StringPtr(in.Name)
		case "status":
			camp.Status = enums.CampaignStatusEnum_CampaignStatus(enums.CampaignStatusEnum_CampaignStatus_value[in.Status])
		case "campaign_budget":
			camp.CampaignBudget = StringPtr(in.CampaignBudget)
		case "network_settings.target_google_search",
			"network_settings.target_search_network",
			"network_settings.target_content_network",
			"network_settings.target_partner_search_network":
			rebuildNetwork = true
		case "geo_target_type_setting.positive_geo_target_type",
			"geo_target_type_setting.negative_geo_target_type":
			rebuildGeo = true
		case "contains_eu_political_advertising":
			applyEuPoliticalAdvertising(camp, in.ContainsEuPoliticalAdvertising)
		case "target_cpa.target_cpa_micros",
			"target_roas.target_roas",
			"maximize_conversions.target_cpa_micros",
			"maximize_conversion_value.target_roas",
			"target_cpa.cpc_bid_ceiling_micros",
			"target_cpa.cpc_bid_floor_micros",
			"target_roas.cpc_bid_ceiling_micros",
			"target_roas.cpc_bid_floor_micros",
			"maximize_conversions.cpc_bid_ceiling_micros",
			"maximize_conversion_value.cpc_bid_ceiling_micros",
			"target_spend.cpc_bid_ceiling_micros":
			rebuildStrategy = true
		}
	}
	if rebuildStrategy {
		applyBiddingStrategy(camp, in)
	}
	if rebuildNetwork {
		applyNetworkSettings(camp, in.NetworkSettings)
	}
	if rebuildGeo {
		applyGeoTargetTypeSetting(camp, in.GeoTargetTypeSetting)
	}
	customerID, _, err := ParseResourceName(resourceName, "campaigns")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaigns(ctx, &services.MutateCampaignsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.CampaignOperation_Update{Update: camp},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetCampaign(ctx context.Context, resourceName string) (*CampaignView, error) {
	customerID, _, err := ParseResourceName(resourceName, "campaigns")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, campaignSelectClause+
		`WHERE campaign.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	view := campaignViewFromRow(row.GetCampaign())
	return &view, nil
}

// campaignSelectClause is the canonical SELECT for the campaign-read
// path. Used by both GetCampaign (resource read) and ListCampaigns
// (tfgen). Kept in one place so a future field add/remove only
// touches one location — and so the API's selectability quirks below
// don't drift between the two queries.
//
// Notable absence: campaign.maximize_conversions.cpc_bid_ceiling_micros
// and campaign.maximize_conversion_value.cpc_bid_ceiling_micros. The
// Google Ads API rejects those in SELECT (they're writable via mutate
// but not queryable). Campaigns using MAXIMIZE_CONVERSIONS or
// MAXIMIZE_CONVERSION_VALUE will read back cpc_bid_ceiling_micros = 0
// even if a non-zero ceiling was actually set; the source of truth in
// that case is the user's HCL, not the API.
const campaignSelectClause = `
	SELECT
		campaign.resource_name,
		campaign.name,
		campaign.status,
		campaign.advertising_channel_type,
		campaign.campaign_budget,
		campaign.bidding_strategy_type,
		campaign.target_cpa.target_cpa_micros,
		campaign.target_cpa.cpc_bid_ceiling_micros,
		campaign.target_cpa.cpc_bid_floor_micros,
		campaign.target_roas.target_roas,
		campaign.target_roas.cpc_bid_ceiling_micros,
		campaign.target_roas.cpc_bid_floor_micros,
		campaign.maximize_conversions.target_cpa_micros,
		campaign.maximize_conversion_value.target_roas,
		campaign.target_spend.cpc_bid_ceiling_micros,
		campaign.network_settings.target_google_search,
		campaign.network_settings.target_search_network,
		campaign.network_settings.target_content_network,
		campaign.network_settings.target_partner_search_network,
		campaign.geo_target_type_setting.positive_geo_target_type,
		campaign.geo_target_type_setting.negative_geo_target_type,
		campaign.contains_eu_political_advertising
	FROM campaign
	`

// campaignViewFromRow flattens the strategy-oneof into the flat
// CampaignView shape the resource layer expects. Shared by Get
// and List so the read-back semantics stay identical.
func campaignViewFromRow(c2 *resources.Campaign) CampaignView {
	view := CampaignView{
		ResourceName:                   c2.GetResourceName(),
		Name:                           c2.GetName(),
		Status:                         c2.GetStatus().String(),
		AdvertisingChannelType:         c2.GetAdvertisingChannelType().String(),
		CampaignBudget:                 c2.GetCampaignBudget(),
		BiddingStrategyType:            c2.GetBiddingStrategyType().String(),
		ContainsEuPoliticalAdvertising: c2.GetContainsEuPoliticalAdvertising().String(),
	}
	switch view.BiddingStrategyType {
	case "TARGET_CPA":
		view.TargetCpaMicros = c2.GetTargetCpa().GetTargetCpaMicros()
		view.CpcBidCeilingMicros = c2.GetTargetCpa().GetCpcBidCeilingMicros()
		view.CpcBidFloorMicros = c2.GetTargetCpa().GetCpcBidFloorMicros()
	case "TARGET_ROAS":
		view.TargetRoas = c2.GetTargetRoas().GetTargetRoas()
		view.CpcBidCeilingMicros = c2.GetTargetRoas().GetCpcBidCeilingMicros()
		view.CpcBidFloorMicros = c2.GetTargetRoas().GetCpcBidFloorMicros()
	case "MAXIMIZE_CONVERSIONS":
		view.TargetCpaMicros = c2.GetMaximizeConversions().GetTargetCpaMicros()
		// cpc_bid_ceiling_micros not selectable for this strategy — see
		// campaignSelectClause comment.
	case "MAXIMIZE_CONVERSION_VALUE":
		view.TargetRoas = c2.GetMaximizeConversionValue().GetTargetRoas()
		// Same selectability gap as MAXIMIZE_CONVERSIONS above.
	case "TARGET_SPEND":
		// "Maximize Clicks" in the UI: the per-click cap is the only
		// knob exposed. target_spend.target_spend_micros is deprecated
		// and no longer settable, so we don't read it.
		view.CpcBidCeilingMicros = c2.GetTargetSpend().GetCpcBidCeilingMicros()
	}
	if ns := c2.GetNetworkSettings(); ns != nil {
		view.NetworkSettings = NetworkSettingsView{
			TargetGoogleSearch:         ns.GetTargetGoogleSearch(),
			TargetSearchNetwork:        ns.GetTargetSearchNetwork(),
			TargetContentNetwork:       ns.GetTargetContentNetwork(),
			TargetPartnerSearchNetwork: ns.GetTargetPartnerSearchNetwork(),
		}
	}
	if g := c2.GetGeoTargetTypeSetting(); g != nil {
		view.GeoTargetTypeSetting = GeoTargetTypeSettingView{
			PositiveGeoTargetType: g.GetPositiveGeoTargetType().String(),
			NegativeGeoTargetType: g.GetNegativeGeoTargetType().String(),
		}
	}
	return view
}

// applyNetworkSettings sets campaign.network_settings from the input. Only
// fields the user provided are written; nil leaves the API value untouched.
// On create, the API supplies channel-appropriate defaults for any field
// the request omits.
func applyNetworkSettings(camp *resources.Campaign, in *NetworkSettings) {
	if in == nil {
		return
	}
	ns := camp.GetNetworkSettings()
	if ns == nil {
		ns = &resources.Campaign_NetworkSettings{}
		camp.NetworkSettings = ns
	}
	if in.TargetGoogleSearch != nil {
		ns.TargetGoogleSearch = in.TargetGoogleSearch
	}
	if in.TargetSearchNetwork != nil {
		ns.TargetSearchNetwork = in.TargetSearchNetwork
	}
	if in.TargetContentNetwork != nil {
		ns.TargetContentNetwork = in.TargetContentNetwork
	}
	if in.TargetPartnerSearchNetwork != nil {
		ns.TargetPartnerSearchNetwork = in.TargetPartnerSearchNetwork
	}
}

func applyGeoTargetTypeSetting(camp *resources.Campaign, in *GeoTargetTypeSetting) {
	if in == nil {
		return
	}
	g := camp.GetGeoTargetTypeSetting()
	if g == nil {
		g = &resources.Campaign_GeoTargetTypeSetting{}
		camp.GeoTargetTypeSetting = g
	}
	if in.PositiveGeoTargetType != "" {
		g.PositiveGeoTargetType = enums.PositiveGeoTargetTypeEnum_PositiveGeoTargetType(
			enums.PositiveGeoTargetTypeEnum_PositiveGeoTargetType_value[in.PositiveGeoTargetType])
	}
	if in.NegativeGeoTargetType != "" {
		g.NegativeGeoTargetType = enums.NegativeGeoTargetTypeEnum_NegativeGeoTargetType(
			enums.NegativeGeoTargetTypeEnum_NegativeGeoTargetType_value[in.NegativeGeoTargetType])
	}
}

// applyEuPoliticalAdvertising sets campaign.contains_eu_political_advertising
// from the input. Empty string = not set by user, so the field is omitted.
func applyEuPoliticalAdvertising(camp *resources.Campaign, status string) {
	if status == "" {
		return
	}
	camp.ContainsEuPoliticalAdvertising = enums.EuPoliticalAdvertisingStatusEnum_EuPoliticalAdvertisingStatus(
		enums.EuPoliticalAdvertisingStatusEnum_EuPoliticalAdvertisingStatus_value[status])
}

func (c *Client) RemoveCampaign(ctx context.Context, resourceName string) error {
	cl, err := c.campaignClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaigns")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaigns(ctx, &services.MutateCampaignsRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignOperation{{
			Operation: &services.CampaignOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

// applyBiddingStrategy sets the appropriate oneof on the campaign for the
// given strategy name, optionally populating the strategy's parameter
// fields from the input. Unknown strategy names leave the oneof unset,
// which makes the API reject the request — surfacing the original
// validation error from Google.
func applyBiddingStrategy(camp *resources.Campaign, in CampaignInput) {
	switch in.BiddingStrategyType {
	case "", "MANUAL_CPC":
		camp.CampaignBiddingStrategy = &resources.Campaign_ManualCpc{ManualCpc: &common.ManualCpc{}}
	case "MANUAL_CPM":
		camp.CampaignBiddingStrategy = &resources.Campaign_ManualCpm{ManualCpm: &common.ManualCpm{}}
	case "MANUAL_CPV":
		camp.CampaignBiddingStrategy = &resources.Campaign_ManualCpv{ManualCpv: &common.ManualCpv{}}
	case "TARGET_SPEND":
		s := &common.TargetSpend{}
		if in.CpcBidCeilingMicros != nil {
			s.CpcBidCeilingMicros = in.CpcBidCeilingMicros
		}
		camp.CampaignBiddingStrategy = &resources.Campaign_TargetSpend{TargetSpend: s}
	case "TARGET_CPA":
		s := &common.TargetCpa{}
		if in.TargetCpaMicros != nil {
			s.TargetCpaMicros = in.TargetCpaMicros
		}
		if in.CpcBidCeilingMicros != nil {
			s.CpcBidCeilingMicros = in.CpcBidCeilingMicros
		}
		if in.CpcBidFloorMicros != nil {
			s.CpcBidFloorMicros = in.CpcBidFloorMicros
		}
		camp.CampaignBiddingStrategy = &resources.Campaign_TargetCpa{TargetCpa: s}
	case "TARGET_ROAS":
		s := &common.TargetRoas{}
		if in.TargetRoas != nil {
			s.TargetRoas = in.TargetRoas
		}
		if in.CpcBidCeilingMicros != nil {
			s.CpcBidCeilingMicros = in.CpcBidCeilingMicros
		}
		if in.CpcBidFloorMicros != nil {
			s.CpcBidFloorMicros = in.CpcBidFloorMicros
		}
		camp.CampaignBiddingStrategy = &resources.Campaign_TargetRoas{TargetRoas: s}
	case "MAXIMIZE_CONVERSIONS":
		s := &common.MaximizeConversions{}
		if in.TargetCpaMicros != nil {
			s.TargetCpaMicros = *in.TargetCpaMicros
		}
		if in.CpcBidCeilingMicros != nil {
			s.CpcBidCeilingMicros = *in.CpcBidCeilingMicros
		}
		camp.CampaignBiddingStrategy = &resources.Campaign_MaximizeConversions{MaximizeConversions: s}
	case "MAXIMIZE_CONVERSION_VALUE":
		s := &common.MaximizeConversionValue{}
		if in.TargetRoas != nil {
			s.TargetRoas = *in.TargetRoas
		}
		if in.CpcBidCeilingMicros != nil {
			s.CpcBidCeilingMicros = *in.CpcBidCeilingMicros
		}
		camp.CampaignBiddingStrategy = &resources.Campaign_MaximizeConversionValue{MaximizeConversionValue: s}
	}
}
