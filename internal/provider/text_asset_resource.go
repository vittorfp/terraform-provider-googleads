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
	_ resource.Resource                = (*textAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*textAssetResource)(nil)
	_ resource.ResourceWithImportState = (*textAssetResource)(nil)
)

func NewTextAssetResource() resource.Resource { return &textAssetResource{} }

type textAssetResource struct {
	client *googleads.Client
}

type textAssetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Name       types.String `tfsdk:"name"`
	Text       types.String `tfsdk:"text"`
}

func (r *textAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_text_asset"
}

func (r *textAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A text asset — a reusable string that backs headlines, descriptions, business names, etc. on Performance Max and Responsive Search Ads. Per the Ads API the underlying text is immutable; changing it forces resource replacement. **The API has no remove operation** — Terraform refuses to delete this resource. To stop managing it, unlink any references, remove the block from HCL, and run `terraform state rm`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Asset resource name (customers/{cid}/assets/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Optional display name. The API auto-generates one if omitted.",
			},
			"text": schema.StringAttribute{
				Required:    true,
				Description: "The text content. Immutable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *textAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *textAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan textAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateTextAsset(ctx, googleads.AssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		Text:       plan.Text.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create text asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back text asset", err.Error())
		return
	}
	applyTextAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *textAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state textAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read text asset", err.Error())
		return
	}
	applyTextAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *textAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state textAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateAsset(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename text asset", err.Error())
			return
		}
	}
	view, err := r.client.GetAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back text asset", err.Error())
		return
	}
	plan.ID = state.ID
	applyTextAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete errors out. The Google Ads API exposes no AssetService.Remove —
// assets can only be orphaned in the UI, not deleted. Silently dropping
// from state would be surprising (the asset keeps living in the account),
// so we force the user to explicitly opt in via `terraform state rm`.
func (r *textAssetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Cannot delete googleads_text_asset",
		"The Google Ads API does not support deleting assets. To stop managing this asset with Terraform: "+
			"(1) unlink it from any ad groups / asset groups that reference it, "+
			"(2) remove it from your HCL, "+
			"(3) run `terraform state rm <address>` to drop it from state. "+
			"The asset itself will remain in the Google Ads account (orphaned).",
	)
}

func (r *textAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyTextAssetView(m *textAssetModel, v *googleads.AssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Text = types.StringValue(v.Text)
}
