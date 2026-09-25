package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*structuredSnippetAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*structuredSnippetAssetResource)(nil)
	_ resource.ResourceWithImportState = (*structuredSnippetAssetResource)(nil)
)

func NewStructuredSnippetAssetResource() resource.Resource {
	return &structuredSnippetAssetResource{}
}

type structuredSnippetAssetResource struct {
	client *googleads.Client
}

type structuredSnippetAssetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Name       types.String `tfsdk:"name"`
	Header     types.String `tfsdk:"header"`
	Values     types.List   `tfsdk:"values"`
}

func (r *structuredSnippetAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_structured_snippet_asset"
}

func (r *structuredSnippetAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A structured snippet asset — a header from the Ads predefined list (\"Models\", \"Services\", \"Brands\", etc.) followed by 3–10 short values. Asset content is immutable; only the wrapper's `name` can be renamed. **The API has no remove operation** — to stop managing, unlink, remove from HCL, then `terraform state rm`.",
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
			"header": schema.StringAttribute{
				Required:    true,
				Description: "Snippet header. Must be one of Google's predefined values (Models, Services, Brands, Featured hotels, etc.). See https://developers.google.com/google-ads/api/reference/data/structured-snippet-headers. Immutable.",
				PlanModifiers: immutable,
			},
			"values": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "3–10 values, each 1–25 characters. Immutable.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(3),
					listvalidator.SizeAtMost(10),
				},
			},
		},
	}
}

func (r *structuredSnippetAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *structuredSnippetAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan structuredSnippetAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.StructuredSnippetAssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		Header:     plan.Header.ValueString(),
	}
	resp.Diagnostics.Append(plan.Values.ElementsAs(ctx, &in.Values, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateStructuredSnippetAsset(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create structured snippet asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetStructuredSnippetAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back structured snippet asset", err.Error())
		return
	}
	resp.Diagnostics.Append(applyStructuredSnippetAssetView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *structuredSnippetAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state structuredSnippetAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetStructuredSnippetAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read structured snippet asset", err.Error())
		return
	}
	resp.Diagnostics.Append(applyStructuredSnippetAssetView(ctx, &state, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *structuredSnippetAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state structuredSnippetAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateAsset(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename structured snippet asset", err.Error())
			return
		}
	}
	view, err := r.client.GetStructuredSnippetAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back structured snippet asset", err.Error())
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(applyStructuredSnippetAssetView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *structuredSnippetAssetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Cannot delete googleads_structured_snippet_asset",
		"The Google Ads API does not support deleting assets. Unlink any campaign/ad_group attachments, remove from HCL, then run `terraform state rm`.",
	)
}

func (r *structuredSnippetAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyStructuredSnippetAssetView(ctx context.Context, m *structuredSnippetAssetModel, v *googleads.StructuredSnippetAssetView) diag.Diagnostics {
	var diags diag.Diagnostics
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Header = types.StringValue(v.Header)
	vals, d := types.ListValueFrom(ctx, types.StringType, v.Values)
	diags.Append(d...)
	m.Values = vals
	return diags
}
