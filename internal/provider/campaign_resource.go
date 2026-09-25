package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*campaignResource)(nil)
	_ resource.ResourceWithConfigure      = (*campaignResource)(nil)
	_ resource.ResourceWithImportState    = (*campaignResource)(nil)
	_ resource.ResourceWithValidateConfig = (*campaignResource)(nil)
)

func NewCampaignResource() resource.Resource { return &campaignResource{} }

type campaignResource struct {
	client *googleads.Client
}

type campaignModel struct {
	ID                             types.String  `tfsdk:"id"`
	CustomerID                     types.String  `tfsdk:"customer_id"`
	Name                           types.String  `tfsdk:"name"`
	Status                         types.String  `tfsdk:"status"`
	AdvertisingChannelType         types.String  `tfsdk:"advertising_channel_type"`
	CampaignBudgetID               types.String  `tfsdk:"campaign_budget_id"`
	BiddingStrategyType            types.String  `tfsdk:"bidding_strategy_type"`
	TargetCpaMicros                types.Int64   `tfsdk:"target_cpa_micros"`
	TargetRoas                     types.Float64 `tfsdk:"target_roas"`
	CpcBidCeilingMicros            types.Int64   `tfsdk:"cpc_bid_ceiling_micros"`
	CpcBidFloorMicros              types.Int64   `tfsdk:"cpc_bid_floor_micros"`
	NetworkSettings                types.Object  `tfsdk:"network_settings"`
	GeoTargetTypeSetting           types.Object  `tfsdk:"geo_target_type_setting"`
	ContainsEuPoliticalAdvertising types.String  `tfsdk:"contains_eu_political_advertising"`
	RemovalPolicy                  types.String  `tfsdk:"removal_policy"`
}

var networkSettingsAttrTypes = map[string]attr.Type{
	"target_google_search":          types.BoolType,
	"target_search_network":         types.BoolType,
	"target_content_network":        types.BoolType,
	"target_partner_search_network": types.BoolType,
}

var geoTargetTypeSettingAttrTypes = map[string]attr.Type{
	"positive_geo_target_type": types.StringType,
	"negative_geo_target_type": types.StringType,
}

func (r *campaignResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign"
}

func (r *campaignResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Google Ads campaign. Channel type is immutable. Bidding strategy is immutable in this provider (the API allows switching it but mapping that cleanly to Terraform is left for later) — Terraform replaces the campaign on strategy changes. The strategy's parameter fields (target_cpa_micros, target_roas, cpc_bid_*) are mutable in place when set under a strategy that supports them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The campaign resource name (customers/{cid}/campaigns/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required:    true,
				Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Campaign name.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ENABLED, PAUSED, or REMOVED. Defaults to PAUSED on create to avoid accidental spend.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"advertising_channel_type": schema.StringAttribute{
				Required:    true,
				Description: "SEARCH, DISPLAY, SHOPPING, VIDEO, PERFORMANCE_MAX, etc. Immutable. Changing this field requires destroying the campaign — blocked at plan time unless the provider's allow_destructive_replace is set.",
				PlanModifiers: []planmodifier.String{
					ProtectedReplace(),
				},
			},
			"campaign_budget_id": schema.StringAttribute{
				Required:    true,
				Description: "Resource name of the campaign budget backing this campaign.",
			},
			"bidding_strategy_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Bidding strategy. Supported: MANUAL_CPC (default), MANUAL_CPM, MANUAL_CPV, TARGET_SPEND, TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS, MAXIMIZE_CONVERSION_VALUE. Smart strategies use the *_micros / target_roas parameter fields below. Switching strategies on an existing campaign loses Smart Bidding learning data — blocked at plan time unless the provider's allow_destructive_replace is set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					ProtectedReplace(),
				},
			},
			"target_cpa_micros": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Target cost-per-acquisition in micros. Applies under TARGET_CPA (required) and MAXIMIZE_CONVERSIONS (optional). Ignored otherwise.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"target_roas": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Target return-on-ad-spend (e.g. 3.5 = aim for $3.50 revenue per $1 spend). Applies under TARGET_ROAS (required) and MAXIMIZE_CONVERSION_VALUE (optional). Ignored otherwise.",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"cpc_bid_ceiling_micros": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Upper CPC bound for portfolio bidding strategies (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS, MAXIMIZE_CONVERSION_VALUE).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"cpc_bid_floor_micros": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Lower CPC bound for portfolio bidding strategies. Currently only TARGET_CPA and TARGET_ROAS accept this field.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"network_settings": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Where this campaign's ads are eligible to serve. Omit the block to accept whichever defaults the API picks for the channel type; set any subset of fields to pin them in Terraform. For brand-safety setups, the common pattern on SEARCH campaigns is to force target_search_network = false and target_content_network = false so the campaign is locked to Google Search proper.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"target_google_search": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Serve on Google Search results.",
					},
					"target_search_network": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Serve on Google's Search Partners network.",
					},
					"target_content_network": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Serve on the Google Display Network.",
					},
					"target_partner_search_network": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Serve on Google's partner search network. Available only to a small set of partners; ignore unless you've been told you have access.",
					},
				},
			},
			"geo_target_type_setting": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "How the campaign matches users to its targeted geographies. Omit to accept API defaults; set explicitly to pin the choice in Terraform.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"positive_geo_target_type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "PRESENCE_OR_INTEREST (default — users in or showing interest in the target), PRESENCE (only users in the target), or SEARCH_INTEREST (search-only campaigns: users searching for the target).",
					},
					"negative_geo_target_type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "PRESENCE (exclude users in the negative location) or SEARCH_INTEREST.",
					},
				},
			},
			"contains_eu_political_advertising": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the campaign contains EU political advertising. CONTAINS_EU_POLITICAL_ADVERTISING or DOES_NOT_CONTAIN_EU_POLITICAL_ADVERTISING. The Google Ads API requires this declaration to create campaigns in affected accounts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"removal_policy": removalPolicyAttribute("campaign"),
		},
	}
}

