package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                     = (*campaignCriterionResource)(nil)
	_ resource.ResourceWithConfigure        = (*campaignCriterionResource)(nil)
	_ resource.ResourceWithImportState      = (*campaignCriterionResource)(nil)
	_ resource.ResourceWithConfigValidators = (*campaignCriterionResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*campaignCriterionResource)(nil)
)

func NewCampaignCriterionResource() resource.Resource { return &campaignCriterionResource{} }

type campaignCriterionResource struct {
	client *googleads.Client
}

type campaignCriterionModel struct {
	ID          types.String  `tfsdk:"id"`
	CustomerID  types.String  `tfsdk:"customer_id"`
	CampaignID  types.String  `tfsdk:"campaign_id"`
	Status      types.String  `tfsdk:"status"`
	BidModifier types.Float64 `tfsdk:"bid_modifier"`
	Negative    types.Bool    `tfsdk:"negative"`

	LocationID types.String `tfsdk:"location_id"`
	LanguageID types.String `tfsdk:"language_id"`
	DeviceType types.String `tfsdk:"device_type"`
	IPAddress  types.String `tfsdk:"ip_address"`

	KeywordText      types.String `tfsdk:"keyword_text"`
	KeywordMatchType types.String `tfsdk:"keyword_match_type"`

	AdScheduleDayOfWeek   types.String `tfsdk:"ad_schedule_day_of_week"`
	AdScheduleStartHour   types.Int64  `tfsdk:"ad_schedule_start_hour"`
	AdScheduleEndHour     types.Int64  `tfsdk:"ad_schedule_end_hour"`
	AdScheduleStartMinute types.String `tfsdk:"ad_schedule_start_minute"`
	AdScheduleEndMinute   types.String `tfsdk:"ad_schedule_end_minute"`

	Latitude           types.Float64 `tfsdk:"latitude"`
	Longitude          types.Float64 `tfsdk:"longitude"`
	Radius             types.Float64 `tfsdk:"radius"`
	RadiusUnits        types.String  `tfsdk:"radius_units"`
	AddressCountryCode types.String  `tfsdk:"address_country_code"`
	AddressCityName    types.String  `tfsdk:"address_city_name"`
}

func (r *campaignCriterionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_criterion"
}

