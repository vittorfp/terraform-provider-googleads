package googleads

import (
	"context"
	"fmt"
	"math"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// CampaignCriterionInput models a campaign-scoped criterion. v1 covers
// the five most-used variants: location, language, device, ad_schedule,
// and ip_block. Everything else from the proto's ~35-way oneof lands
// when a real user needs it.
//
// Mutable: Status, BidModifier (per the API).
// Immutable: campaign_id, negative, and all variant-specific fields.
type CampaignCriterionInput struct {
	CustomerID  string
	Campaign    string
	Status      string  // ENABLED | PAUSED | REMOVED
	BidModifier float32 // 0 to opt out (Device only); 0.1–10.0 otherwise
	Negative    *bool

	// Exactly one of the variants must be set on create.
	LocationID string // geo_target_constant resource name, e.g. geoTargetConstants/2840
	LanguageID string // language_constant resource name
	DeviceType string // MOBILE | TABLET | DESKTOP | CONNECTED_TV | OTHER
	IPAddress  string

	// Keyword variant: typically used with Negative = true to add a
	// campaign-level negative keyword applied to every ad group.
	KeywordText      string // the keyword text
	KeywordMatchType string // EXACT | PHRASE | BROAD

	// AdSchedule fields — all five required together when scheduling.
	AdScheduleDayOfWeek   string // MONDAY..SUNDAY
	AdScheduleStartHour   *int32 // 0–23
	AdScheduleEndHour     *int32 // 0–24
	AdScheduleStartMinute string // ZERO | FIFTEEN | THIRTY | FORTY_FIVE
	AdScheduleEndMinute   string // ZERO | FIFTEEN | THIRTY | FORTY_FIVE

	// Proximity (radius targeting). Lat/Lng/Radius/RadiusUnits required
	// together; Address* is an optional label that does not affect serving.
	// Variant is selected when ProximityLatitudeSet is true (callers should
	// set the bool explicitly so a literal 0.0 latitude can be expressed).
	ProximityLatitudeSet     bool
	ProximityLatitude        float64 // degrees
	ProximityLongitude       float64 // degrees
	ProximityRadius          float64 // KM or MI per ProximityRadiusUnits
	ProximityRadiusUnits     string  // KILOMETERS | MILES
	ProximityAddressCountry  string  // ISO 3166-1 alpha-2 (label only)
	ProximityAddressCityName string  // label only
}

type CampaignCriterionView struct {
	ResourceName string
	Campaign     string
	Status       string
	Type         string
	BidModifier  float32
	Negative     bool

	LocationID string
	LanguageID string
	DeviceType string
	IPAddress  string

	KeywordText      string
	KeywordMatchType string

	AdScheduleDayOfWeek   string
	AdScheduleStartHour   int32
	AdScheduleEndHour     int32
	AdScheduleStartMinute string
	AdScheduleEndMinute   string

	// HasProximity is true when the criterion is a proximity variant. The
	// API returns 0 for unset micro_degrees, which is indistinguishable
	// from a literal 0°,0° point, so callers must consult this flag.
	HasProximity             bool
	ProximityLatitude        float64
	ProximityLongitude       float64
	ProximityRadius          float64
	ProximityRadiusUnits     string
	ProximityAddressCountry  string
	ProximityAddressCityName string
}

func (c *Client) CreateCampaignCriterion(ctx context.Context, in CampaignCriterionInput) (string, error) {
	cl, err := c.campaignCriterionClient(ctx)
	if err != nil {
		return "", err
	}
	cr := &resources.CampaignCriterion{
		Campaign: StringPtr(in.Campaign),
	}
	if in.Status != "" {
		cr.Status = enums.CampaignCriterionStatusEnum_CampaignCriterionStatus(enums.CampaignCriterionStatusEnum_CampaignCriterionStatus_value[in.Status])
	}
	if in.BidModifier != 0 {
		bm := in.BidModifier
		cr.BidModifier = &bm
	}
	if in.Negative != nil {
		cr.Negative = in.Negative
	}
	applyCampaignCriterionVariant(cr, in)
	resp, err := cl.MutateCampaignCriteria(ctx, &services.MutateCampaignCriteriaRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.CampaignCriterionOperation{{
			Operation: &services.CampaignCriterionOperation_Create{Create: cr},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

// UpdateCampaignCriterion supports status and bid_modifier only — the
// variant fields and `negative` are immutable per the API.
func (c *Client) UpdateCampaignCriterion(ctx context.Context, resourceName string, in CampaignCriterionInput, paths []string) error {
	cl, err := c.campaignCriterionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignCriteria")
	if err != nil {
		return err
	}
	cr := &resources.CampaignCriterion{ResourceName: resourceName}
	for _, p := range paths {
		switch p {
		case "status":
			cr.Status = enums.CampaignCriterionStatusEnum_CampaignCriterionStatus(enums.CampaignCriterionStatusEnum_CampaignCriterionStatus_value[in.Status])
		case "bid_modifier":
			bm := in.BidModifier
			cr.BidModifier = &bm
		}
	}
	_, err = cl.MutateCampaignCriteria(ctx, &services.MutateCampaignCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignCriterionOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.CampaignCriterionOperation_Update{Update: cr},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetCampaignCriterion(ctx context.Context, resourceName string) (*CampaignCriterionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "campaignCriteria")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			campaign_criterion.resource_name,
			campaign_criterion.campaign,
			campaign_criterion.status,
			campaign_criterion.type,
			campaign_criterion.bid_modifier,
			campaign_criterion.negative,
			campaign_criterion.location.geo_target_constant,
			campaign_criterion.language.language_constant,
			campaign_criterion.device.type,
			campaign_criterion.ip_block.ip_address,
			campaign_criterion.keyword.text,
			campaign_criterion.keyword.match_type,
			campaign_criterion.ad_schedule.day_of_week,
			campaign_criterion.ad_schedule.start_hour,
			campaign_criterion.ad_schedule.end_hour,
			campaign_criterion.ad_schedule.start_minute,
			campaign_criterion.ad_schedule.end_minute,
			campaign_criterion.proximity.geo_point.latitude_in_micro_degrees,
			campaign_criterion.proximity.geo_point.longitude_in_micro_degrees,
			campaign_criterion.proximity.radius,
			campaign_criterion.proximity.radius_units,
			campaign_criterion.proximity.address.country_code,
			campaign_criterion.proximity.address.city_name
		FROM campaign_criterion
		WHERE campaign_criterion.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetCampaignCriterion()
	v := &CampaignCriterionView{
		ResourceName:          x.GetResourceName(),
		Campaign:              x.GetCampaign(),
		Status:                x.GetStatus().String(),
		Type:                  x.GetType().String(),
		BidModifier:           x.GetBidModifier(),
		Negative:              x.GetNegative(),
		LocationID:            x.GetLocation().GetGeoTargetConstant(),
		LanguageID:            x.GetLanguage().GetLanguageConstant(),
		DeviceType:            x.GetDevice().GetType().String(),
		IPAddress:             x.GetIpBlock().GetIpAddress(),
		KeywordText:           x.GetKeyword().GetText(),
		KeywordMatchType:      x.GetKeyword().GetMatchType().String(),
		AdScheduleDayOfWeek:   x.GetAdSchedule().GetDayOfWeek().String(),
		AdScheduleStartHour:   x.GetAdSchedule().GetStartHour(),
		AdScheduleEndHour:     x.GetAdSchedule().GetEndHour(),
		AdScheduleStartMinute: x.GetAdSchedule().GetStartMinute().String(),
		AdScheduleEndMinute:   x.GetAdSchedule().GetEndMinute().String(),
	}
	if p := x.GetProximity(); p != nil {
		v.HasProximity = true
		gp := p.GetGeoPoint()
		v.ProximityLatitude = float64(gp.GetLatitudeInMicroDegrees()) / 1e6
		v.ProximityLongitude = float64(gp.GetLongitudeInMicroDegrees()) / 1e6
		v.ProximityRadius = p.GetRadius()
		v.ProximityRadiusUnits = p.GetRadiusUnits().String()
		v.ProximityAddressCountry = p.GetAddress().GetCountryCode()
		v.ProximityAddressCityName = p.GetAddress().GetCityName()
	}
	return v, nil
}

func (c *Client) RemoveCampaignCriterion(ctx context.Context, resourceName string) error {
	cl, err := c.campaignCriterionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "campaignCriteria")
	if err != nil {
		return err
	}
	_, err = cl.MutateCampaignCriteria(ctx, &services.MutateCampaignCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.CampaignCriterionOperation{{
			Operation: &services.CampaignCriterionOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) campaignCriterionClient(ctx context.Context) (*pbclients.CampaignCriterionClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.campaignCriteria != nil {
		return c.campaignCriteria, nil
	}
	cl, err := pbclients.NewCampaignCriterionClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: campaign criterion client: %w", err)
	}
	c.campaignCriteria = cl
	return cl, nil
}

func applyCampaignCriterionVariant(cr *resources.CampaignCriterion, in CampaignCriterionInput) {
	switch {
	case in.LocationID != "":
		cr.Criterion = &resources.CampaignCriterion_Location{
			Location: &common.LocationInfo{GeoTargetConstant: StringPtr(in.LocationID)},
		}
	case in.LanguageID != "":
		cr.Criterion = &resources.CampaignCriterion_Language{
			Language: &common.LanguageInfo{LanguageConstant: StringPtr(in.LanguageID)},
		}
	case in.DeviceType != "":
		cr.Criterion = &resources.CampaignCriterion_Device{
			Device: &common.DeviceInfo{Type: enums.DeviceEnum_Device(enums.DeviceEnum_Device_value[in.DeviceType])},
		}
	case in.IPAddress != "":
		cr.Criterion = &resources.CampaignCriterion_IpBlock{
			IpBlock: &common.IpBlockInfo{IpAddress: StringPtr(in.IPAddress)},
		}
	case in.KeywordText != "":
		cr.Criterion = &resources.CampaignCriterion_Keyword{
			Keyword: &common.KeywordInfo{
				Text:      StringPtr(in.KeywordText),
				MatchType: enums.KeywordMatchTypeEnum_KeywordMatchType(enums.KeywordMatchTypeEnum_KeywordMatchType_value[in.KeywordMatchType]),
			},
		}
	case in.AdScheduleDayOfWeek != "":
		sched := &common.AdScheduleInfo{
			DayOfWeek:   enums.DayOfWeekEnum_DayOfWeek(enums.DayOfWeekEnum_DayOfWeek_value[in.AdScheduleDayOfWeek]),
			StartMinute: enums.MinuteOfHourEnum_MinuteOfHour(enums.MinuteOfHourEnum_MinuteOfHour_value[in.AdScheduleStartMinute]),
			EndMinute:   enums.MinuteOfHourEnum_MinuteOfHour(enums.MinuteOfHourEnum_MinuteOfHour_value[in.AdScheduleEndMinute]),
		}
		if in.AdScheduleStartHour != nil {
			sched.StartHour = in.AdScheduleStartHour
		}
		if in.AdScheduleEndHour != nil {
			sched.EndHour = in.AdScheduleEndHour
		}
		cr.Criterion = &resources.CampaignCriterion_AdSchedule{AdSchedule: sched}
	case in.ProximityLatitudeSet:
		lat := int32(math.Round(in.ProximityLatitude * 1e6))
		lng := int32(math.Round(in.ProximityLongitude * 1e6))
		radius := in.ProximityRadius
		info := &common.ProximityInfo{
			GeoPoint: &common.GeoPointInfo{
				LatitudeInMicroDegrees:  &lat,
				LongitudeInMicroDegrees: &lng,
			},
			Radius: &radius,
			RadiusUnits: enums.ProximityRadiusUnitsEnum_ProximityRadiusUnits(
				enums.ProximityRadiusUnitsEnum_ProximityRadiusUnits_value[in.ProximityRadiusUnits],
			),
		}
		if in.ProximityAddressCountry != "" || in.ProximityAddressCityName != "" {
			info.Address = &common.AddressInfo{}
			if in.ProximityAddressCountry != "" {
				info.Address.CountryCode = StringPtr(in.ProximityAddressCountry)
			}
			if in.ProximityAddressCityName != "" {
				info.Address.CityName = StringPtr(in.ProximityAddressCityName)
			}
		}
		cr.Criterion = &resources.CampaignCriterion_Proximity{Proximity: info}
	}
}
