package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*campaignAssetResource)(nil)
	_ resource.ResourceWithConfigure      = (*campaignAssetResource)(nil)
	_ resource.ResourceWithImportState    = (*campaignAssetResource)(nil)
	_ resource.ResourceWithValidateConfig = (*campaignAssetResource)(nil)
)

func NewCampaignAssetResource() resource.Resource { return &campaignAssetResource{} }

type campaignAssetResource struct {
	client *googleads.Client
}

type campaignAssetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	CampaignID types.String `tfsdk:"campaign_id"`
	AssetID    types.String `tfsdk:"asset_id"`
	FieldType  types.String `tfsdk:"field_type"`
	Status     types.String `tfsdk:"status"`
}

func (r *campaignAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_asset"
}

func (r *campaignAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Links an asset (sitelink, callout, structured snippet, or anything else from `googleads_*_asset`) to a campaign with a specific `field_type` that tells the API how the asset is used. The triple (campaign, asset, field_type) is immutable; only `status` mutates in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/campaignAssets/{campaign_id}~{asset_id}~{field_type}).",
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
			"asset_id": schema.StringAttribute{
				Required: true, Description: "Asset resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"field_type": schema.StringAttribute{
				Required: true,
				Description: "How the asset is used in the campaign. Common: SITELINK, CALLOUT, STRUCTURED_SNIPPET, MOBILE_APP, CALL, PROMOTION, PRICE, LEAD_FORM. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"SITELINK", "CALLOUT", "STRUCTURED_SNIPPET",
						"MOBILE_APP", "CALL", "PROMOTION", "PRICE", "LEAD_FORM",
						"HOTEL_CALLOUT", "HOTEL_PROPERTY", "BUSINESS_NAME",
					),
				},
				PlanModifiers: immutable,
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *campaignAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *campaignAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateCampaignAsset(ctx, googleads.CampaignAssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Campaign:   plan.CampaignID.ValueString(),
		Asset:      plan.AssetID.ValueString(),
		FieldType:  plan.FieldType.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create campaign asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCampaignAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign asset", err.Error())
		return
	}
	applyCampaignAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCampaignAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read campaign asset", err.Error())
		return
	}
	applyCampaignAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state campaignAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Status.Equal(state.Status) {
		if err := r.client.UpdateCampaignAssetStatus(ctx, state.ID.ValueString(), plan.Status.ValueString()); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update campaign asset status", nil, err)
			return
		}
	}
	view, err := r.client.GetCampaignAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyCampaignAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignAssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCampaignAsset(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove campaign asset", err.Error())
	}
}

func (r *campaignAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *campaignAssetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg campaignAssetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_id": cfg.CampaignID,
		"asset_id":    cfg.AssetID,
	})
}

func applyCampaignAssetView(m *campaignAssetModel, v *googleads.CampaignAssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "campaignAssets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.CampaignID = types.StringValue(v.Campaign)
	m.AssetID = types.StringValue(v.Asset)
	m.FieldType = types.StringValue(v.FieldType)
	m.Status = types.StringValue(v.Status)
}
