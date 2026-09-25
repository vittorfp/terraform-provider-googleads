package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*adGroupAdResource)(nil)
	_ resource.ResourceWithConfigure      = (*adGroupAdResource)(nil)
	_ resource.ResourceWithImportState    = (*adGroupAdResource)(nil)
	_ resource.ResourceWithValidateConfig = (*adGroupAdResource)(nil)
)

func NewAdGroupAdResource() resource.Resource { return &adGroupAdResource{} }

type adGroupAdResource struct {
	client *googleads.Client
}

type adGroupAdModel struct {
	ID            types.String `tfsdk:"id"`
	CustomerID    types.String `tfsdk:"customer_id"`
	AdGroupID     types.String `tfsdk:"ad_group_id"`
	Status        types.String `tfsdk:"status"`
	FinalURLs     types.List   `tfsdk:"final_urls"`
	Headlines     types.List   `tfsdk:"headlines"`
	Descriptions  types.List   `tfsdk:"descriptions"`
	Path1         types.String `tfsdk:"path1"`
	Path2         types.String `tfsdk:"path2"`
	RemovalPolicy types.String `tfsdk:"removal_policy"`
}

func (r *adGroupAdResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_ad"
}

func (r *adGroupAdResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Responsive Search Ad under an ad group. Per the Ads API, the ad body (headlines, descriptions, URLs, paths) is immutable — Terraform replaces the resource on changes. Only status is mutable in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Ad group ad resource name (customers/{cid}/adGroupAds/{ag}~{ad}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ad_group_id": schema.StringAttribute{
				Required: true, Description: "Ad group resource name.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"final_urls": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "At least one landing URL.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"headlines": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "3–15 headline strings, each ≤30 characters.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(3),
					listvalidator.SizeAtMost(15),
					listvalidator.ValueStringsAre(runesLengthAtMost(30)),
				},
			},
			"descriptions": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "2–4 description strings, each ≤90 characters.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(2),
					listvalidator.SizeAtMost(4),
					listvalidator.ValueStringsAre(runesLengthAtMost(90)),
				},
			},
			"path1": schema.StringAttribute{
				Optional:    true,
				Description: "Optional display URL path, ≤15 characters.",
				Validators:  []validator.String{runesLengthAtMost(15)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path2": schema.StringAttribute{
				Optional:    true,
				Description: "Optional second display URL path, ≤15 characters. Only valid when path1 is set.",
				Validators:  []validator.String{runesLengthAtMost(15)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"removal_policy": removalPolicyAttribute("ad"),
		},
	}
}

func (r *adGroupAdResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *adGroupAdResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupAdModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.AdGroupAdInput{
		CustomerID: plan.CustomerID.ValueString(),
		AdGroup:    plan.AdGroupID.ValueString(),
		Status:     valueOrDefault(plan.Status, "ENABLED"),
		Path1:      plan.Path1.ValueString(),
		Path2:      plan.Path2.ValueString(),
	}
	resp.Diagnostics.Append(plan.FinalURLs.ElementsAs(ctx, &in.FinalURLs, false)...)
	resp.Diagnostics.Append(plan.Headlines.ElementsAs(ctx, &in.Headlines, false)...)
	resp.Diagnostics.Append(plan.Descriptions.ElementsAs(ctx, &in.Descriptions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateAdGroupAd(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create ad group ad", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAdGroupAd(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group ad", err.Error())
		return
	}
	resp.Diagnostics.Append(applyAdGroupAdView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAdResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupAdModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAdGroupAd(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read ad group ad", err.Error())
		return
	}
	resp.Diagnostics.Append(applyAdGroupAdView(ctx, &state, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupAdResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adGroupAdModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Status.Equal(state.Status) {
		in := googleads.AdGroupAdInput{Status: plan.Status.ValueString()}
		if err := r.client.UpdateAdGroupAd(ctx, state.ID.ValueString(), in, []string{"status"}); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update ad group ad", nil, err)
			return
		}
	}
	view, err := r.client.GetAdGroupAd(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back ad group ad", err.Error())
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(applyAdGroupAdView(ctx, &plan, view)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAdResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupAdModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !enforceRemovalPolicy(&resp.Diagnostics, state.RemovalPolicy, "ad") {
		return
	}
	if err := r.client.RemoveAdGroupAd(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove ad group ad", err.Error())
	}
}

func (r *adGroupAdResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *adGroupAdResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg adGroupAdModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"ad_group_id": cfg.AdGroupID,
	})
}

func applyAdGroupAdView(ctx context.Context, m *adGroupAdModel, v *googleads.AdGroupAdView) diag.Diagnostics {
	var diags diag.Diagnostics
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "adGroupAds"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AdGroupID = types.StringValue(v.AdGroup)
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
	heads, d := types.ListValueFrom(ctx, types.StringType, v.Headlines)
	diags.Append(d...)
	m.Headlines = heads
	descs, d := types.ListValueFrom(ctx, types.StringType, v.Descriptions)
	diags.Append(d...)
	m.Descriptions = descs
	return diags
}
