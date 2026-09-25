package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                = (*campaignBudgetResource)(nil)
	_ resource.ResourceWithConfigure   = (*campaignBudgetResource)(nil)
	_ resource.ResourceWithImportState = (*campaignBudgetResource)(nil)
)

func NewCampaignBudgetResource() resource.Resource { return &campaignBudgetResource{} }

type campaignBudgetResource struct {
	client *googleads.Client
}

type campaignBudgetModel struct {
	ID               types.String `tfsdk:"id"`
	CustomerID       types.String `tfsdk:"customer_id"`
	Name             types.String `tfsdk:"name"`
	AmountMicros     types.Int64  `tfsdk:"amount_micros"`
	DeliveryMethod   types.String `tfsdk:"delivery_method"`
	ExplicitlyShared types.Bool   `tfsdk:"explicitly_shared"`
	Status           types.String `tfsdk:"status"`
	RemovalPolicy    types.String `tfsdk:"removal_policy"`
}

func (r *campaignBudgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_budget"
}

func (r *campaignBudgetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A daily spend cap for one or more campaigns. Budgets must exist before any campaign can reference them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The resource name of the campaign budget (customers/{cid}/campaignBudgets/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required:    true,
				Description: "Google Ads customer ID (digits only).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Budget name.",
			},
			"amount_micros": schema.Int64Attribute{
				Required:    true,
				Description: "Average daily spend cap in micros of the account's currency (1,000,000 micros = 1 currency unit).",
			},
			"delivery_method": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "STANDARD (default) or ACCELERATED.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"explicitly_shared": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, this budget can back multiple campaigns. Cannot be flipped from true back to false.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Output-only budget status (ENABLED, REMOVED).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"removal_policy": removalPolicyAttribute("campaign budget"),
		},
	}
}

func (r *campaignBudgetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *campaignBudgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan campaignBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := googleads.BudgetInput{
		CustomerID:     plan.CustomerID.ValueString(),
		Name:           plan.Name.ValueString(),
		AmountMicros:   plan.AmountMicros.ValueInt64(),
		DeliveryMethod: plan.DeliveryMethod.ValueString(),
	}
	if !plan.ExplicitlyShared.IsNull() && !plan.ExplicitlyShared.IsUnknown() {
		v := plan.ExplicitlyShared.ValueBool()
		in.ExplicitlyShared = &v
	}

	resourceName, err := r.client.CreateBudget(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create campaign budget", budgetAttrMap(), err)
		return
	}
	plan.ID = types.StringValue(resourceName)

	view, err := r.client.GetBudget(ctx, resourceName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign budget", err.Error())
		return
	}
	applyBudgetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignBudgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state campaignBudgetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetBudget(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read campaign budget", err.Error())
		return
	}
	applyBudgetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *campaignBudgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state campaignBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var paths []string
	in := googleads.BudgetInput{
		CustomerID:     plan.CustomerID.ValueString(),
		Name:           plan.Name.ValueString(),
		AmountMicros:   plan.AmountMicros.ValueInt64(),
		DeliveryMethod: plan.DeliveryMethod.ValueString(),
	}
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.AmountMicros.Equal(state.AmountMicros) {
		paths = append(paths, "amount_micros")
	}
	if !plan.DeliveryMethod.Equal(state.DeliveryMethod) {
		paths = append(paths, "delivery_method")
	}
	if !plan.ExplicitlyShared.Equal(state.ExplicitlyShared) && !plan.ExplicitlyShared.IsNull() && !plan.ExplicitlyShared.IsUnknown() {
		v := plan.ExplicitlyShared.ValueBool()
		in.ExplicitlyShared = &v
		paths = append(paths, "explicitly_shared")
	}

	if len(paths) > 0 {
		if err := r.client.UpdateBudget(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update campaign budget", budgetAttrMap(), err)
			return
		}
	}

	view, err := r.client.GetBudget(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back campaign budget", err.Error())
		return
	}
	plan.ID = state.ID
	applyBudgetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *campaignBudgetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state campaignBudgetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !enforceRemovalPolicy(&resp.Diagnostics, state.RemovalPolicy, "campaign budget") {
		return
	}
	if err := r.client.RemoveBudget(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove campaign budget", err.Error())
	}
}

func (r *campaignBudgetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyBudgetView(m *campaignBudgetModel, v *googleads.BudgetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "campaignBudgets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.AmountMicros = types.Int64Value(v.AmountMicros)
	m.DeliveryMethod = types.StringValue(v.DeliveryMethod)
	m.ExplicitlyShared = types.BoolValue(v.ExplicitlyShared)
	m.Status = types.StringValue(v.Status)
}

// budgetAttrMap maps the Ads API's per-field error paths' leaf segments
// to the corresponding HCL attribute on this resource. Used by
// addAPIErrorDiagnostics so plan/apply errors land on the right line in
// the user's HCL.
func budgetAttrMap() map[string]path.Path {
	return map[string]path.Path{
		"name":              path.Root("name"),
		"amount_micros":     path.Root("amount_micros"),
		"delivery_method":   path.Root("delivery_method"),
		"explicitly_shared": path.Root("explicitly_shared"),
	}
}
