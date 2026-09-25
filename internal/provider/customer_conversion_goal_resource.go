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
	_ resource.Resource                = (*customerConversionGoalResource)(nil)
	_ resource.ResourceWithConfigure   = (*customerConversionGoalResource)(nil)
	_ resource.ResourceWithImportState = (*customerConversionGoalResource)(nil)
)

func NewCustomerConversionGoalResource() resource.Resource {
	return &customerConversionGoalResource{}
}

type customerConversionGoalResource struct {
	client *googleads.Client
}

type customerConversionGoalModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Category   types.String `tfsdk:"category"`
	Origin     types.String `tfsdk:"origin"`
	Biddable   types.Bool   `tfsdk:"biddable"`
}

func (r *customerConversionGoalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer_conversion_goal"
}

func (r *customerConversionGoalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Customer-level conversion goal — the bridge between conversion actions and Smart Bidding strategies. Performance Max and other Smart Bidding strategies optimize toward the `biddable=true` (category, origin) pairs. " +
			"Every (category, origin) pair always exists in every account — the resource doesn't create or delete; it just flips `biddable`. Deleting the resource resets `biddable` to `false` (which is the API's default), undoing the apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/customerConversionGoals/{category}~{origin}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"category": schema.StringAttribute{
				Required:    true,
				Description: "Conversion category — DEFAULT, PAGE_VIEW, PURCHASE, SIGNUP, DOWNLOAD, ADD_TO_CART, BEGIN_CHECKOUT, SUBSCRIBE_PAID, etc. Immutable (part of the resource identity).",
				PlanModifiers: immutable,
			},
			"origin": schema.StringAttribute{
				Required: true,
				Description: "Conversion origin — WEBSITE, APP, STORE, CALL_FROM_ADS, GOOGLE_HOSTED, YOUTUBE_HOSTED. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf("WEBSITE", "APP", "STORE", "CALL_FROM_ADS", "GOOGLE_HOSTED", "YOUTUBE_HOSTED"),
				},
				PlanModifiers: immutable,
			},
			"biddable": schema.BoolAttribute{
				Required:    true,
				Description: "When true, Smart Bidding strategies (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS, MAXIMIZE_CONVERSION_VALUE) optimize toward conversions in this (category, origin) bucket.",
			},
		},
	}
}

func (r *customerConversionGoalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *customerConversionGoalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customerConversionGoalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.SetCustomerConversionGoalBiddable(ctx, googleads.CustomerConversionGoalInput{
		CustomerID: plan.CustomerID.ValueString(),
		Category:   plan.Category.ValueString(),
		Origin:     plan.Origin.ValueString(),
		Biddable:   plan.Biddable.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set customer conversion goal", err.Error())
		return
	}
	plan.ID = types.StringValue(rn)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customerConversionGoalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customerConversionGoalModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCustomerConversionGoal(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read customer conversion goal", err.Error())
		return
	}
	if cid, _, err := googleads.ParseResourceName(view.ResourceName, "customerConversionGoals"); err == nil {
		state.CustomerID = types.StringValue(cid)
	}
	state.Category = types.StringValue(view.Category)
	state.Origin = types.StringValue(view.Origin)
	state.Biddable = types.BoolValue(view.Biddable)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customerConversionGoalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customerConversionGoalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SetCustomerConversionGoalBiddable(ctx, googleads.CustomerConversionGoalInput{
		CustomerID: plan.CustomerID.ValueString(),
		Category:   plan.Category.ValueString(),
		Origin:     plan.Origin.ValueString(),
		Biddable:   plan.Biddable.ValueBool(),
	}); err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update customer conversion goal", nil, err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete resets biddable to false — the API's default and the closest
// thing to "remove" since these (category, origin) pairs always exist
// in the account. Leaves no API-side trace of having been managed by
// Terraform.
func (r *customerConversionGoalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customerConversionGoalModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SetCustomerConversionGoalBiddable(ctx, googleads.CustomerConversionGoalInput{
		CustomerID: state.CustomerID.ValueString(),
		Category:   state.Category.ValueString(),
		Origin:     state.Origin.ValueString(),
		Biddable:   false,
	}); err != nil {
		resp.Diagnostics.AddError("Failed to reset customer conversion goal biddable to false", err.Error())
	}
}

func (r *customerConversionGoalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