func (r *campaignResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *campaignResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status := plan.Status.ValueString()
	if plan.Status.IsNull() || plan.Status.IsUnknown() {
		status = "PAUSED"
	}
	in := googleads.CampaignInput{
		CustomerID:                     plan.CustomerID.ValueString(),
		Name:                           plan.Name.ValueString(),
		Status:                         status,
		AdvertisingChannelType:         plan.AdvertisingChannelType.ValueString(),
		CampaignBudget:                 plan.CampaignBudgetID.ValueString(),
		BiddingStrategyType:            plan.BiddingStrategyType.ValueString(),
		ContainsEuPoliticalAdvertising: plan.ContainsEuPoliticalAdvertising.ValueString(),
	}
	applyCampaignBiddingParams(&plan, &in)
	in.NetworkSettings = networkSettingsFromPlan(plan.NetworkSettings)
	in.GeoTargetTypeSetting = geoTargetTypeSettingFromPlan(plan.GeoTargetTypeSetting)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceName, err := r.client.CreateCampaign(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create campaign", nil, err)
		return
	}
	plan.ID = types.StringValue(resourceName)
	view, err := r.client.GetCampaign(ctx, resourceName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign", err.Error())
		return
	}
	applyCampaignView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCampaign(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read campaign", err.Error())
		return
	}
	applyCampaignView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state campaignModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.CampaignBudgetID.Equal(state.CampaignBudgetID) {
		paths = append(paths, "campaign_budget")
	}
	// Bidding parameter changes map to the API's nested field-mask paths;
	// each strategy variant has its own canonical path the API recognises.
	strategy := plan.BiddingStrategyType.ValueString()
	if !plan.TargetCpaMicros.Equal(state.TargetCpaMicros) {
		switch strategy {
		case "TARGET_CPA":
			paths = append(paths, "target_cpa.target_cpa_micros")
		case "MAXIMIZE_CONVERSIONS":
			paths = append(paths, "maximize_conversions.target_cpa_micros")
		}
	}
	if !plan.TargetRoas.Equal(state.TargetRoas) {
		switch strategy {
		case "TARGET_ROAS":
			paths = append(paths, "target_roas.target_roas")
		case "MAXIMIZE_CONVERSION_VALUE":
			paths = append(paths, "maximize_conversion_value.target_roas")
		}
	}
	if !plan.CpcBidCeilingMicros.Equal(state.CpcBidCeilingMicros) {
		switch strategy {
		case "TARGET_CPA":
			paths = append(paths, "target_cpa.cpc_bid_ceiling_micros")
		case "TARGET_ROAS":
			paths = append(paths, "target_roas.cpc_bid_ceiling_micros")
		case "MAXIMIZE_CONVERSIONS":
			paths = append(paths, "maximize_conversions.cpc_bid_ceiling_micros")
		case "MAXIMIZE_CONVERSION_VALUE":
			paths = append(paths, "maximize_conversion_value.cpc_bid_ceiling_micros")
		case "TARGET_SPEND":
			paths = append(paths, "target_spend.cpc_bid_ceiling_micros")
		}
	}
	if !plan.CpcBidFloorMicros.Equal(state.CpcBidFloorMicros) {
		switch strategy {
		case "TARGET_CPA":
			paths = append(paths, "target_cpa.cpc_bid_floor_micros")
		case "TARGET_ROAS":
			paths = append(paths, "target_roas.cpc_bid_floor_micros")
		}
	}
	if !plan.ContainsEuPoliticalAdvertising.Equal(state.ContainsEuPoliticalAdvertising) {
		paths = append(paths, "contains_eu_political_advertising")
	}
	paths = append(paths, diffNetworkSettingsPaths(plan.NetworkSettings, state.NetworkSettings)...)
	paths = append(paths, diffGeoTargetTypeSettingPaths(plan.GeoTargetTypeSetting, state.GeoTargetTypeSetting)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := googleads.CampaignInput{
		CustomerID:                     plan.CustomerID.ValueString(),
		Name:                           plan.Name.ValueString(),
		Status:                         plan.Status.ValueString(),
		CampaignBudget:                 plan.CampaignBudgetID.ValueString(),
		BiddingStrategyType:            strategy,
		ContainsEuPoliticalAdvertising: plan.ContainsEuPoliticalAdvertising.ValueString(),
	}
	applyCampaignBiddingParams(&plan, &in)
	in.NetworkSettings = networkSettingsFromPlan(plan.NetworkSettings)
	in.GeoTargetTypeSetting = geoTargetTypeSettingFromPlan(plan.GeoTargetTypeSetting)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(paths) > 0 {
		if err := r.client.UpdateCampaign(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update campaign", nil, err)
			return
		}
	}
	view, err := r.client.GetCampaign(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign", err.Error())
		return
	}
	plan.ID = state.ID
	applyCampaignView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !enforceRemovalPolicy(&resp.Diagnostics, state.RemovalPolicy, "campaign") {
		return
	}
	if err := r.client.RemoveCampaign(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove campaign", err.Error())
	}
}

func (r *campaignResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *campaignResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg campaignModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_budget_id": cfg.CampaignBudgetID,
	})
}

