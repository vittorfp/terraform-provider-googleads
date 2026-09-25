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
	_ resource.Resource                = (*sharedSetResource)(nil)
	_ resource.ResourceWithConfigure   = (*sharedSetResource)(nil)
	_ resource.ResourceWithImportState = (*sharedSetResource)(nil)
)

func NewSharedSetResource() resource.Resource { return &sharedSetResource{} }

type sharedSetResource struct {
	client *googleads.Client
}

type sharedSetModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Status     types.String `tfsdk:"status"`
}

func (r *sharedSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_set"
}

func (r *sharedSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A shared set — a reusable bundle of negative keywords (or other criteria) that can be attached to many campaigns via `googleads_campaign_shared_set`. `type` is immutable; only `name` can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/sharedSets/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Shared set name. Must be unique within the account among active shared sets of the same type.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "NEGATIVE_KEYWORDS, NEGATIVE_PLACEMENTS, etc. Immutable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:      true,
				Description:   "Output-only status (ENABLED, REMOVED).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *sharedSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *sharedSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sharedSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateSharedSet(ctx, googleads.SharedSetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		Type:       plan.Type.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create shared set", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetSharedSet(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back shared set", err.Error())
		return
	}
	applySharedSetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sharedSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetSharedSet(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read shared set", err.Error())
		return
	}
	applySharedSetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sharedSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state sharedSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateSharedSet(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename shared set", err.Error())
			return
		}
	}
	view, err := r.client.GetSharedSet(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back shared set", err.Error())
		return
	}
	plan.ID = state.ID
	applySharedSetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sharedSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveSharedSet(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove shared set", err.Error())
	}
}

func (r *sharedSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applySharedSetView(m *sharedSetModel, v *googleads.SharedSetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "sharedSets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Type = types.StringValue(v.Type)
	m.Status = types.StringValue(v.Status)
}