func (r *campaignCriterionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	immutableInt := []planmodifier.Int64{}
	immutableFloat := []planmodifier.Float64{float64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Campaign-scoped targeting or exclusion: location, language, device bid modifier, ad schedule, or IP block. Exactly one variant must be set. Status and bid_modifier are mutable; everything else is immutable per the API. v1 covers the five most-used variants — keyword / placement / audience / topic etc. land later.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/campaignCriteria/{campaign_id}~{criterion_id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"campaign_id": schema.StringAttribute{
				Required: true, Description: "Parent campaign resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED, PAUSED, or REMOVED. Mutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bid_modifier": schema.Float64Attribute{
				Optional: true, Computed: true,
				Description:   "Bid multiplier in range 0.1–10.0. Use 0 to opt out (Device only). Mutable.",
				PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},
			"negative": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "If true, exclude the criterion rather than target it. Immutable.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"location_id": schema.StringAttribute{
				Optional: true, Description: "Target a geo location. Value is a geoTargetConstants/{id} resource name, e.g. \"geoTargetConstants/2840\" (United States).",
				PlanModifiers: immutable,
			},
			"language_id": schema.StringAttribute{
				Optional: true, Description: "Target a language. Value is a languageConstants/{id} resource name, e.g. \"languageConstants/1014\" (Portuguese).",
				PlanModifiers: immutable,
			},
			"device_type": schema.StringAttribute{
				Optional: true, Description: "Apply a device-level bid modifier. One of MOBILE, TABLET, DESKTOP, CONNECTED_TV, OTHER.",
				Validators: []validator.String{
					stringvalidator.OneOf("MOBILE", "TABLET", "DESKTOP", "CONNECTED_TV", "OTHER"),
				},
				PlanModifiers: immutable,
			},
			"ip_address": schema.StringAttribute{
				Optional: true, Description: "Exclude an IP / CIDR.",
				PlanModifiers: immutable,
			},
			"keyword_text": schema.StringAttribute{
				Optional: true,
				Description: "Keyword text. Typically used with `negative = true` to add a campaign-level negative keyword that applies to every ad group. Pair with `keyword_match_type`. Immutable.",
				PlanModifiers: immutable,
			},
			"keyword_match_type": schema.StringAttribute{
				Optional: true,
				Description: "EXACT, PHRASE, or BROAD. Required when keyword_text is set.",
				Validators: []validator.String{
					stringvalidator.OneOf("EXACT", "PHRASE", "BROAD"),
				},
				PlanModifiers: immutable,
			},
			"ad_schedule_day_of_week": schema.StringAttribute{
				Optional: true, Description: "MONDAY..SUNDAY. Required when configuring an ad schedule.",
				Validators: []validator.String{
					stringvalidator.OneOf("MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"),
				},
				PlanModifiers: immutable,
			},
			"ad_schedule_start_hour": schema.Int64Attribute{
				Optional: true, Description: "Schedule start hour (0–23). Required when configuring an ad schedule.",
				PlanModifiers: immutableInt,
			},
			"ad_schedule_end_hour": schema.Int64Attribute{
				Optional: true, Description: "Schedule end hour (0–24, 24 = end of day). Required when configuring an ad schedule.",
				PlanModifiers: immutableInt,
			},
			"ad_schedule_start_minute": schema.StringAttribute{
				Optional: true, Description: "ZERO, FIFTEEN, THIRTY, or FORTY_FIVE. Required when configuring an ad schedule.",
				Validators: []validator.String{
					stringvalidator.OneOf("ZERO", "FIFTEEN", "THIRTY", "FORTY_FIVE"),
				},
				PlanModifiers: immutable,
			},
			"ad_schedule_end_minute": schema.StringAttribute{
				Optional: true, Description: "ZERO, FIFTEEN, THIRTY, or FORTY_FIVE. Required when configuring an ad schedule.",
				Validators: []validator.String{
					stringvalidator.OneOf("ZERO", "FIFTEEN", "THIRTY", "FORTY_FIVE"),
				},
				PlanModifiers: immutable,
			},
			"latitude": schema.Float64Attribute{
				Optional: true,
				Description: "Latitude (degrees, WGS84) of the centre of a radius-targeting proximity. Required together with longitude, radius, and radius_units. Stored at micro-degree precision (~0.11 m).",
				PlanModifiers: immutableFloat,
			},
			"longitude": schema.Float64Attribute{
				Optional: true,
				Description: "Longitude (degrees, WGS84). Required together with latitude.",
				PlanModifiers: immutableFloat,
			},
			"radius": schema.Float64Attribute{
				Optional: true,
				Description: "Proximity radius. Units determined by radius_units (1–800 KM or 1–500 MI).",
				PlanModifiers: immutableFloat,
			},
			"radius_units": schema.StringAttribute{
				Optional: true,
				Description: "KILOMETERS or MILES.",
				Validators: []validator.String{
					stringvalidator.OneOf("KILOMETERS", "MILES"),
				},
				PlanModifiers: immutable,
			},
			"address_country_code": schema.StringAttribute{
				Optional: true,
				Description: "Optional ISO 3166-1 alpha-2 country code attached as a label on the proximity (e.g. \"BR\"). Does not affect serving.",
				PlanModifiers: immutable,
			},
			"address_city_name": schema.StringAttribute{
				Optional: true,
				Description: "Optional city-name label attached to the proximity. Does not affect serving.",
				PlanModifiers: immutable,
			},
		},
	}
}

func (r *campaignCriterionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("location_id"),
			path.MatchRoot("language_id"),
			path.MatchRoot("device_type"),
			path.MatchRoot("ip_address"),
			path.MatchRoot("keyword_text"),
			path.MatchRoot("ad_schedule_day_of_week"),
			path.MatchRoot("latitude"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("keyword_text"),
			path.MatchRoot("keyword_match_type"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("latitude"),
			path.MatchRoot("longitude"),
			path.MatchRoot("radius"),
			path.MatchRoot("radius_units"),
		),
	}
}

func (r *campaignCriterionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *campaignCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := buildCampaignCriterionInput(&plan)
	rn, err := r.client.CreateCampaignCriterion(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create campaign criterion", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCampaignCriterion(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign criterion", err.Error())
		return
	}
	applyCampaignCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCampaignCriterion(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read campaign criterion", err.Error())
		return
	}
	applyCampaignCriterionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state campaignCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.BidModifier.Equal(state.BidModifier) {
		paths = append(paths, "bid_modifier")
	}
	if len(paths) > 0 {
		in := buildCampaignCriterionInput(&plan)
		if err := r.client.UpdateCampaignCriterion(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update campaign criterion", nil, err)
			return
		}
	}
	view, err := r.client.GetCampaignCriterion(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign criterion", err.Error())
		return
	}
	plan.ID = state.ID
	applyCampaignCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCampaignCriterion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove campaign criterion", err.Error())
	}
}

func (r *campaignCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *campaignCriterionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg campaignCriterionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_id": cfg.CampaignID,
	})
}