func applyCampaignView(m *campaignModel, v *googleads.CampaignView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "campaigns"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Status = types.StringValue(v.Status)
	m.AdvertisingChannelType = types.StringValue(v.AdvertisingChannelType)
	m.CampaignBudgetID = types.StringValue(v.CampaignBudget)
	m.BiddingStrategyType = types.StringValue(v.BiddingStrategyType)
	m.ContainsEuPoliticalAdvertising = types.StringValue(v.ContainsEuPoliticalAdvertising)
	m.TargetCpaMicros = types.Int64Value(v.TargetCpaMicros)
	m.TargetRoas = types.Float64Value(v.TargetRoas)
	m.CpcBidCeilingMicros = types.Int64Value(v.CpcBidCeilingMicros)
	m.CpcBidFloorMicros = types.Int64Value(v.CpcBidFloorMicros)
	m.NetworkSettings = types.ObjectValueMust(networkSettingsAttrTypes, map[string]attr.Value{
		"target_google_search":          types.BoolValue(v.NetworkSettings.TargetGoogleSearch),
		"target_search_network":         types.BoolValue(v.NetworkSettings.TargetSearchNetwork),
		"target_content_network":        types.BoolValue(v.NetworkSettings.TargetContentNetwork),
		"target_partner_search_network": types.BoolValue(v.NetworkSettings.TargetPartnerSearchNetwork),
	})
	m.GeoTargetTypeSetting = types.ObjectValueMust(geoTargetTypeSettingAttrTypes, map[string]attr.Value{
		"positive_geo_target_type": types.StringValue(v.GeoTargetTypeSetting.PositiveGeoTargetType),
		"negative_geo_target_type": types.StringValue(v.GeoTargetTypeSetting.NegativeGeoTargetType),
	})
}

