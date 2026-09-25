package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*campaignSharedSetResource)(nil)
	_ resource.ResourceWithConfigure      = (*campaignSharedSetResource)(nil)
	_ resource.ResourceWithImportState    = (*campaignSharedSetResource)(nil)
	_ resource.ResourceWithValidateConfig = (*campaignSharedSetResource)(nil)
)

func NewCampaignSharedSetResource() resource.Resource { return &campaignSharedSetResource{} }

type campaignSharedSetResource struct {
	client *googleads.Client
}

type campaignSharedSetModel struct {
	ID          types.String `tfsdk:"id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	CampaignID  types.String `tfsdk:"campaign_id"`
	SharedSetID types.String `tfsdk:"shared_set_id"`
	Status      types.String `tfsdk:"status"`
}

func (r *campaignSharedSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_shared_set"
}

func (r *campaignSharedSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a `googleads_shared_set` to a `googleads_campaign`. Both endpoints are immutable per the API; the link has no update operation, so any change forces resource replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/campaignSharedSets/{campaign_id}~{shared_set_id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"campaign_id": schema.StringAttribute{
				Required: true, Description: "Campaign resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"shared_set_id": schema.StringAttribute{
				Required: true, Description: "Shared set resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Computed:      true,
				Description:   "Output-only link status (ENABLED, REMOVED).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *campaignSharedSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *campaignSharedSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignSharedSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateCampaignSharedSet(ctx, googleads.CampaignSharedSetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Campaign:   plan.CampaignID.ValueString(),
		SharedSet:  plan.SharedSetID.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create campaign shared set", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCampaignSharedSet(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign shared set", err.Error())
		return
	}
	applyCampaignSharedSetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignSharedSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignSharedSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCampaignSharedSet(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read campaign shared set", err.Error())
		return
	}
	applyCampaignSharedSetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignSharedSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan campaignSharedSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignSharedSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignSharedSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCampaignSharedSet(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove campaign shared set", err.Error())
	}
}

func (r *campaignSharedSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *campaignSharedSetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg campaignSharedSetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_id":   cfg.CampaignID,
		"shared_set_id": cfg.SharedSetID,
	})
}

func applyCampaignSharedSetView(m *campaignSharedSetModel, v *googleads.CampaignSharedSetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "campaignSharedSets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.CampaignID = types.StringValue(v.Campaign)
	m.SharedSetID = types.StringValue(v.SharedSet)
	m.Status = types.StringValue(v.Status)
}
