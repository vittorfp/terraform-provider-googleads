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
	_ resource.Resource                = (*calloutAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*calloutAssetResource)(nil)
	_ resource.ResourceWithImportState = (*calloutAssetResource)(nil)
)

func NewCalloutAssetResource() resource.Resource { return &calloutAssetResource{} }

type calloutAssetResource struct {
	client *googleads.Client
}

type calloutAssetModel struct {
	ID          types.String `tfsdk:"id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	Name        types.String `tfsdk:"name"`
	CalloutText types.String `tfsdk:"callout_text"`
}

func (r *calloutAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_callout_asset"
}

func (r *calloutAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A callout asset — a short non-clickable phrase shown under a search ad (e.g. \"Free shipping\", \"24/7 support\"). Asset content is immutable; only the wrapper's `name` can be renamed. **The API has no remove operation** — to stop managing, unlink, remove from HCL, then `terraform state rm`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Asset resource name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Optional display name.",
			},
			"callout_text": schema.StringAttribute{
				Required:    true,
				Description: "The callout text (1–25 characters). Immutable.",
				PlanModifiers: immutable,
			},
		},
	}
}

func (r *calloutAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *calloutAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan calloutAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateCalloutAsset(ctx, googleads.CalloutAssetInput{
		CustomerID:  plan.CustomerID.ValueString(),
		Name:        plan.Name.ValueString(),
		CalloutText: plan.CalloutText.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create callout asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCalloutAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back callout asset", err.Error())
		return
	}
	applyCalloutAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *calloutAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state calloutAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCalloutAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read callout asset", err.Error())
		return
	}
	applyCalloutAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *calloutAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state calloutAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateAsset(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename callout asset", err.Error())
			return
		}
	}
	view, err := r.client.GetCalloutAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back callout asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyCalloutAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *calloutAssetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Cannot delete googleads_callout_asset",
		"The Google Ads API does not support deleting assets. Unlink any campaign/ad_group attachments, remove from HCL, then run `terraform state rm`.",
	)
}

func (r *calloutAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyCalloutAssetView(m *calloutAssetModel, v *googleads.CalloutAssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.CalloutText = types.StringValue(v.CalloutText)
}