// networkSettingsFromPlan extracts user-set fields from the plan's
// network_settings block. Unknown/null fields stay nil so the client knows
// not to write them; that matters on create (API picks channel defaults)
// and on update (we only patch what changed).
func networkSettingsFromPlan(o types.Object) *googleads.NetworkSettings {
	if o.IsNull() || o.IsUnknown() {
		return nil
	}
	attrs := o.Attributes()
	out := &googleads.NetworkSettings{}
	out.TargetGoogleSearch = boolFromAttr(attrs["target_google_search"])
	out.TargetSearchNetwork = boolFromAttr(attrs["target_search_network"])
	out.TargetContentNetwork = boolFromAttr(attrs["target_content_network"])
	out.TargetPartnerSearchNetwork = boolFromAttr(attrs["target_partner_search_network"])
	if out.TargetGoogleSearch == nil && out.TargetSearchNetwork == nil &&
		out.TargetContentNetwork == nil && out.TargetPartnerSearchNetwork == nil {
		return nil
	}
	return out
}

func geoTargetTypeSettingFromPlan(o types.Object) *googleads.GeoTargetTypeSetting {
	if o.IsNull() || o.IsUnknown() {
		return nil
	}
	attrs := o.Attributes()
	out := &googleads.GeoTargetTypeSetting{
		PositiveGeoTargetType: stringFromAttr(attrs["positive_geo_target_type"]),
		NegativeGeoTargetType: stringFromAttr(attrs["negative_geo_target_type"]),
	}
	if out.PositiveGeoTargetType == "" && out.NegativeGeoTargetType == "" {
		return nil
	}
	return out
}

// diffNetworkSettingsPaths returns the field-mask paths whose values
// differ between plan and state. A nested attribute showing up as Unknown
// in the plan is treated as "no user intent expressed" — the API value
// stays put.
func diffNetworkSettingsPaths(plan, state types.Object) []string {
	if plan.IsNull() || plan.IsUnknown() {
		return nil
	}
	pAttrs := plan.Attributes()
	var sAttrs map[string]attr.Value
	if !state.IsNull() && !state.IsUnknown() {
		sAttrs = state.Attributes()
	}
	var paths []string
	for _, k := range []string{"target_google_search", "target_search_network", "target_content_network", "target_partner_search_network"} {
		pv := pAttrs[k]
		if pv == nil || pv.IsNull() || pv.IsUnknown() {
			continue
		}
		if sAttrs == nil || sAttrs[k] == nil || !pv.Equal(sAttrs[k]) {
			paths = append(paths, "network_settings."+k)
		}
	}
	return paths
}

func diffGeoTargetTypeSettingPaths(plan, state types.Object) []string {
	if plan.IsNull() || plan.IsUnknown() {
		return nil
	}
	pAttrs := plan.Attributes()
	var sAttrs map[string]attr.Value
	if !state.IsNull() && !state.IsUnknown() {
		sAttrs = state.Attributes()
	}
	var paths []string
	for _, k := range []string{"positive_geo_target_type", "negative_geo_target_type"} {
		pv := pAttrs[k]
		if pv == nil || pv.IsNull() || pv.IsUnknown() {
			continue
		}
		if sAttrs == nil || sAttrs[k] == nil || !pv.Equal(sAttrs[k]) {
			paths = append(paths, "geo_target_type_setting."+k)
		}
	}
	return paths
}

func boolFromAttr(v attr.Value) *bool {
	b, ok := v.(basetypes.BoolValue)
	if !ok || b.IsNull() || b.IsUnknown() {
		return nil
	}
	x := b.ValueBool()
	return &x
}

func stringFromAttr(v attr.Value) string {
	s, ok := v.(basetypes.StringValue)
	if !ok || s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

// applyCampaignBiddingParams copies optional numeric attrs from the plan
// into the client input as pointers. nil = "user didn't set it" so the
// client knows to skip the field; the API will leave the existing value
// (or its default) in place.
func applyCampaignBiddingParams(plan *campaignModel, in *googleads.CampaignInput) {
	if !plan.TargetCpaMicros.IsNull() && !plan.TargetCpaMicros.IsUnknown() {
		v := plan.TargetCpaMicros.ValueInt64()
		in.TargetCpaMicros = &v
	}
	if !plan.TargetRoas.IsNull() && !plan.TargetRoas.IsUnknown() {
		v := plan.TargetRoas.ValueFloat64()
		in.TargetRoas = &v
	}
	if !plan.CpcBidCeilingMicros.IsNull() && !plan.CpcBidCeilingMicros.IsUnknown() {
		v := plan.CpcBidCeilingMicros.ValueInt64()
		in.CpcBidCeilingMicros = &v
	}
	if !plan.CpcBidFloorMicros.IsNull() && !plan.CpcBidFloorMicros.IsUnknown() {
		v := plan.CpcBidFloorMicros.ValueInt64()
		in.CpcBidFloorMicros = &v
	}
}
