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
	_ resource.Resource                = (*customerAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*customerAssetResource)(nil)
	_ resource.ResourceWithImportState = (*customerAssetResource)(nil)
)

func NewCustomerAssetResource() resource.Resource { return &customerAssetResource{} }

type customerAssetResource struct {
	client *googleads.Client
}

type customerAssetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	AssetID    types.String `tfsdk:"asset_id"`
	FieldType  types.String `tfsdk:"field_type"`
	Status     types.String `tfsdk:"status"`
}

func (r *customerAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer_asset"
}

func (r *customerAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Links an asset (sitelink, callout, structured snippet, etc.) to the entire customer account, making it eligible across every campaign and ad group unless overridden. The pair (asset, field_type) is immutable; only `status` mutates in place. Broadest scope of the three asset-link resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/customerAssets/{asset_id}~{field_type}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID. Immutable.",
				PlanModifiers: immutable,
			},
			"asset_id": schema.StringAttribute{
				Required: true, Description: "Asset resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"field_type": schema.StringAttribute{
				Required:    true,
				Description: "How the asset is used at the account level. Common: SITELINK, CALLOUT, STRUCTURED_SNIPPET, MOBILE_APP, CALL, PROMOTION, PRICE. Immutable.",
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

func (r *customerAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *customerAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customerAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateCustomerAsset(ctx, googleads.CustomerAssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Asset:      plan.AssetID.ValueString(),
		FieldType:  plan.FieldType.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create customer asset", err.Error())
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCustomerAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back customer asset", err.Error())
		return
	}
	applyCustomerAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customerAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customerAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCustomerAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read customer asset", err.Error())
		return
	}
	applyCustomerAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customerAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customerAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Status.Equal(state.Status) {
		if err := r.client.UpdateCustomerAssetStatus(ctx, state.ID.ValueString(), plan.Status.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update customer asset status", err.Error())
			return
		}
	}
	view, err := r.client.GetCustomerAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back customer asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyCustomerAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customerAssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customerAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCustomerAsset(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove customer asset", err.Error())
	}
}

func (r *customerAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyCustomerAssetView(m *customerAssetModel, v *googleads.CustomerAssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "customerAssets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AssetID = types.StringValue(v.Asset)
	m.FieldType = types.StringValue(v.FieldType)
	m.Status = types.StringValue(v.Status)
}
