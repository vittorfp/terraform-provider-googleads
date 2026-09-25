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
	_ resource.Resource                   = (*assetGroupAssetResource)(nil)
	_ resource.ResourceWithConfigure      = (*assetGroupAssetResource)(nil)
	_ resource.ResourceWithImportState    = (*assetGroupAssetResource)(nil)
	_ resource.ResourceWithValidateConfig = (*assetGroupAssetResource)(nil)
)

func NewAssetGroupAssetResource() resource.Resource { return &assetGroupAssetResource{} }

type assetGroupAssetResource struct {
	client *googleads.Client
}

type assetGroupAssetModel struct {
	ID           types.String `tfsdk:"id"`
	CustomerID   types.String `tfsdk:"customer_id"`
	AssetGroupID types.String `tfsdk:"asset_group_id"`
	AssetID      types.String `tfsdk:"asset_id"`
	FieldType    types.String `tfsdk:"field_type"`
	Status       types.String `tfsdk:"status"`
}

func (r *assetGroupAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_asset_group_asset"
}

func (r *assetGroupAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links a `googleads_text_asset` or `googleads_image_asset` to a `googleads_asset_group` with a specific `field_type` describing how the asset is used (HEADLINE, DESCRIPTION, MARKETING_IMAGE, LOGO, etc.). The triple (asset_group, asset, field_type) is the resource's identity and is immutable; only `status` can be updated in place — anything else forces resource replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/assetGroupAssets/{ag_id}~{asset_id}~{field_type}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"asset_group_id": schema.StringAttribute{
				Required: true, Description: "Asset group resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"asset_id": schema.StringAttribute{
				Required: true, Description: "Asset resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"field_type": schema.StringAttribute{
				Required: true,
				Description: "How the asset is used in the group. Common values: HEADLINE, DESCRIPTION, LONG_HEADLINE, BUSINESS_NAME, MARKETING_IMAGE, SQUARE_MARKETING_IMAGE, PORTRAIT_MARKETING_IMAGE, LOGO, LANDSCAPE_LOGO, YOUTUBE_VIDEO, CALL_TO_ACTION_SELECTION.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"HEADLINE", "DESCRIPTION", "LONG_HEADLINE", "BUSINESS_NAME",
						"MARKETING_IMAGE", "SQUARE_MARKETING_IMAGE", "PORTRAIT_MARKETING_IMAGE",
						"LOGO", "LANDSCAPE_LOGO",
						"YOUTUBE_VIDEO", "CALL_TO_ACTION_SELECTION",
					),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *assetGroupAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *assetGroupAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assetGroupAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateAssetGroupAsset(ctx, googleads.AssetGroupAssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		AssetGroup: plan.AssetGroupID.ValueString(),
		Asset:      plan.AssetID.ValueString(),
		FieldType:  plan.FieldType.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create asset group asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAssetGroupAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back asset group asset", err.Error())
		return
	}
	applyAssetGroupAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assetGroupAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assetGroupAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAssetGroupAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read asset group asset", err.Error())
		return
	}
	applyAssetGroupAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *assetGroupAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state assetGroupAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Status.Equal(state.Status) {
		if err := r.client.UpdateAssetGroupAssetStatus(ctx, state.ID.ValueString(), plan.Status.ValueString()); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update asset group asset status", nil, err)
			return
		}
	}
	view, err := r.client.GetAssetGroupAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back asset group asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyAssetGroupAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assetGroupAssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assetGroupAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveAssetGroupAsset(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove asset group asset", err.Error())
	}
}

func (r *assetGroupAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *assetGroupAssetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg assetGroupAssetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"asset_group_id": cfg.AssetGroupID,
		"asset_id":       cfg.AssetID,
	})
}

func applyAssetGroupAssetView(m *assetGroupAssetModel, v *googleads.AssetGroupAssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assetGroupAssets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AssetGroupID = types.StringValue(v.AssetGroup)
	m.AssetID = types.StringValue(v.Asset)
	m.FieldType = types.StringValue(v.FieldType)
	m.Status = types.StringValue(v.Status)
}
