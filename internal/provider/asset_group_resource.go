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
	_ resource.Resource                   = (*assetGroupResource)(nil)
	_ resource.ResourceWithConfigure      = (*assetGroupResource)(nil)
	_ resource.ResourceWithImportState    = (*assetGroupResource)(nil)
	_ resource.ResourceWithValidateConfig = (*assetGroupResource)(nil)
)

func NewAssetGroupResource() resource.Resource { return &assetGroupResource{} }

type assetGroupResource struct {
	client *googleads.Client
}

type assetGroupModel struct {
	ID              types.String `tfsdk:"id"`
	CustomerID      types.String `tfsdk:"customer_id"`
	CampaignID      types.String `tfsdk:"campaign_id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	FinalURLs       types.List   `tfsdk:"final_urls"`
	FinalMobileURLs types.List   `tfsdk:"final_mobile_urls"`
	Path1           types.String `tfsdk:"path1"`
	Path2           types.String `tfsdk:"path2"`
}

func (r *assetGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_asset_group"
}

func (r *assetGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Performance Max asset group — the bundle of headlines, descriptions, images, logos, and videos (linked via `googleads_asset_group_asset`) that fuels a PMax campaign's automated creative assembly. The parent campaign is immutable; name, status, URLs, and paths can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Asset group resource name (customers/{cid}/assetGroups/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"campaign_id": schema.StringAttribute{
				Required: true, Description: "Parent PMax campaign resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Asset group name (1–128 characters, unique within the campaign).",
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"final_urls": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Landing URLs. At least one required.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"final_mobile_urls": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Optional mobile-specific landing URLs.",
			},
			"path1": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "First part of the display URL path.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"path2": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Second part of the display URL path. Only valid when path1 is set.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *assetGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *assetGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assetGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.AssetGroupInput{
		CustomerID: plan.CustomerID.ValueString(),
		Campaign:   plan.CampaignID.ValueString(),
		Name:       plan.Name.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
		Path1:      plan.Path1.ValueString(),
		Path2:      plan.Path2.ValueString(),
	}
	resp.Diagnostics.Append(plan.FinalURLs.ElementsAs(ctx, &in.FinalURLs, false)...)
	if !plan.FinalMobileURLs.IsNull() && !plan.FinalMobileURLs.IsUnknown() {
		resp.Diagnostics.Append(plan.FinalMobileURLs.ElementsAs(ctx, &in.FinalMobileURLs, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateAssetGroup(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create asset group", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAssetGroup(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back asset group", err.Error())
		return
	}
	resp.Diagnostics.Append(applyAssetGroupView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assetGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAssetGroup(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read asset group", err.Error())
		return
	}
	resp.Diagnostics.Append(applyAssetGroupView(ctx, &state, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *assetGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state assetGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	in := googleads.AssetGroupInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		Status:     plan.Status.ValueString(),
		Path1:      plan.Path1.ValueString(),
		Path2:      plan.Path2.ValueString(),
	}
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.FinalURLs.Equal(state.FinalURLs) {
		paths = append(paths, "final_urls")
		resp.Diagnostics.Append(plan.FinalURLs.ElementsAs(ctx, &in.FinalURLs, false)...)
	}
	if !plan.FinalMobileURLs.Equal(state.FinalMobileURLs) {
		paths = append(paths, "final_mobile_urls")
		if !plan.FinalMobileURLs.IsNull() && !plan.FinalMobileURLs.IsUnknown() {
			resp.Diagnostics.Append(plan.FinalMobileURLs.ElementsAs(ctx, &in.FinalMobileURLs, false)...)
		}
	}
	if !plan.Path1.Equal(state.Path1) {
		paths = append(paths, "path1")
	}
	if !plan.Path2.Equal(state.Path2) {
		paths = append(paths, "path2")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(paths) > 0 {
		if err := r.client.UpdateAssetGroup(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update asset group", nil, err)
			return
		}
	}
	view, err := r.client.GetAssetGroup(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back asset group", err.Error())
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(applyAssetGroupView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assetGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveAssetGroup(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove asset group", err.Error())
	}
}

func (r *assetGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *assetGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg assetGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"campaign_id": cfg.CampaignID,
	})
}

func applyAssetGroupView(ctx context.Context, m *assetGroupModel, v *googleads.AssetGroupView) diag.Diagnostics {
	var diags diag.Diagnostics
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assetGroups"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.CampaignID = types.StringValue(v.Campaign)
	m.Name = types.StringValue(v.Name)
	m.Status = types.StringValue(v.Status)
	if v.Path1 == "" {
		m.Path1 = types.StringNull()
	} else {
		m.Path1 = types.StringValue(v.Path1)
	}
	if v.Path2 == "" {
		m.Path2 = types.StringNull()
	} else {
		m.Path2 = types.StringValue(v.Path2)
	}
	finals, d := types.ListValueFrom(ctx, types.StringType, v.FinalURLs)
	diags.Append(d...)
	m.FinalURLs = finals
	if len(v.FinalMobileURLs) == 0 {
		m.FinalMobileURLs = types.ListNull(types.StringType)
	} else {
		mobiles, d := types.ListValueFrom(ctx, types.StringType, v.FinalMobileURLs)
		diags.Append(d...)
		m.FinalMobileURLs = mobiles
	}
	return diags
}
