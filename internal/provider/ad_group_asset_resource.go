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
	_ resource.Resource                = (*adGroupAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*adGroupAssetResource)(nil)
	_ resource.ResourceWithImportState = (*adGroupAssetResource)(nil)
)

func NewAdGroupAssetResource() resource.Resource { return &adGroupAssetResource{} }

type adGroupAssetResource struct {
	client *googleads.Client
}

type adGroupAssetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	AdGroupID  types.String `tfsdk:"ad_group_id"`
	AssetID    types.String `tfsdk:"asset_id"`
	FieldType  types.String `tfsdk:"field_type"`
	Status     types.String `tfsdk:"status"`
}

func (r *adGroupAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_asset"
}

func (r *adGroupAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Links an asset (sitelink, callout, structured snippet, etc.) to an ad group with a specific `field_type`. The triple (ad_group, asset, field_type) is immutable; only `status` mutates in place. Scoped narrower than `googleads_campaign_asset` — use this when an extension should only show for one ad group rather than the whole campaign.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/adGroupAssets/{ad_group_id}~{asset_id}~{field_type}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"ad_group_id": schema.StringAttribute{
				Required: true, Description: "Parent ad group resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"asset_id": schema.StringAttribute{
				Required: true, Description: "Asset resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"field_type": schema.StringAttribute{
				Required:    true,
				Description: "How the asset is used in the ad group. Common: SITELINK, CALLOUT, STRUCTURED_SNIPPET, MOBILE_APP, CALL, PROMOTION, PRICE. Immutable.",
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

func (r *adGroupAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *adGroupAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateAdGroupAssetLink(ctx, googleads.AdGroupAssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		AdGroup:    plan.AdGroupID.ValueString(),
		Asset:      plan.AssetID.ValueString(),
		FieldType:  plan.FieldType.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ad group asset", err.Error())
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAdGroupAssetLink(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group asset", err.Error())
		return
	}
	applyAdGroupAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAdGroupAssetLink(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read ad group asset", err.Error())
		return
	}
	applyAdGroupAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adGroupAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Status.Equal(state.Status) {
		if err := r.client.UpdateAdGroupAssetLinkStatus(ctx, state.ID.ValueString(), plan.Status.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update ad group asset status", err.Error())
			return
		}
	}
	view, err := r.client.GetAdGroupAssetLink(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyAdGroupAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveAdGroupAssetLink(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove ad group asset", err.Error())
	}
}

func (r *adGroupAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyAdGroupAssetView(m *adGroupAssetModel, v *googleads.AdGroupAssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "adGroupAssets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AdGroupID = types.StringValue(v.AdGroup)
	m.AssetID = types.StringValue(v.Asset)
	m.FieldType = types.StringValue(v.FieldType)
	m.Status = types.StringValue(v.Status)
}
