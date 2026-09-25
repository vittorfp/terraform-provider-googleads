package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                = (*sitelinkAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*sitelinkAssetResource)(nil)
	_ resource.ResourceWithImportState = (*sitelinkAssetResource)(nil)
)

func NewSitelinkAssetResource() resource.Resource { return &sitelinkAssetResource{} }

type sitelinkAssetResource struct {
	client *googleads.Client
}

type sitelinkAssetModel struct {
	ID           types.String `tfsdk:"id"`
	CustomerID   types.String `tfsdk:"customer_id"`
	Name         types.String `tfsdk:"name"`
	LinkText     types.String `tfsdk:"link_text"`
	Description1 types.String `tfsdk:"description1"`
	Description2 types.String `tfsdk:"description2"`
	FinalURLs    types.List   `tfsdk:"final_urls"`
}

func (r *sitelinkAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sitelink_asset"
}

func (r *sitelinkAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A sitelink asset — short clickable text under a search ad pointing to a deeper page. Asset content is immutable per the API; the wrapper's `name` can be renamed in place. **The API has no remove operation** — to stop managing this asset, unlink any campaign/ad_group attachments, remove from HCL, then `terraform state rm`.",
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
				PlanModifiers: immutable,
			},
			"name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Optional display name. The API auto-generates one if omitted.",
			},
			"link_text": schema.StringAttribute{
				Required:    true,
				Description: "Display text (1–25 characters). Immutable.",
				PlanModifiers: immutable,
			},
			"description1": schema.StringAttribute{
				Optional:    true,
				Description: "First description line (1–35 characters). If set, description2 must also be set. Immutable.",
				PlanModifiers: immutable,
			},
			"description2": schema.StringAttribute{
				Optional:    true,
				Description: "Second description line (1–35 characters). If set, description1 must also be set. Immutable.",
				PlanModifiers: immutable,
			},
			"final_urls": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Landing URLs the sitelink points to. Immutable.",
				PlanModifiers: []planmodifier.List{
					// existing listplanmodifier import is already present in the package via ad_group_ad_resource
				},
			},
		},
	}
}

func (r *sitelinkAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *sitelinkAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sitelinkAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.SitelinkAssetInput{
		CustomerID:   plan.CustomerID.ValueString(),
		Name:         plan.Name.ValueString(),
		LinkText:     plan.LinkText.ValueString(),
		Description1: plan.Description1.ValueString(),
		Description2: plan.Description2.ValueString(),
	}
	resp.Diagnostics.Append(plan.FinalURLs.ElementsAs(ctx, &in.FinalURLs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateSitelinkAsset(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create sitelink asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetSitelinkAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back sitelink asset", err.Error())
		return
	}
	resp.Diagnostics.Append(applySitelinkAssetView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sitelinkAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sitelinkAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetSitelinkAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read sitelink asset", err.Error())
		return
	}
	resp.Diagnostics.Append(applySitelinkAssetView(ctx, &state, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sitelinkAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state sitelinkAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateAsset(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename sitelink asset", err.Error())
			return
		}
	}
	view, err := r.client.GetSitelinkAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back sitelink asset", err.Error())
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(applySitelinkAssetView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sitelinkAssetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Cannot delete googleads_sitelink_asset",
		"The Google Ads API does not support deleting assets. Unlink any campaign/ad_group attachments, remove from HCL, then run `terraform state rm` to drop from state.",
	)
}

func (r *sitelinkAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applySitelinkAssetView(ctx context.Context, m *sitelinkAssetModel, v *googleads.SitelinkAssetView) diag.Diagnostics {
	var diags diag.Diagnostics
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.LinkText = types.StringValue(v.LinkText)
	m.Description1 = nullableString(v.Description1)
	m.Description2 = nullableString(v.Description2)
	finals, d := types.ListValueFrom(ctx, types.StringType, v.FinalURLs)
	diags.Append(d...)
	m.FinalURLs = finals
	return diags
}