func buildCampaignCriterionInput(m *campaignCriterionModel) googleads.CampaignCriterionInput {
	in := googleads.CampaignCriterionInput{
		CustomerID:               m.CustomerID.ValueString(),
		Campaign:                 m.CampaignID.ValueString(),
		Status:                   m.Status.ValueString(),
		LocationID:               m.LocationID.ValueString(),
		LanguageID:               m.LanguageID.ValueString(),
		DeviceType:               m.DeviceType.ValueString(),
		IPAddress:                m.IPAddress.ValueString(),
		KeywordText:              m.KeywordText.ValueString(),
		KeywordMatchType:         m.KeywordMatchType.ValueString(),
		AdScheduleDayOfWeek:      m.AdScheduleDayOfWeek.ValueString(),
		AdScheduleStartMinute:    m.AdScheduleStartMinute.ValueString(),
		AdScheduleEndMinute:      m.AdScheduleEndMinute.ValueString(),
		ProximityRadiusUnits:     m.RadiusUnits.ValueString(),
		ProximityAddressCountry:  m.AddressCountryCode.ValueString(),
		ProximityAddressCityName: m.AddressCityName.ValueString(),
	}
	if !m.BidModifier.IsNull() && !m.BidModifier.IsUnknown() {
		in.BidModifier = float32(m.BidModifier.ValueFloat64())
	}
	if !m.Negative.IsNull() && !m.Negative.IsUnknown() {
		v := m.Negative.ValueBool()
		in.Negative = &v
	}
	if !m.AdScheduleStartHour.IsNull() && !m.AdScheduleStartHour.IsUnknown() {
		v := int32(m.AdScheduleStartHour.ValueInt64())
		in.AdScheduleStartHour = &v
	}
	if !m.AdScheduleEndHour.IsNull() && !m.AdScheduleEndHour.IsUnknown() {
		v := int32(m.AdScheduleEndHour.ValueInt64())
		in.AdScheduleEndHour = &v
	}
	if !m.Latitude.IsNull() && !m.Latitude.IsUnknown() {
		in.ProximityLatitudeSet = true
		in.ProximityLatitude = m.Latitude.ValueFloat64()
		in.ProximityLongitude = m.Longitude.ValueFloat64()
		in.ProximityRadius = m.Radius.ValueFloat64()
	}
	return in
}

func applyCampaignCriterionView(m *campaignCriterionModel, v *googleads.CampaignCriterionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "campaignCriteria"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.CampaignID = types.StringValue(v.Campaign)
	m.Status = types.StringValue(v.Status)
	m.BidModifier = types.Float64Value(float64(v.BidModifier))
	m.Negative = types.BoolValue(v.Negative)
	m.LocationID = nullableString(v.LocationID)
	m.LanguageID = nullableString(v.LanguageID)
	m.DeviceType = nullableEnum(v.DeviceType)
	m.IPAddress = nullableString(v.IPAddress)
	m.KeywordText = nullableString(v.KeywordText)
	m.KeywordMatchType = nullableEnum(v.KeywordMatchType)
	m.AdScheduleDayOfWeek = nullableEnum(v.AdScheduleDayOfWeek)
	m.AdScheduleStartHour = nullableInt(v.AdScheduleStartHour)
	m.AdScheduleEndHour = nullableInt(v.AdScheduleEndHour)
	m.AdScheduleStartMinute = nullableEnum(v.AdScheduleStartMinute)
	m.AdScheduleEndMinute = nullableEnum(v.AdScheduleEndMinute)
	if v.HasProximity {
		m.Latitude = types.Float64Value(v.ProximityLatitude)
		m.Longitude = types.Float64Value(v.ProximityLongitude)
		m.Radius = types.Float64Value(v.ProximityRadius)
		m.RadiusUnits = nullableEnum(v.ProximityRadiusUnits)
		m.AddressCountryCode = nullableString(v.ProximityAddressCountry)
		m.AddressCityName = nullableString(v.ProximityAddressCityName)
	} else {
		m.Latitude = types.Float64Null()
		m.Longitude = types.Float64Null()
		m.Radius = types.Float64Null()
		m.RadiusUnits = types.StringNull()
		m.AddressCountryCode = types.StringNull()
		m.AddressCityName = types.StringNull()
	}
}

// nullableEnum is like nullableString but also normalises the API's
// UNSPECIFIED / UNKNOWN sentinel strings back to null so an unset device
// type (for example) doesn't surface as the string "UNSPECIFIED".
func nullableEnum(s string) types.String {
	if s == "" || s == "UNSPECIFIED" || s == "UNKNOWN" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func nullableInt(v int32) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(v))
}
