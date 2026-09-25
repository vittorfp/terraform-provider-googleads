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
	_ resource.Resource                = (*labelResource)(nil)
	_ resource.ResourceWithConfigure   = (*labelResource)(nil)
	_ resource.ResourceWithImportState = (*labelResource)(nil)
)

func NewLabelResource() resource.Resource { return &labelResource{} }

type labelResource struct {
	client *googleads.Client
}

type labelModel struct {
	ID              types.String `tfsdk:"id"`
	CustomerID      types.String `tfsdk:"customer_id"`
	Name            types.String `tfsdk:"name"`
	BackgroundColor types.String `tfsdk:"background_color"`
	Description     types.String `tfsdk:"description"`
	Status          types.String `tfsdk:"status"`
}

func (r *labelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label"
}

func (r *labelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A label — a reusable annotation that can be attached to campaigns, ad groups, ads, and keyword criteria via the four `googleads_*_label` link resources. Useful for grouping resources in reports.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/labels/{id}).",
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
				Description: "Label name (1–80 characters). Must be unique within the account.",
			},
			"background_color": schema.StringAttribute{
				Optional:    true,
				Description: "Display color in HEX, e.g. \"#FFAABB\". Not visible on manager accounts.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Short description, ≤200 characters.",
			},
			"status": schema.StringAttribute{
				Computed:      true,
				Description:   "Output-only label status.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *labelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *labelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateLabel(ctx, googleads.LabelInput{
		CustomerID:      plan.CustomerID.ValueString(),
		Name:            plan.Name.ValueString(),
		BackgroundColor: plan.BackgroundColor.ValueString(),
		Description:     plan.Description.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create label", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetLabel(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back label", err.Error())
		return
	}
	applyLabelView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetLabel(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read label", err.Error())
		return
	}
	applyLabelView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *labelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.BackgroundColor.Equal(state.BackgroundColor) {
		paths = append(paths, "text_label.background_color")
	}
	if !plan.Description.Equal(state.Description) {
		paths = append(paths, "text_label.description")
	}
	if len(paths) > 0 {
		if err := r.client.UpdateLabel(ctx, state.ID.ValueString(), googleads.LabelInput{
			Name:            plan.Name.ValueString(),
			BackgroundColor: plan.BackgroundColor.ValueString(),
			Description:     plan.Description.ValueString(),
		}, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update label", nil, err)
			return
		}
	}
	view, err := r.client.GetLabel(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back label", err.Error())
		return
	}
	plan.ID = state.ID
	applyLabelView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveLabel(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove label", err.Error())
	}
}

func (r *labelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyLabelView(m *labelModel, v *googleads.LabelView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "labels"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Status = types.StringValue(v.Status)
	m.BackgroundColor = nullableString(v.BackgroundColor)
	m.Description = nullableString(v.Description)
}
