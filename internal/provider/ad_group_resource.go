package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*adGroupResource)(nil)
	_ resource.ResourceWithConfigure      = (*adGroupResource)(nil)
	_ resource.ResourceWithImportState    = (*adGroupResource)(nil)
	_ resource.ResourceWithValidateConfig = (*adGroupResource)(nil)
)

func NewAdGroupResource() resource.Resource { return &adGroupResource{} }

type adGroupResource struct {
	client *googleads.Client
}

type adGroupModel struct {
	ID            types.String `tfsdk:"id"`
	CustomerID    types.String `tfsdk:"customer_id"`
	CampaignID    types.String `tfsdk:"campaign_id"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	Type          types.String `tfsdk:"type"`
	CpcBidMicros  types.Int64  `tfsdk:"cpc_bid_micros"`
	RemovalPolicy types.String `tfsdk:"removal_policy"`
}

func (r *adGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group"
}

func (r *adGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An ad group containing ads and keyword criteria. Campaign and type are immutable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "Ad group resource name.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"campaign_id": schema.StringAttribute{
				Required: true, Description: "Campaign resource name.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true, Description: "Ad group name.",
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true, Description: "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Ad group type, e.g. SEARCH_STANDARD (default). Immutable. Changing this destroys the ad group and loses serving history — blocked at plan time unless the provider's allow_destructive_replace is set.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), ProtectedReplace()},
			},
			"cpc_bid_micros": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "Max CPC bid in micros. Ignored under non-manual-CPC campaign bidding strategies.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"removal_policy": removalPolicyAttribute("ad group"),
		},
	}
}

func (r *adGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *adGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.AdGroupInput{
		CustomerID: plan.CustomerID.ValueString(),
		Campaign:   plan.CampaignID.ValueString(),
		Name:       plan.Name.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
		Type:       valueOrDefault(plan.Type, "SEARCH_STANDARD"),
	}
	if !plan.CpcBidMicros.IsNull() && !plan.CpcBidMicros.IsUnknown() {
		v := plan.CpcBidMicros.ValueInt64()
		in.CpcBidMicros = &v
	}
	rn, err := r.client.CreateAdGroup(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create ad group", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAdGroup(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group", err.Error())
		return
	}
	applyAdGroupView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAdGroup(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read ad group", err.Error())
		return
	}
	applyAdGroupView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	in := googleads.AdGroupInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		Status:     plan.Status.ValueString(),
	}
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.CpcBidMicros.Equal(state.CpcBidMicros) {
		paths = append(paths, "cpc_bid_micros")
		if !plan.CpcBidMicros.IsNull() && !plan.CpcBidMicros.IsUnknown() {
			v := plan.CpcBidMicros.ValueInt64()
			in.CpcBidMicros = &v
		}
	}
	if len(paths) > 0 {
		if err := r.client.UpdateAdGroup(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update ad group", nil, err)
			return
		}
	}
	view, err := r.client.GetAdGroup(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group", err.Error())
		return
	}
	plan.ID = state.ID
	applyAdGroupView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !enforceRemovalPolicy(&resp.Diagnostics, state.RemovalPolicy, "ad group") {
		return
	}
	if err := r.client.RemoveAdGroup(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove ad group", err.Error())
	}
}

func (r *adGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *adGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg adGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_id": cfg.CampaignID,
	})
}

func applyAdGroupView(m *adGroupModel, v *googleads.AdGroupView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "adGroups"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Status = types.StringValue(v.Status)
	m.Type = types.StringValue(v.Type)
	m.CampaignID = types.StringValue(v.Campaign)
	m.CpcBidMicros = types.Int64Value(v.CpcBidMicros)
}

func valueOrDefault(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return def
	}
	return v.ValueString()
}
