package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*adGroupCriterionResource)(nil)
	_ resource.ResourceWithConfigure      = (*adGroupCriterionResource)(nil)
	_ resource.ResourceWithImportState    = (*adGroupCriterionResource)(nil)
	_ resource.ResourceWithValidateConfig = (*adGroupCriterionResource)(nil)
)

func NewAdGroupCriterionResource() resource.Resource { return &adGroupCriterionResource{} }

type adGroupCriterionResource struct {
	client *googleads.Client
}

type adGroupCriterionModel struct {
	ID           types.String `tfsdk:"id"`
	CustomerID   types.String `tfsdk:"customer_id"`
	AdGroupID    types.String `tfsdk:"ad_group_id"`
	KeywordText  types.String `tfsdk:"keyword_text"`
	MatchType    types.String `tfsdk:"match_type"`
	Status       types.String `tfsdk:"status"`
	Negative     types.Bool   `tfsdk:"negative"`
	CpcBidMicros types.Int64  `tfsdk:"cpc_bid_micros"`
}

func (r *adGroupCriterionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_criterion"
}

func (r *adGroupCriterionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A keyword targeting (or excluding) criterion attached to an ad group. v1 only covers the keyword criterion type — placements, audiences, etc. are out of scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/adGroupCriteria/{ag}~{crit}).",
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
			"keyword_text": schema.StringAttribute{
				Required:    true,
				Description: "Keyword text (≤80 characters, ≤10 words). Immutable.",
				Validators:  []validator.String{runesLengthAtMost(80)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_type": schema.StringAttribute{
				Required:    true,
				Description: "EXACT, PHRASE, or BROAD. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf("EXACT", "PHRASE", "BROAD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "ENABLED (default), PAUSED, or REMOVED.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"negative": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the keyword is excluded rather than targeted. Immutable.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"cpc_bid_micros": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Optional max CPC bid in micros. Ignored under non-manual-CPC bidding strategies.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *adGroupCriterionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *adGroupCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := googleads.CriterionInput{
		CustomerID:  plan.CustomerID.ValueString(),
		AdGroup:     plan.AdGroupID.ValueString(),
		KeywordText: plan.KeywordText.ValueString(),
		MatchType:   plan.MatchType.ValueString(),
		Status:      valueOrDefault(plan.Status, "ENABLED"),
	}
	if !plan.Negative.IsNull() && !plan.Negative.IsUnknown() {
		v := plan.Negative.ValueBool()
		in.Negative = &v
	}
	if !plan.CpcBidMicros.IsNull() && !plan.CpcBidMicros.IsUnknown() {
		v := plan.CpcBidMicros.ValueInt64()
		in.CpcBidMicros = &v
	}
	rn, err := r.client.CreateCriterion(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create criterion", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCriterion(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back criterion", err.Error())
		return
	}
	applyCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCriterion(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read criterion", err.Error())
		return
	}
	applyCriterionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adGroupCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	in := googleads.CriterionInput{
		CustomerID: plan.CustomerID.ValueString(),
		Status:     plan.Status.ValueString(),
	}
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.CpcBidMicros.Equal(state.CpcBidMicros) {
		paths = append(paths, "cpc_bid_micros")
		if !plan.CpcBidMicros.IsNull() && !plan.CpcBidMicros.IsUnknown() {
			v := plan.CpcBidMicros.ValueInt64()
			in.CpcBidMicros = &v
		}
	}
	if len(paths) > 0 {
		if err := r.client.UpdateCriterion(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update criterion", nil, err)
			return
		}
	}
	view, err := r.client.GetCriterion(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back criterion", err.Error())
		return
	}
	plan.ID = state.ID
	applyCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCriterion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove criterion", err.Error())
	}
}

func (r *adGroupCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *adGroupCriterionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg adGroupCriterionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"ad_group_id": cfg.AdGroupID,
	})
}

func applyCriterionView(m *adGroupCriterionModel, v *googleads.CriterionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "adGroupCriteria"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AdGroupID = types.StringValue(v.AdGroup)
	m.KeywordText = types.StringValue(v.KeywordText)
	m.MatchType = types.StringValue(v.MatchType)
	m.Status = types.StringValue(v.Status)
	m.Negative = types.BoolValue(v.Negative)
	m.CpcBidMicros = types.Int64Value(v.CpcBidMicros)
}
